package handlers

import (
	"net/http"
	"time"

	"github.com/gojangframework/gojang/gojang/models"
	"github.com/gojangframework/gojang/gojang/models/user"
	"github.com/gojangframework/gojang/gojang/utils"
	"github.com/gojangframework/gojang/gojang/views/forms"
	"github.com/gojangframework/gojang/gojang/views/renderers"

	"github.com/alexedwards/scs/v2"
)

type AuthHandler struct {
	Client   *models.Client
	Sessions *scs.SessionManager
	Renderer *renderers.Renderer
}

func NewAuthHandler(client *models.Client, sessions *scs.SessionManager, renderer *renderers.Renderer) *AuthHandler {
	return &AuthHandler{
		Client:   client,
		Sessions: sessions,
		Renderer: renderer,
	}
}

// LoginGET shows the login form
func (h *AuthHandler) LoginGET(w http.ResponseWriter, r *http.Request) {
	// Pass the "next" parameter to the template
	nextURL := r.URL.Query().Get("next")
	h.Renderer.Render(w, r, "auth/login.html", &renderers.TemplateData{
		Data: map[string]interface{}{
			"Next": nextURL,
		},
	})
}

// LoginPOST handles login submission
func (h *AuthHandler) LoginPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	form := forms.LoginForm{
		Email:    r.Form.Get("email"),
		Password: r.Form.Get("password"),
	}

	// Validate form
	errors := forms.Validate(form)
	if len(errors) > 0 {
		h.Renderer.Render(w, r, "auth/login.html", &renderers.TemplateData{
			Errors: errors,
		})
		return
	}

	// Find user
	u, err := h.Client.User.Query().Where(user.EmailEQ(form.Email)).Only(r.Context())
	if err != nil {
		h.Renderer.Render(w, r, "auth/login.html", &renderers.TemplateData{
			Errors: map[string]string{"general": "Invalid email or password"},
		})
		return
	}

	// Check password
	ok, err := utils.CheckPassword(u.PasswordHash, form.Password)
	if err != nil || !ok {
		h.Renderer.Render(w, r, "auth/login.html", &renderers.TemplateData{
			Errors: map[string]string{"general": "Invalid email or password"},
		})
		return
	}

	// Check if user is active
	if !u.IsActive {
		h.Renderer.Render(w, r, "auth/login.html", &renderers.TemplateData{
			Errors: map[string]string{"general": "Your account is inactive"},
		})
		return
	}

	// Update last login
	if _, err := h.Client.User.UpdateOneID(u.ID).SetLastLogin(time.Now()).Save(r.Context()); err != nil {
		// Log error but don't fail login
		utils.Warnw("user.update_last_login_failed", "user_id", u.ID, "error", err)
	}

	// Create session
	h.Sessions.Put(r.Context(), "user_id", u.ID)
	h.Sessions.RenewToken(r.Context())

	// Determine redirect URL (check for "next" parameter from form or query)
	redirectURL := r.Form.Get("next")
	if redirectURL == "" {
		redirectURL = r.URL.Query().Get("next")
	}
	if redirectURL == "" {
		redirectURL = "/dashboard"
	}

	// Handle htmx vs regular request
	if r.Header.Get("HX-Request") == "true" {
		// Use htmx redirect header for client-side redirect
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// RegisterGET shows the registration form
func (h *AuthHandler) RegisterGET(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "auth/register.html", nil)
}

// RegisterPOST handles registration submission
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

	// Validate form
	errors := forms.Validate(form)
	if len(errors) > 0 {
		h.Renderer.Render(w, r, "auth/register.html", &renderers.TemplateData{
			Errors: errors,
			Data: map[string]interface{}{
				"Email": form.Email,
			},
		})
		return
	}

	// Check if user already exists
	exists, err := h.Client.User.Query().Where(user.EmailEQ(form.Email)).Exist(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to check email availability")
		return
	}
	if exists {
		h.Renderer.Render(w, r, "auth/register.html", &renderers.TemplateData{
			Errors: map[string]string{"Email": "Email already registered"},
			Data: map[string]interface{}{
				"Email": form.Email,
			},
		})
		return
	}

	// Hash password
	hash, err := utils.HashPassword(form.Password)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Create user
	u, err := h.Client.User.Create().
		SetEmail(form.Email).
		SetPasswordHash(hash).
		Save(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Auto-login
	h.Sessions.Put(r.Context(), "user_id", u.ID)
	h.Sessions.RenewToken(r.Context())

	// Handle htmx vs regular request
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/dashboard")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// LogoutPOST handles logout
func (h *AuthHandler) LogoutPOST(w http.ResponseWriter, r *http.Request) {
	_ = h.Sessions.Destroy(r.Context())

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
