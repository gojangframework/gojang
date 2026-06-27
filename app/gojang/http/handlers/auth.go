package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gojangframework/gojang/app/gojang/models"
	"github.com/gojangframework/gojang/app/gojang/models/user"
	"github.com/gojangframework/gojang/app/gojang/utils"
	"github.com/gojangframework/gojang/app/gojang/views/renderers"
	"github.com/gojangframework/gojang/app/views/forms"
	"github.com/google/uuid"

	"github.com/alexedwards/scs/v2"
)

type authEmailSender interface {
	SendHTMLEmail(to []string, subject, htmlBody string) error
	SendPasswordResetEmail(to, resetURL string) error
}

type recaptchaVerifier interface {
	SiteKey() string
	ScriptURL() string
	Verify(ctx context.Context, token, action, remoteIP string) error
}

type AuthHandler struct {
	Client       *models.Client
	Sessions     *scs.SessionManager
	Renderer     *renderers.Renderer
	EmailService authEmailSender
	AppBaseURL   string
	IsDebug      bool
	Recaptcha    recaptchaVerifier
}

func NewAuthHandler(client *models.Client, sessions *scs.SessionManager, renderer *renderers.Renderer, emailService authEmailSender, appBaseURL string, debug bool, recaptcha recaptchaVerifier) *AuthHandler {
	return &AuthHandler{
		Client:       client,
		Sessions:     sessions,
		Renderer:     renderer,
		EmailService: emailService,
		AppBaseURL:   appBaseURL,
		IsDebug:      debug,
		Recaptcha:    recaptcha,
	}
}

// LoginGET shows the login form.
func (h *AuthHandler) LoginGET(w http.ResponseWriter, r *http.Request) {
	nextURL := r.URL.Query().Get("next")
	h.Renderer.Render(w, r, "auth/login.html", &renderers.TemplateData{
		Data: map[string]interface{}{
			"Next": nextURL,
		},
	})
}

// LoginPOST handles login submission.
func (h *AuthHandler) LoginPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	form := forms.LoginForm{
		Email:    r.Form.Get("email"),
		Password: r.Form.Get("password"),
	}

	errors := forms.Validate(form)
	if len(errors) > 0 {
		h.Renderer.Render(w, r, "auth/login.html", &renderers.TemplateData{
			Errors: errors,
		})
		return
	}

	u, err := h.Client.User.Query().Where(user.EmailEQ(form.Email)).Only(r.Context())
	if err != nil {
		h.Renderer.Render(w, r, "auth/login.html", &renderers.TemplateData{
			Errors: map[string]string{"general": "Invalid email or password"},
		})
		return
	}

	ok, err := utils.CheckPassword(u.PasswordHash, form.Password)
	if err != nil || !ok {
		h.Renderer.Render(w, r, "auth/login.html", &renderers.TemplateData{
			Errors: map[string]string{"general": "Invalid email or password"},
		})
		return
	}

	if !u.IsActive {
		h.Renderer.Render(w, r, "auth/login.html", &renderers.TemplateData{
			Errors: map[string]string{"general": "Your account is inactive"},
		})
		return
	}

	if _, err := h.Client.User.UpdateOneID(u.ID).SetLastLogin(time.Now()).Save(r.Context()); err != nil {
		utils.Warnw("user.update_last_login_failed", "user_id", u.ID, "error", err)
	}

	h.putUserSession(r, u)

	if !u.IsEmailVerified {
		h.redirect(w, r, "/register-verify-email")
		return
	}

	redirectURL := r.Form.Get("next")
	if redirectURL == "" {
		redirectURL = r.URL.Query().Get("next")
	}
	if redirectURL == "" {
		redirectURL = "/dashboard"
	}

	h.redirect(w, r, redirectURL)
}

// RegisterGET shows the registration form.
func (h *AuthHandler) RegisterGET(w http.ResponseWriter, r *http.Request) {
	h.renderRegister(w, r, nil, "")
}

// RegisterPOST handles registration submission.
func (h *AuthHandler) RegisterPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	form := forms.RegisterForm{
		Email:           r.Form.Get("email"),
		Password:        r.Form.Get("password"),
		PasswordConfirm: r.Form.Get("password_confirm"),
	}

	errors := forms.Validate(form)
	if len(errors) > 0 {
		h.renderRegister(w, r, errors, form.Email)
		return
	}

	if h.recaptchaSiteKey() != "" {
		if err := h.Recaptcha.Verify(r.Context(), r.Form.Get("g-recaptcha-response"), "register", requestRemoteIP(r)); err != nil {
			logRecaptchaFailure("register", err)
			h.renderRegister(w, r, map[string]string{"general": recaptchaErrorMessage(err, "We could not verify this signup. Please try again.")}, form.Email)
			return
		}
	}

	exists, err := h.Client.User.Query().Where(user.EmailEQ(form.Email)).Exist(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to check email availability")
		return
	}
	if exists {
		h.renderRegister(w, r, map[string]string{"Email": "Email already registered"}, form.Email)
		return
	}

	hash, err := utils.HashPassword(form.Password)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	u, err := h.Client.User.Create().
		SetEmail(form.Email).
		SetPasswordHash(hash).
		Save(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to create user")
		return
	}

	h.putUserSession(r, u)
	if _, err := h.sendVerificationEmail(r, u); err != nil {
		utils.Warnw("verify_email.initial_send_failed", "user_id", u.ID, "error", err)
		h.Sessions.Put(r.Context(), "flash", "We could not send the verification email. Please try again.")
		h.Sessions.Put(r.Context(), "flash_type", "error")
	} else {
		h.Sessions.Put(r.Context(), "verification_email_sent", "true")
	}

	h.redirect(w, r, "/register-verify-email")
}

func (h *AuthHandler) renderRegister(w http.ResponseWriter, r *http.Request, errors map[string]string, email string) {
	data := map[string]interface{}{}
	if email != "" {
		data["Email"] = email
	}
	if siteKey := h.recaptchaSiteKey(); siteKey != "" {
		data["RecaptchaSiteKey"] = siteKey
		data["RecaptchaScriptURL"] = h.Recaptcha.ScriptURL()
	}

	h.Renderer.Render(w, r, "auth/register.html", &renderers.TemplateData{
		Errors: errors,
		Data:   data,
	})
}

func (h *AuthHandler) recaptchaSiteKey() string {
	if h.Recaptcha == nil {
		return ""
	}
	return h.Recaptcha.SiteKey()
}

func requestRemoteIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
		return ""
	}
	return ip.String()
}

func logRecaptchaFailure(scope string, err error) {
	if scope == "" {
		scope = "auth"
	}
	var verifyErr *utils.RecaptchaVerificationError
	if errors.As(err, &verifyErr) {
		utils.Warnw(scope+".recaptcha_failed",
			"error", err,
			"reason", verifyErr.Reason,
			"detail", verifyErr.Detail,
			"score", verifyErr.Score,
			"min_score", verifyErr.MinScore,
			"recaptcha_action", verifyErr.Action,
			"expected_action", verifyErr.ExpectedAction,
			"hostname", verifyErr.Hostname,
			"google_error_codes", strings.Join(verifyErr.ErrorCodes, ","),
			"status_code", verifyErr.StatusCode,
		)
		return
	}
	utils.Warnw(scope+".recaptcha_failed", "error", err)
}

func recaptchaErrorMessage(err error, genericMessage string) string {
	if recaptchaErrorHasCode(err, "browser-error") {
		return "reCAPTCHA could not complete in this browser. Please retry, disable browser blockers, or make sure this hostname is allowed for the site key."
	}
	return genericMessage
}

func recaptchaErrorHasCode(err error, code string) bool {
	var verifyErr *utils.RecaptchaVerificationError
	if !errors.As(err, &verifyErr) {
		return false
	}
	for _, errorCode := range verifyErr.ErrorCodes {
		if strings.EqualFold(strings.TrimSpace(errorCode), code) {
			return true
		}
	}
	return false
}

// LogoutPOST handles logout.
func (h *AuthHandler) LogoutPOST(w http.ResponseWriter, r *http.Request) {
	_ = h.Sessions.Destroy(r.Context())
	h.redirect(w, r, "/")
}

func (h *AuthHandler) putUserSession(r *http.Request, u *models.User) {
	h.Sessions.Put(r.Context(), "user_id", u.ID.String())
	h.Sessions.Put(r.Context(), "email", u.Email)
	h.Sessions.RenewToken(r.Context())
}

func (h *AuthHandler) redirect(w http.ResponseWriter, r *http.Request, redirectURL string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (h *AuthHandler) buildAuthURL(path string, query url.Values) string {
	base := strings.TrimSpace(h.AppBaseURL)
	if base == "" {
		base = "http://localhost:8080"
	}

	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		u = &url.URL{Scheme: "http", Host: "localhost:8080"}
	}
	u.Path = path
	u.RawQuery = query.Encode()
	return u.String()
}

func (h *AuthHandler) sendVerificationEmail(r *http.Request, u *models.User) (string, error) {
	expiresAt := time.Now().Add(48 * time.Hour)
	token := ""

	if !u.IsEmailVerified && u.EmailVerificationToken != nil && u.EmailVerificationExpiry != nil && time.Now().Before(*u.EmailVerificationExpiry) {
		token = *u.EmailVerificationToken
	} else {
		generated, err := generateSecureToken()
		if err != nil {
			utils.Errorw("verify_email.token_generation_failed", "error", err)
			return "", err
		}
		token = generated
	}

	if _, err := h.Client.User.UpdateOneID(u.ID).
		SetEmailVerificationToken(token).
		SetEmailVerificationExpiry(expiresAt).
		Save(r.Context()); err != nil {
		utils.Errorw("verify_email.update_user_failed", "user_id", u.ID, "error", err)
		return "", err
	}

	verificationURL := h.buildAuthURL("/verify-email", url.Values{"token": {token}})
	if h.EmailService == nil {
		return verificationURL, fmt.Errorf("email service not configured")
	}

	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Verify your email</title></head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #222;">
  <div style="max-width: 600px; margin: 0 auto; padding: 24px;">
    <h2>Verify your email</h2>
    <p>Thanks for creating a Gojang account. Verify your email address to finish setting it up.</p>
    <p style="margin: 28px 0;">
      <a href="%s" style="background-color: #2563eb; color: #fff; padding: 12px 20px; text-decoration: none; border-radius: 4px; display: inline-block;">Verify email</a>
    </p>
    <p>If the button does not work, copy and paste this link into your browser:</p>
    <p style="word-break: break-all;"><a href="%s">%s</a></p>
    <p style="color: #64748b; font-size: 12px; margin-top: 28px;">This link expires in 48 hours.</p>
  </div>
</body>
</html>`, verificationURL, verificationURL, verificationURL)

	if err := h.EmailService.SendHTMLEmail([]string{u.Email}, "Verify your Gojang email", body); err != nil {
		utils.Errorw("verify_email.send_failed", "user_id", u.ID, "email", u.Email, "error", err)
		return verificationURL, err
	}

	utils.Infow("verify_email.email_queued", "user_id", u.ID, "email", u.Email)
	return verificationURL, nil
}

func (h *AuthHandler) sendPasswordResetEmail(r *http.Request, u *models.User) (string, error) {
	token, err := generateSecureToken()
	if err != nil {
		utils.Errorw("password_reset.token_generation_failed", "error", err)
		return "", err
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	if _, err := h.Client.User.UpdateOneID(u.ID).
		SetPasswordResetToken(token).
		SetPasswordResetExpiry(expiresAt).
		Save(r.Context()); err != nil {
		utils.Errorw("password_reset.update_user_failed", "user_id", u.ID, "error", err)
		return "", err
	}

	resetURL := h.buildAuthURL("/reset-password", url.Values{"token": {token}})
	if h.EmailService == nil {
		return resetURL, fmt.Errorf("email service not configured")
	}

	if err := h.EmailService.SendPasswordResetEmail(u.Email, resetURL); err != nil {
		utils.Errorw("password_reset.send_failed", "user_id", u.ID, "email", u.Email, "error", err)
		return resetURL, err
	}

	utils.Infow("password_reset.email_queued", "user_id", u.ID, "email", u.Email)
	return resetURL, nil
}

// ForgotPasswordGET shows the forgot password form.
func (h *AuthHandler) ForgotPasswordGET(w http.ResponseWriter, r *http.Request) {
	h.renderForgotPassword(w, r, nil, map[string]interface{}{})
}

// ForgotPasswordPOST handles forgot password submission.
func (h *AuthHandler) ForgotPasswordPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	form := forms.ForgotPasswordForm{Email: r.Form.Get("email")}
	if errors := forms.Validate(form); len(errors) > 0 {
		h.renderForgotPassword(w, r, errors, map[string]interface{}{"Email": form.Email})
		return
	}

	if h.recaptchaSiteKey() != "" {
		if err := h.Recaptcha.Verify(r.Context(), r.Form.Get("g-recaptcha-response"), "forgot_password", requestRemoteIP(r)); err != nil {
			logRecaptchaFailure("forgot_password", err)
			h.renderForgotPassword(w, r, map[string]string{"general": recaptchaErrorMessage(err, "We could not verify this request. Please try again.")}, map[string]interface{}{"Email": form.Email})
			return
		}
	}

	debugURL := ""
	u, err := h.Client.User.Query().
		Where(user.EmailEQ(form.Email), user.IsActiveEQ(true)).
		Only(r.Context())
	if err == nil {
		resetURL, sendErr := h.sendPasswordResetEmail(r, u)
		if sendErr != nil {
			utils.Warnw("password_reset.email_not_sent", "user_id", u.ID, "error", sendErr)
		}
		if h.IsDebug {
			debugURL = resetURL
		}
	} else {
		utils.Debugw("password_reset.user_not_found", "error", err)
	}

	h.renderForgotPassword(w, r, nil, map[string]interface{}{
		"Email":     form.Email,
		"EmailSent": true,
		"DebugURL":  debugURL,
	})
}

func (h *AuthHandler) renderForgotPassword(w http.ResponseWriter, r *http.Request, errors map[string]string, data map[string]interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	if siteKey := h.recaptchaSiteKey(); siteKey != "" {
		data["RecaptchaSiteKey"] = siteKey
		data["RecaptchaScriptURL"] = h.Recaptcha.ScriptURL()
	}

	h.Renderer.Render(w, r, "auth/forgot_password.html", &renderers.TemplateData{
		Errors: errors,
		Data:   data,
	})
}

// ResetPasswordGET shows the reset password form for a valid token.
func (h *AuthHandler) ResetPasswordGET(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		h.Renderer.Render(w, r, "auth/reset_password_expired.html", nil)
		return
	}

	u, err := h.Client.User.Query().
		Where(user.PasswordResetTokenEQ(token), user.IsActiveEQ(true)).
		Only(r.Context())
	if err != nil || u.PasswordResetExpiry == nil || time.Now().After(*u.PasswordResetExpiry) {
		utils.Warnw("password_reset.invalid_or_expired_token", "error", err)
		h.Renderer.Render(w, r, "auth/reset_password_expired.html", nil)
		return
	}

	h.Renderer.Render(w, r, "auth/reset_password.html", &renderers.TemplateData{
		Data: map[string]interface{}{
			"Token": token,
		},
	})
}

// ResetPasswordPOST handles password reset submission.
func (h *AuthHandler) ResetPasswordPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	form := forms.ResetPasswordForm{
		Token:           r.Form.Get("token"),
		Password:        r.Form.Get("password"),
		PasswordConfirm: r.Form.Get("password_confirm"),
	}

	if errors := forms.Validate(form); len(errors) > 0 {
		h.Renderer.Render(w, r, "auth/reset_password.html", &renderers.TemplateData{
			Errors: errors,
			Data: map[string]interface{}{
				"Token": form.Token,
			},
		})
		return
	}

	u, err := h.Client.User.Query().
		Where(user.PasswordResetTokenEQ(form.Token), user.IsActiveEQ(true)).
		Only(r.Context())
	if err != nil || u.PasswordResetExpiry == nil || time.Now().After(*u.PasswordResetExpiry) {
		utils.Warnw("password_reset.invalid_or_expired_token", "error", err)
		h.Renderer.Render(w, r, "auth/reset_password_expired.html", nil)
		return
	}

	hash, err := utils.HashPassword(form.Password)
	if err != nil {
		utils.Errorw("password_reset.hash_password_failed", "user_id", u.ID, "error", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to update password")
		return
	}

	if _, err := h.Client.User.UpdateOneID(u.ID).
		SetPasswordHash(hash).
		ClearPasswordResetToken().
		ClearPasswordResetExpiry().
		SetIsEmailVerified(true).
		ClearEmailVerificationToken().
		ClearEmailVerificationExpiry().
		Save(r.Context()); err != nil {
		utils.Errorw("password_reset.update_user_failed", "user_id", u.ID, "error", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to update password")
		return
	}

	h.Sessions.Put(r.Context(), "flash", "Password updated successfully. Please sign in.")
	h.Sessions.Put(r.Context(), "flash_type", "success")
	h.redirect(w, r, "/login")
}

// RegisterVerifyEmailGET shows the email verification page.
func (h *AuthHandler) RegisterVerifyEmailGET(w http.ResponseWriter, r *http.Request) {
	userIDStr := h.Sessions.GetString(r.Context(), "user_id")
	email := h.Sessions.GetString(r.Context(), "email")
	if userIDStr == "" || email == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	h.Renderer.Render(w, r, "auth/verify_email.html", &renderers.TemplateData{
		Data: map[string]interface{}{
			"Email":     email,
			"EmailSent": h.Sessions.PopString(r.Context(), "verification_email_sent") == "true",
		},
	})
}

// RegisterSendVerificationEmailPOST sends or resends the verification email.
func (h *AuthHandler) RegisterSendVerificationEmailPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	userIDStr := h.Sessions.GetString(r.Context(), "user_id")
	var u *models.User
	var err error

	if userIDStr != "" {
		userID, parseErr := uuid.Parse(userIDStr)
		if parseErr != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		u, err = h.Client.User.Query().Where(user.IDEQ(userID)).Only(r.Context())
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
	} else {
		token := r.Form.Get("verification_token")
		if token == "" {
			h.Renderer.Render(w, r, "auth/verify_email_expired.html", &renderers.TemplateData{
				Flash:     "Please sign in to request a new verification email.",
				FlashType: "error",
			})
			return
		}

		u, err = h.Client.User.Query().
			Where(user.EmailVerificationTokenEQ(token), user.IsActiveEQ(true)).
			Only(r.Context())
		if err != nil {
			utils.Warnw("verify_email.resend_invalid_token", "error", err)
			h.Renderer.Render(w, r, "auth/verify_email_expired.html", &renderers.TemplateData{
				Flash:     "Verification link expired. Please sign in to request a new email.",
				FlashType: "error",
			})
			return
		}
	}

	if u.IsEmailVerified {
		h.redirect(w, r, "/dashboard")
		return
	}

	verificationURL, err := h.sendVerificationEmail(r, u)
	if err != nil {
		utils.Warnw("verify_email.resend_failed", "user_id", u.ID, "error", err)
		h.Renderer.Render(w, r, "auth/verify_email.html", &renderers.TemplateData{
			Flash:     "We could not send the verification email. Please try again.",
			FlashType: "error",
			Data: map[string]interface{}{
				"Email": u.Email,
			},
		})
		return
	}

	debugURL := ""
	if h.IsDebug {
		debugURL = verificationURL
	}

	h.Renderer.Render(w, r, "auth/verify_email.html", &renderers.TemplateData{
		Flash:     "Verification email sent. Please check your inbox.",
		FlashType: "success",
		Data: map[string]interface{}{
			"Email":     u.Email,
			"EmailSent": true,
			"DebugURL":  debugURL,
		},
	})
}

// VerifyEmailGET handles the email verification link.
func (h *AuthHandler) VerifyEmailGET(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid verification link")
		return
	}

	u, err := h.Client.User.Query().
		Where(user.EmailVerificationTokenEQ(token), user.IsActiveEQ(true)).
		Only(r.Context())
	if err != nil {
		utils.Warnw("verify_email.invalid_token", "error", err)
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid or expired verification link")
		return
	}

	if u.IsEmailVerified {
		h.Sessions.Put(r.Context(), "flash", "Email already verified.")
		h.Sessions.Put(r.Context(), "flash_type", "success")
		h.redirect(w, r, "/login")
		return
	}

	if u.EmailVerificationExpiry == nil || time.Now().After(*u.EmailVerificationExpiry) {
		h.Renderer.Render(w, r, "auth/verify_email_expired.html", &renderers.TemplateData{
			Data: map[string]interface{}{
				"Email": u.Email,
				"Token": token,
			},
		})
		return
	}

	if _, err := h.Client.User.UpdateOneID(u.ID).
		SetIsEmailVerified(true).
		ClearEmailVerificationToken().
		ClearEmailVerificationExpiry().
		Save(r.Context()); err != nil {
		utils.Errorw("verify_email.update_failed", "user_id", u.ID, "error", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to verify email")
		return
	}

	h.putUserSession(r, u)
	h.redirect(w, r, "/dashboard")
}
