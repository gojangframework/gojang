package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultRecaptchaVerifyURL = "https://www.google.com/recaptcha/api/siteverify"
	defaultRecaptchaTimeout   = 5 * time.Second
)

var ErrRecaptchaVerificationFailed = errors.New("recaptcha verification failed")

type RecaptchaVerificationError struct {
	Reason         string
	Detail         string
	Score          float64
	MinScore       float64
	Action         string
	ExpectedAction string
	Hostname       string
	ErrorCodes     []string
	StatusCode     int
}

func (e *RecaptchaVerificationError) Error() string {
	if e == nil || e.Reason == "" {
		return ErrRecaptchaVerificationFailed.Error()
	}
	if e.Detail == "" {
		return ErrRecaptchaVerificationFailed.Error() + ": " + e.Reason
	}
	return ErrRecaptchaVerificationFailed.Error() + ": " + e.Reason + ": " + e.Detail
}

func (e *RecaptchaVerificationError) Unwrap() error {
	return ErrRecaptchaVerificationFailed
}

type RecaptchaConfig struct {
	SiteKey          string
	SecretKey        string
	MinScore         float64
	VerifyURL        string
	Client           *http.Client
	AllowedHostnames []string
}

type RecaptchaVerifier struct {
	siteKey          string
	secretKey        string
	minScore         float64
	verifyURL        string
	client           *http.Client
	allowedHostnames []string
}

type recaptchaVerifyResponse struct {
	Success    bool     `json:"success"`
	Score      float64  `json:"score"`
	Action     string   `json:"action"`
	Hostname   string   `json:"hostname"`
	ErrorCodes []string `json:"error-codes"`
}

func NewRecaptchaVerifier(cfg RecaptchaConfig) *RecaptchaVerifier {
	minScore := cfg.MinScore
	if minScore <= 0 {
		minScore = 0.5
	}

	verifyURL := strings.TrimSpace(cfg.VerifyURL)
	if verifyURL == "" {
		verifyURL = defaultRecaptchaVerifyURL
	}

	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: defaultRecaptchaTimeout}
	}

	return &RecaptchaVerifier{
		siteKey:          strings.TrimSpace(cfg.SiteKey),
		secretKey:        strings.TrimSpace(cfg.SecretKey),
		minScore:         minScore,
		verifyURL:        verifyURL,
		client:           client,
		allowedHostnames: cleanStringValues(cfg.AllowedHostnames),
	}
}

func (v *RecaptchaVerifier) Enabled() bool {
	return v != nil && v.siteKey != "" && v.secretKey != ""
}

func (v *RecaptchaVerifier) SiteKey() string {
	if !v.Enabled() {
		return ""
	}
	return v.siteKey
}

func (v *RecaptchaVerifier) ScriptURL() string {
	if !v.Enabled() {
		return ""
	}
	return "https://www.google.com/recaptcha/api.js?render=" + url.QueryEscape(v.siteKey)
}

func (v *RecaptchaVerifier) Verify(ctx context.Context, token, action, remoteIP string) error {
	if !v.Enabled() {
		return nil
	}

	token, action = cleanTokenAction(token, action)
	if token == "" {
		return recaptchaFailure("missing token")
	}
	if action == "" {
		return recaptchaFailure("missing expected action")
	}

	form := url.Values{
		"secret":   {v.secretKey},
		"response": {token},
	}
	if strings.TrimSpace(remoteIP) != "" {
		form.Set("remoteip", strings.TrimSpace(remoteIP))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("%w: build request", ErrRecaptchaVerificationFailed)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
	if err != nil {
		return recaptchaFailureWithDetail("verify request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &RecaptchaVerificationError{
			Reason:     "unexpected status",
			StatusCode: resp.StatusCode,
		}
	}

	var payload recaptchaVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return recaptchaFailureWithDetail("decode response", err)
	}
	if !payload.Success {
		return &RecaptchaVerificationError{
			Reason:     "google rejected token",
			Score:      payload.Score,
			Action:     payload.Action,
			Hostname:   payload.Hostname,
			ErrorCodes: append([]string(nil), payload.ErrorCodes...),
		}
	}
	if payload.Action != action {
		return &RecaptchaVerificationError{
			Reason:         "action mismatch",
			Score:          payload.Score,
			Action:         payload.Action,
			ExpectedAction: action,
			Hostname:       payload.Hostname,
			ErrorCodes:     append([]string(nil), payload.ErrorCodes...),
		}
	}
	if err := v.verifyHostname(payload.Hostname, payload.Score, action); err != nil {
		return err
	}
	if payload.Score < v.minScore {
		return &RecaptchaVerificationError{
			Reason:         "score below threshold",
			Score:          payload.Score,
			MinScore:       v.minScore,
			Action:         payload.Action,
			ExpectedAction: action,
			Hostname:       payload.Hostname,
			ErrorCodes:     append([]string(nil), payload.ErrorCodes...),
		}
	}

	return nil
}

func (v *RecaptchaVerifier) verifyHostname(hostname string, score float64, action string) error {
	if len(v.allowedHostnames) == 0 {
		return nil
	}
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if hostname == "" {
		return &RecaptchaVerificationError{
			Reason:         "hostname missing",
			Score:          score,
			Action:         action,
			ExpectedAction: action,
		}
	}
	for _, allowed := range v.allowedHostnames {
		if hostnameMatches(allowed, hostname) {
			return nil
		}
	}
	return &RecaptchaVerificationError{
		Reason:         "hostname mismatch",
		Score:          score,
		Action:         action,
		ExpectedAction: action,
		Hostname:       hostname,
	}
}

func hostnameMatches(pattern, hostname string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if pattern == "" || hostname == "" {
		return false
	}
	if pattern == hostname {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*.")
		return hostname != suffix && strings.HasSuffix(hostname, "."+suffix)
	}
	return false
}

func cleanTokenAction(token, action string) (string, string) {
	return strings.TrimSpace(token), strings.TrimSpace(action)
}

func cleanStringValues(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func recaptchaFailure(reason string) error {
	return &RecaptchaVerificationError{Reason: reason}
}

func recaptchaFailureWithDetail(reason string, err error) error {
	if err == nil {
		return recaptchaFailure(reason)
	}
	return &RecaptchaVerificationError{
		Reason: reason,
		Detail: err.Error(),
	}
}
