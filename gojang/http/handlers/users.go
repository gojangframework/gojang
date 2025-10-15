package handlers

import (
	"net/http"
	"strconv"

	"github.com/gojangframework/gojang/gojang/utils"

	"github.com/go-chi/chi/v5"

	"github.com/gojangframework/gojang/gojang/models"
	"github.com/gojangframework/gojang/gojang/models/user"
	"github.com/gojangframework/gojang/gojang/views/forms"
	"github.com/gojangframework/gojang/gojang/views/renderers"
)

type UserHandler struct {
	Client   *models.Client
	Renderer *renderers.Renderer
}

func NewUserHandler(client *models.Client, renderer *renderers.Renderer) *UserHandler {
	return &UserHandler{
		Client:   client,
		Renderer: renderer,
	}
}

// Index lists all users
func (h *UserHandler) Index(w http.ResponseWriter, r *http.Request) {
	users, err := h.Client.User.Query().All(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load users")
		return
	}

	h.Renderer.Render(w, r, "users/index.html", &renderers.TemplateData{
		Title: "User Management",
		Data: map[string]interface{}{
			"Users": users,
		},
	})
}

// New shows the create user form
func (h *UserHandler) New(w http.ResponseWriter, r *http.Request) {
	// Prevent direct access - modal forms must be loaded via HTMX
	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	h.Renderer.Render(w, r, "users/new.partial.html", nil)
}

// Create creates a new user
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	form := forms.UserForm{
		Email:       r.Form.Get("email"),
		Password:    r.Form.Get("password"),
		IsActive:    r.Form.Get("is_active") == "true",
		IsStaff:     r.Form.Get("is_staff") == "true",
		IsSuperuser: r.Form.Get("is_superuser") == "true",
	}

	// Validate
	errors := forms.Validate(form)
	if len(errors) > 0 {
		h.Renderer.Render(w, r, "users/new.partial.html", &renderers.TemplateData{
			Errors: errors,
		})
		return
	}

	// Check if user exists
	exists, err := h.Client.User.Query().Where(user.EmailEQ(form.Email)).Exist(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to check email availability")
		return
	}
	if exists {
		h.Renderer.Render(w, r, "users/new.partial.html", &renderers.TemplateData{
			Errors: map[string]string{"Email": "Email already exists"},
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
		SetIsActive(form.IsActive).
		SetIsStaff(form.IsStaff).
		SetIsSuperuser(form.IsSuperuser).
		Save(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Return new row
	h.Renderer.Render(w, r, "users/row.partial.html", &renderers.TemplateData{
		Data: map[string]interface{}{
			"User": u,
		},
	})
}

// Edit shows the edit user form
func (h *UserHandler) Edit(w http.ResponseWriter, r *http.Request) {
	// Prevent direct access - modal forms must be loaded via HTMX
	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid user ID")
		return
	}

	u, err := h.Client.User.Get(r.Context(), id)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "User not found")
		return
	}

	h.Renderer.Render(w, r, "users/edit.partial.html", &renderers.TemplateData{
		Data: map[string]interface{}{
			"User": u,
		},
	})
}

// Update updates a user
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	form := forms.UserForm{
		Email:       r.Form.Get("email"),
		Password:    r.Form.Get("password"),
		IsActive:    r.Form.Get("is_active") == "true",
		IsStaff:     r.Form.Get("is_staff") == "true",
		IsSuperuser: r.Form.Get("is_superuser") == "true",
	}

	// Update user
	updateQuery := h.Client.User.UpdateOneID(id).
		SetEmail(form.Email).
		SetIsActive(form.IsActive).
		SetIsStaff(form.IsStaff).
		SetIsSuperuser(form.IsSuperuser)

	// Update password if provided
	if form.Password != "" {
		hash, err := utils.HashPassword(form.Password)
		if err != nil {
			h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to hash password")
			return
		}
		updateQuery = updateQuery.SetPasswordHash(hash)
	}

	u, err := updateQuery.Save(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to update user")
		return
	}

	// Return updated row
	h.Renderer.Render(w, r, "users/row.partial.html", &renderers.TemplateData{
		Data: map[string]interface{}{
			"User": u,
		},
	})
}

// DeleteConfirm shows the delete confirmation modal
func (h *UserHandler) DeleteConfirm(w http.ResponseWriter, r *http.Request) {
	// Prevent direct access - modal forms must be loaded via HTMX
	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid user ID")
		return
	}

	u, err := h.Client.User.Get(r.Context(), id)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "User not found")
		return
	}

	h.Renderer.Render(w, r, "users/delete.partial.html", &renderers.TemplateData{
		Data: map[string]interface{}{
			"User": u,
		},
	})
}

// Delete deletes a user
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid user ID")
		return
	}

	err = h.Client.User.DeleteOneID(id).Exec(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	// Return empty response (row will be removed by htmx)
	w.WriteHeader(http.StatusOK)
}
