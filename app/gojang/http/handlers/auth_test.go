package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gojangframework/gojang/app/gojang/models"
	"github.com/gojangframework/gojang/app/gojang/models/enttest"
	"github.com/gojangframework/gojang/app/gojang/models/user"
	"github.com/gojangframework/gojang/app/gojang/utils"
	"github.com/gojangframework/gojang/app/gojang/views/renderers"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type stubAuthEmailSender struct {
	htmlCalls      int
	resetCalls     int
	lastHTMLTo     []string
	lastSubject    string
	lastHTMLBody   string
	lastResetTo    string
	lastResetURL   string
	returnSendErr  error
	returnResetErr error
}

type stubRecaptchaVerifier struct {
	siteKey      string
	calls        int
	lastToken    string
	lastAction   string
	lastRemoteIP string
	requireToken bool
	returnErr    error
}

func (s *stubAuthEmailSender) SendHTMLEmail(to []string, subject, htmlBody string) error {
	s.htmlCalls++
	s.lastHTMLTo = append([]string(nil), to...)
	s.lastSubject = subject
	s.lastHTMLBody = htmlBody
	return s.returnSendErr
}

func (s *stubAuthEmailSender) SendPasswordResetEmail(to, resetURL string) error {
	s.resetCalls++
	s.lastResetTo = to
	s.lastResetURL = resetURL
	return s.returnResetErr
}

func (s *stubRecaptchaVerifier) SiteKey() string {
	return s.siteKey
}

func (s *stubRecaptchaVerifier) ScriptURL() string {
	if s.siteKey == "" {
		return ""
	}
	return "https://www.google.com/recaptcha/api.js?render=" + s.siteKey
}

func (s *stubRecaptchaVerifier) Verify(ctx context.Context, token, action, remoteIP string) error {
	s.calls++
	s.lastToken = token
	s.lastAction = action
	s.lastRemoteIP = remoteIP
	if s.returnErr != nil {
		return s.returnErr
	}
	if s.requireToken && token == "" {
		return utils.ErrRecaptchaVerificationFailed
	}
	return nil
}

func newAuthHandlerTestEnv(t *testing.T) (*AuthHandler, *models.Client, *scs.SessionManager, *stubAuthEmailSender) {
	t.Helper()

	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", name))
	t.Cleanup(func() { client.Close() })

	sessions := scs.New()
	sessions.Lifetime = time.Hour
	mailer := &stubAuthEmailSender{}
	renderer, err := renderers.NewRenderer(false, renderers.WithSessionManager(sessions))
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}

	handler := NewAuthHandler(client, sessions, renderer, mailer, "https://app.example.com", true, nil)
	return handler, client, sessions, mailer
}

func performAuthRequest(sessions *scs.SessionManager, handler http.HandlerFunc, method, target string, form url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var body *strings.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}

	req := httptest.NewRequest(method, target, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	rec := httptest.NewRecorder()
	sessions.LoadAndSave(handler).ServeHTTP(rec, req)
	return rec
}

func createAuthTestUser(t *testing.T, client *models.Client, email, password string, verified bool) *models.User {
	t.Helper()
	hash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	u, err := client.User.Create().
		SetID(uuid.New()).
		SetEmail(email).
		SetPasswordHash(hash).
		SetIsActive(true).
		SetIsStaff(false).
		SetIsSuperuser(false).
		SetIsEmailVerified(verified).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create user error = %v", err)
	}
	return u
}

func TestRegisterGETRendersRecaptchaHTMXWiring(t *testing.T) {
	handler, _, sessions, _ := newAuthHandlerTestEnv(t)
	handler.Recaptcha = &stubRecaptchaVerifier{siteKey: "site-key"}

	rec := performAuthRequest(sessions, handler.RegisterGET, http.MethodGet, "/register", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`https://www.google.com/recaptcha/api.js?render=site-key`,
		`hx-trigger="recaptcha-verified"`,
		`data-recaptcha-site-key="site-key"`,
		`name="g-recaptcha-response"`,
		`grecaptcha.ready`,
		`requestRecaptchaToken(0)`,
		`window.htmx.trigger(form, 'recaptcha-verified')`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected register body to contain %q, got: %s", want, body)
		}
	}
}

func TestRegisterPOSTRedirectsToVerifyAndSendsEmail(t *testing.T) {
	handler, client, sessions, mailer := newAuthHandlerTestEnv(t)

	rec := performAuthRequest(sessions, handler.RegisterPOST, http.MethodPost, "/register", url.Values{
		"email":            {"new@example.com"},
		"password":         {"Password123!"},
		"password_confirm": {"Password123!"},
	})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/register-verify-email" {
		t.Fatalf("Location = %q, want /register-verify-email", got)
	}
	if mailer.htmlCalls != 1 {
		t.Fatalf("htmlCalls = %d, want 1", mailer.htmlCalls)
	}
	if !strings.Contains(mailer.lastHTMLBody, "https://app.example.com/verify-email?token=") {
		t.Fatalf("verification email missing app base URL: %s", mailer.lastHTMLBody)
	}

	u, err := client.User.Query().Where(user.EmailEQ("new@example.com")).Only(context.Background())
	if err != nil {
		t.Fatalf("query user error = %v", err)
	}
	if u.IsEmailVerified {
		t.Fatal("registered user should start unverified")
	}
	if u.EmailVerificationToken == nil || u.EmailVerificationExpiry == nil {
		t.Fatalf("expected verification token and expiry, got token=%v expiry=%v", u.EmailVerificationToken, u.EmailVerificationExpiry)
	}
}

func TestRegisterPOSTMissingRecaptchaTokenBlocksWhenConfigured(t *testing.T) {
	handler, client, sessions, mailer := newAuthHandlerTestEnv(t)
	recaptcha := &stubRecaptchaVerifier{siteKey: "site-key", requireToken: true}
	handler.Recaptcha = recaptcha

	rec := performAuthRequest(sessions, handler.RegisterPOST, http.MethodPost, "/register", url.Values{
		"email":            {"recaptcha@example.com"},
		"password":         {"Password123!"},
		"password_confirm": {"Password123!"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if recaptcha.calls != 1 {
		t.Fatalf("recaptcha calls = %d, want 1", recaptcha.calls)
	}
	if mailer.htmlCalls != 0 {
		t.Fatalf("htmlCalls = %d, want 0", mailer.htmlCalls)
	}
	exists, err := client.User.Query().Where(user.EmailEQ("recaptcha@example.com")).Exist(context.Background())
	if err != nil {
		t.Fatalf("query user error = %v", err)
	}
	if exists {
		t.Fatal("user should not be created when recaptcha fails")
	}
	if !strings.Contains(rec.Body.String(), "We could not verify this signup") {
		t.Fatalf("expected recaptcha failure message, got: %s", rec.Body.String())
	}
}

func TestRegisterPOSTBrowserErrorShowsActionableRecaptchaMessage(t *testing.T) {
	handler, _, sessions, _ := newAuthHandlerTestEnv(t)
	handler.Recaptcha = &stubRecaptchaVerifier{
		siteKey: "site-key",
		returnErr: &utils.RecaptchaVerificationError{
			Reason:     "google rejected token",
			ErrorCodes: []string{"browser-error"},
		},
	}

	rec := performAuthRequest(sessions, handler.RegisterPOST, http.MethodPost, "/register", url.Values{
		"email":                {"browser-error@example.com"},
		"password":             {"Password123!"},
		"password_confirm":     {"Password123!"},
		"g-recaptcha-response": {"token"},
	})

	if !strings.Contains(rec.Body.String(), "reCAPTCHA could not complete in this browser") {
		t.Fatalf("expected actionable recaptcha message, got: %s", rec.Body.String())
	}
}

func TestRegisterPOSTValidRecaptchaAllowsRegistration(t *testing.T) {
	handler, client, sessions, _ := newAuthHandlerTestEnv(t)
	recaptcha := &stubRecaptchaVerifier{siteKey: "site-key", requireToken: true}
	handler.Recaptcha = recaptcha

	rec := performAuthRequest(sessions, handler.RegisterPOST, http.MethodPost, "/register", url.Values{
		"email":                {"recaptcha-valid@example.com"},
		"password":             {"Password123!"},
		"password_confirm":     {"Password123!"},
		"g-recaptcha-response": {"valid-token"},
	})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if recaptcha.calls != 1 {
		t.Fatalf("recaptcha calls = %d, want 1", recaptcha.calls)
	}
	if recaptcha.lastToken != "valid-token" {
		t.Fatalf("recaptcha token = %q, want valid-token", recaptcha.lastToken)
	}
	if recaptcha.lastAction != "register" {
		t.Fatalf("recaptcha action = %q, want register", recaptcha.lastAction)
	}
	exists, err := client.User.Query().Where(user.EmailEQ("recaptcha-valid@example.com")).Exist(context.Background())
	if err != nil {
		t.Fatalf("query user error = %v", err)
	}
	if !exists {
		t.Fatal("user should be created when recaptcha succeeds")
	}
}

func TestRegisterVerifyEmailGETShowsAutoSentState(t *testing.T) {
	handler, _, sessions, _ := newAuthHandlerTestEnv(t)

	register := performAuthRequest(sessions, handler.RegisterPOST, http.MethodPost, "/register", url.Values{
		"email":            {"sent@example.com"},
		"password":         {"Password123!"},
		"password_confirm": {"Password123!"},
	})
	if register.Code != http.StatusSeeOther {
		t.Fatalf("register status = %d, want %d", register.Code, http.StatusSeeOther)
	}

	rec := performAuthRequest(sessions, handler.RegisterVerifyEmailGET, http.MethodGet, "/register-verify-email", nil, register.Result().Cookies()...)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Verification email sent") {
		t.Fatalf("expected verify page to show email sent state, got: %s", rec.Body.String())
	}
}

func TestLoginPOSTUnverifiedUserRedirectsToVerifyWithoutSendingEmail(t *testing.T) {
	handler, _, sessions, mailer := newAuthHandlerTestEnv(t)
	u := createAuthTestUser(t, handler.Client, "unverified@example.com", "Password123!", false)

	rec := performAuthRequest(sessions, handler.LoginPOST, http.MethodPost, "/login", url.Values{
		"email":    {"unverified@example.com"},
		"password": {"Password123!"},
	})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/register-verify-email" {
		t.Fatalf("Location = %q, want /register-verify-email", got)
	}
	if mailer.htmlCalls != 0 {
		t.Fatalf("htmlCalls = %d, want 0", mailer.htmlCalls)
	}
	updated, err := handler.Client.User.Get(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("get user error = %v", err)
	}
	if updated.EmailVerificationToken != nil || updated.EmailVerificationExpiry != nil {
		t.Fatalf("expected no verification token before send button, got token=%v expiry=%v", updated.EmailVerificationToken, updated.EmailVerificationExpiry)
	}
}

func TestSendVerificationEmailReusesActiveTokenAndExtendsExpiry(t *testing.T) {
	handler, client, _, mailer := newAuthHandlerTestEnv(t)
	u := createAuthTestUser(t, client, "reuse@example.com", "Password123!", false)
	req := httptest.NewRequest(http.MethodPost, "/register-send-verification-email", nil)

	if _, err := handler.sendVerificationEmail(req, u); err != nil {
		t.Fatalf("first sendVerificationEmail() error = %v", err)
	}
	first, err := client.User.Get(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("get user error = %v", err)
	}
	firstToken := *first.EmailVerificationToken
	firstExpiry := *first.EmailVerificationExpiry

	time.Sleep(10 * time.Millisecond)
	if _, err := handler.sendVerificationEmail(req, first); err != nil {
		t.Fatalf("second sendVerificationEmail() error = %v", err)
	}
	second, err := client.User.Get(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("get user error = %v", err)
	}
	if *second.EmailVerificationToken != firstToken {
		t.Fatalf("token changed: first=%q second=%q", firstToken, *second.EmailVerificationToken)
	}
	if !second.EmailVerificationExpiry.After(firstExpiry) {
		t.Fatalf("expiry was not extended: first=%v second=%v", firstExpiry, *second.EmailVerificationExpiry)
	}
	if mailer.htmlCalls != 2 {
		t.Fatalf("htmlCalls = %d, want 2", mailer.htmlCalls)
	}
}

func TestVerifyEmailGETFirstLinkStillWorksAfterResend(t *testing.T) {
	handler, client, sessions, _ := newAuthHandlerTestEnv(t)
	u := createAuthTestUser(t, client, "verify@example.com", "Password123!", false)
	req := httptest.NewRequest(http.MethodPost, "/register-send-verification-email", nil)

	if _, err := handler.sendVerificationEmail(req, u); err != nil {
		t.Fatalf("first sendVerificationEmail() error = %v", err)
	}
	first, err := client.User.Get(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("get user error = %v", err)
	}
	token := *first.EmailVerificationToken
	if _, err := handler.sendVerificationEmail(req, first); err != nil {
		t.Fatalf("second sendVerificationEmail() error = %v", err)
	}

	rec := performAuthRequest(sessions, handler.VerifyEmailGET, http.MethodGet, "/verify-email?token="+url.QueryEscape(token), nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/dashboard" {
		t.Fatalf("Location = %q, want /dashboard", got)
	}

	verified, err := client.User.Get(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("get user error = %v", err)
	}
	if !verified.IsEmailVerified {
		t.Fatal("user should be verified")
	}
	if verified.EmailVerificationToken != nil || verified.EmailVerificationExpiry != nil {
		t.Fatalf("expected verification token fields to be cleared")
	}
}

func TestVerifyEmailGETExpiredTokenRendersExpiredPage(t *testing.T) {
	handler, client, sessions, _ := newAuthHandlerTestEnv(t)
	u := createAuthTestUser(t, client, "expired@example.com", "Password123!", false)
	token := "expired-token"
	_, err := client.User.UpdateOneID(u.ID).
		SetEmailVerificationToken(token).
		SetEmailVerificationExpiry(time.Now().Add(-time.Hour)).
		Save(context.Background())
	if err != nil {
		t.Fatalf("set expired token error = %v", err)
	}

	rec := performAuthRequest(sessions, handler.VerifyEmailGET, http.MethodGet, "/verify-email?token="+token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "Verification link expired") {
		t.Fatalf("expected expired page, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `name="verification_token" value="expired-token"`) {
		t.Fatalf("expected expired page to include resend token, got: %s", rec.Body.String())
	}
}

func TestRegisterSendVerificationEmailPOSTExpiredTokenWithoutSession(t *testing.T) {
	handler, client, sessions, mailer := newAuthHandlerTestEnv(t)
	u := createAuthTestUser(t, client, "expired-resend@example.com", "Password123!", false)
	_, err := client.User.UpdateOneID(u.ID).
		SetEmailVerificationToken("expired-resend-token").
		SetEmailVerificationExpiry(time.Now().Add(-time.Hour)).
		Save(context.Background())
	if err != nil {
		t.Fatalf("set expired token error = %v", err)
	}

	rec := performAuthRequest(sessions, handler.RegisterSendVerificationEmailPOST, http.MethodPost, "/register-send-verification-email", url.Values{
		"verification_token": {"expired-resend-token"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if mailer.htmlCalls != 1 {
		t.Fatalf("htmlCalls = %d, want 1", mailer.htmlCalls)
	}
	if !strings.Contains(rec.Body.String(), "Verification email sent") {
		t.Fatalf("expected success page, got: %s", rec.Body.String())
	}

	updated, err := client.User.Get(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("get user error = %v", err)
	}
	if updated.EmailVerificationToken == nil {
		t.Fatal("expected verification token to be set")
	}
	if *updated.EmailVerificationToken == "expired-resend-token" {
		t.Fatal("expected expired token to be replaced")
	}
	if updated.EmailVerificationExpiry == nil || !updated.EmailVerificationExpiry.After(time.Now()) {
		t.Fatalf("expected future verification expiry, got %v", updated.EmailVerificationExpiry)
	}
}

func TestRegisterSendVerificationEmailPOSTFailureDoesNotShowSuccess(t *testing.T) {
	handler, client, sessions, mailer := newAuthHandlerTestEnv(t)
	mailer.returnSendErr = errors.New("queue full")
	u := createAuthTestUser(t, client, "resend-fail@example.com", "Password123!", false)
	_, err := client.User.UpdateOneID(u.ID).
		SetEmailVerificationToken("resend-fail-token").
		SetEmailVerificationExpiry(time.Now().Add(-time.Hour)).
		Save(context.Background())
	if err != nil {
		t.Fatalf("set expired token error = %v", err)
	}

	rec := performAuthRequest(sessions, handler.RegisterSendVerificationEmailPOST, http.MethodPost, "/register-send-verification-email", url.Values{
		"verification_token": {"resend-fail-token"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "Verification email sent") {
		t.Fatalf("expected failure page not success, got: %s", body)
	}
	if !strings.Contains(body, "could not send the verification email") {
		t.Fatalf("expected delivery failure message, got: %s", body)
	}
}

func TestForgotPasswordPOSTUnknownEmailGenericSuccess(t *testing.T) {
	handler, _, sessions, mailer := newAuthHandlerTestEnv(t)

	rec := performAuthRequest(sessions, handler.ForgotPasswordPOST, http.MethodPost, "/forgot-password", url.Values{
		"email": {"missing@example.com"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if mailer.resetCalls != 0 {
		t.Fatalf("resetCalls = %d, want 0", mailer.resetCalls)
	}
	if !strings.Contains(rec.Body.String(), "Password reset email sent") {
		t.Fatalf("expected generic success response, got: %s", rec.Body.String())
	}
}

func TestForgotPasswordGETRendersRecaptchaHTMXWiring(t *testing.T) {
	handler, _, sessions, _ := newAuthHandlerTestEnv(t)
	handler.Recaptcha = &stubRecaptchaVerifier{siteKey: "site-key"}

	rec := performAuthRequest(sessions, handler.ForgotPasswordGET, http.MethodGet, "/forgot-password", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`https://www.google.com/recaptcha/api.js?render=site-key`,
		`hx-trigger="recaptcha-verified"`,
		`data-recaptcha-site-key="site-key"`,
		`name="g-recaptcha-response"`,
		`grecaptcha.ready`,
		`requestRecaptchaToken(0)`,
		`window.htmx.trigger(form, 'recaptcha-verified')`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected forgot-password body to contain %q, got: %s", want, body)
		}
	}
}

func TestForgotPasswordPOSTMissingRecaptchaTokenBlocksWhenConfigured(t *testing.T) {
	handler, _, sessions, mailer := newAuthHandlerTestEnv(t)
	recaptcha := &stubRecaptchaVerifier{siteKey: "site-key", requireToken: true}
	handler.Recaptcha = recaptcha

	rec := performAuthRequest(sessions, handler.ForgotPasswordPOST, http.MethodPost, "/forgot-password", url.Values{
		"email": {"reset-recaptcha@example.com"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if recaptcha.calls != 1 {
		t.Fatalf("recaptcha calls = %d, want 1", recaptcha.calls)
	}
	if recaptcha.lastAction != "forgot_password" {
		t.Fatalf("recaptcha action = %q, want forgot_password", recaptcha.lastAction)
	}
	if mailer.resetCalls != 0 {
		t.Fatalf("resetCalls = %d, want 0 when recaptcha fails", mailer.resetCalls)
	}
	if !strings.Contains(rec.Body.String(), "We could not verify this request") {
		t.Fatalf("expected recaptcha failure message, got: %s", rec.Body.String())
	}
}

func TestForgotPasswordPOSTValidRecaptchaSendsResetEmail(t *testing.T) {
	handler, client, sessions, mailer := newAuthHandlerTestEnv(t)
	createAuthTestUser(t, client, "recaptcha-reset@example.com", "Password123!", true)
	recaptcha := &stubRecaptchaVerifier{siteKey: "site-key", requireToken: true}
	handler.Recaptcha = recaptcha

	rec := performAuthRequest(sessions, handler.ForgotPasswordPOST, http.MethodPost, "/forgot-password", url.Values{
		"email":                {"recaptcha-reset@example.com"},
		"g-recaptcha-response": {"valid-token"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if recaptcha.calls != 1 {
		t.Fatalf("recaptcha calls = %d, want 1", recaptcha.calls)
	}
	if recaptcha.lastToken != "valid-token" {
		t.Fatalf("recaptcha token = %q, want valid-token", recaptcha.lastToken)
	}
	if recaptcha.lastAction != "forgot_password" {
		t.Fatalf("recaptcha action = %q, want forgot_password", recaptcha.lastAction)
	}
	if mailer.resetCalls != 1 {
		t.Fatalf("resetCalls = %d, want 1", mailer.resetCalls)
	}
}

func TestForgotPasswordPOSTSendsResetEmail(t *testing.T) {
	handler, client, sessions, mailer := newAuthHandlerTestEnv(t)
	u := createAuthTestUser(t, client, "reset@example.com", "Password123!", true)

	rec := performAuthRequest(sessions, handler.ForgotPasswordPOST, http.MethodPost, "/forgot-password", url.Values{
		"email": {"reset@example.com"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if mailer.resetCalls != 1 {
		t.Fatalf("resetCalls = %d, want 1", mailer.resetCalls)
	}
	if mailer.lastResetTo != "reset@example.com" {
		t.Fatalf("lastResetTo = %q", mailer.lastResetTo)
	}
	if !strings.HasPrefix(mailer.lastResetURL, "https://app.example.com/reset-password?token=") {
		t.Fatalf("lastResetURL = %q", mailer.lastResetURL)
	}

	updated, err := client.User.Get(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("get user error = %v", err)
	}
	if updated.PasswordResetToken == nil || updated.PasswordResetExpiry == nil {
		t.Fatalf("expected reset token and expiry, got token=%v expiry=%v", updated.PasswordResetToken, updated.PasswordResetExpiry)
	}
}

func TestResetPasswordPOSTSuccessClearsTokensAndVerifiesEmail(t *testing.T) {
	handler, client, sessions, _ := newAuthHandlerTestEnv(t)
	u := createAuthTestUser(t, client, "success@example.com", "Password123!", false)
	_, err := client.User.UpdateOneID(u.ID).
		SetPasswordResetToken("reset-token").
		SetPasswordResetExpiry(time.Now().Add(time.Hour)).
		SetEmailVerificationToken("verify-token").
		SetEmailVerificationExpiry(time.Now().Add(time.Hour)).
		Save(context.Background())
	if err != nil {
		t.Fatalf("set token error = %v", err)
	}

	rec := performAuthRequest(sessions, handler.ResetPasswordPOST, http.MethodPost, "/reset-password", url.Values{
		"token":            {"reset-token"},
		"password":         {"NewPassword123!"},
		"password_confirm": {"NewPassword123!"},
	})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/login" {
		t.Fatalf("Location = %q, want /login", got)
	}

	updated, err := client.User.Get(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("get user error = %v", err)
	}
	if !updated.IsEmailVerified {
		t.Fatal("email should be marked verified after reset")
	}
	if updated.PasswordResetToken != nil || updated.PasswordResetExpiry != nil {
		t.Fatalf("expected reset token fields to be cleared")
	}
	if updated.EmailVerificationToken != nil || updated.EmailVerificationExpiry != nil {
		t.Fatalf("expected verification token fields to be cleared")
	}
	ok, err := utils.CheckPassword(updated.PasswordHash, "NewPassword123!")
	if err != nil || !ok {
		t.Fatalf("new password did not verify: ok=%v err=%v", ok, err)
	}
}

func TestResetPasswordPOSTTokenCannotBeReused(t *testing.T) {
	handler, client, sessions, _ := newAuthHandlerTestEnv(t)
	u := createAuthTestUser(t, client, "reuse-reset@example.com", "Password123!", true)
	_, err := client.User.UpdateOneID(u.ID).
		SetPasswordResetToken("one-time-token").
		SetPasswordResetExpiry(time.Now().Add(time.Hour)).
		Save(context.Background())
	if err != nil {
		t.Fatalf("set token error = %v", err)
	}

	form := url.Values{
		"token":            {"one-time-token"},
		"password":         {"NewPassword123!"},
		"password_confirm": {"NewPassword123!"},
	}
	first := performAuthRequest(sessions, handler.ResetPasswordPOST, http.MethodPost, "/reset-password", form)
	if first.Code != http.StatusSeeOther {
		t.Fatalf("first status = %d, want %d, body: %s", first.Code, http.StatusSeeOther, first.Body.String())
	}

	second := performAuthRequest(sessions, handler.ResetPasswordPOST, http.MethodPost, "/reset-password", form)
	if second.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", second.Code, http.StatusOK)
	}
	if !strings.Contains(second.Body.String(), "Reset link invalid") {
		t.Fatalf("expected invalid reset page, got: %s", second.Body.String())
	}
}
