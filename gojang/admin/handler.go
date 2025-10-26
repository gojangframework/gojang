package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gojangframework/gojang/gojang/utils"
	"github.com/google/uuid"

	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/gojangframework/gojang/gojang/models/user"
)

// Handler handles all admin panel requests
type Handler struct {
	Registry *Registry
	Renderer *AdminRenderer
	DB       *models.Client
}

// NewHandler creates a new admin handler
func NewHandler(registry *Registry, renderer *AdminRenderer, db *models.Client) *Handler {
	return &Handler{
		Registry: registry,
		Renderer: renderer,
		DB:       db,
	}
}

// Dashboard shows the admin dashboard with all registered models
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	models := h.Registry.List()

	h.Renderer.Render(w, r, "admin_main.html", &TemplateData{
		Title: "Admin Dashboard",
		Data: map[string]interface{}{
			"Models": models,
		},
	})
}

// Index lists all records for a model
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")

	config, err := h.Registry.Get(modelName)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Model not found")
		return
	}

	// Parse pagination params
	page := 1
	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	perPage := 20
	if v := r.URL.Query().Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && (pp == 20 || pp == 50 || pp == 100) {
			perPage = pp
		}
	}
	offset := (page - 1) * perPage

	totalCount, err := config.CountAll(r.Context())
	if err != nil {
		utils.Errorw("admin.count_failed", "model", config.Name, "error", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, fmt.Sprintf("Failed to load %s", config.NamePlural))
		return
	}

	records, err := config.QueryAllPaginated(r.Context(), perPage, offset)
	if err != nil {
		utils.Errorw("admin.query_failed", "model", config.Name, "error", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, fmt.Sprintf("Failed to load %s", config.NamePlural))
		return
	}

	totalPages := (totalCount + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	h.Renderer.Render(w, r, "model_index.html", &TemplateData{
		Title: config.NamePlural,
		Data: map[string]interface{}{
			"Config":     config,
			"Records":    records,
			"Page":       page,
			"PerPage":    perPage,
			"TotalPages": totalPages,
			"TotalCount": totalCount,
		},
	})
}

// New shows the create form for a model
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")

	config, err := h.Registry.Get(modelName)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Model not found")
		return
	}

	// Prevent direct access - modal forms must be loaded via HTMX
	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, "/admin/"+modelName, http.StatusSeeOther)
		return
	}

	// Get pagination params to pass to template for form submission
	page := 1
	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	perPage := 20
	if v := r.URL.Query().Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && (pp == 20 || pp == 50 || pp == 100) {
			perPage = pp
		}
	}

	h.Renderer.Render(w, r, "model_form.partial.html", &TemplateData{
		Title: "New " + config.Name,
		Data: map[string]interface{}{
			"Config":  config,
			"Action":  "create",
			"Page":    page,
			"PerPage": perPage,
		},
	})
}

// Create creates a new record
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")

	config, err := h.Registry.Get(modelName)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Model not found")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	// Extract form data into map
	data := make(map[string]interface{})
	for _, field := range config.Fields {
		if field.Readonly || field.Hidden {
			continue
		}

		// Special handling for checkboxes: unchecked boxes don't appear in form data
		if field.Type == FieldTypeBool {
			_, exists := r.Form[field.Name]
			data[field.Name] = exists
		} else {
			value := r.Form.Get(field.Name)
			data[field.Name] = h.parseFieldValue(field, value)
		}
	}

	// Validate required fields
	errors := h.validateFields(config, data, true) // true = creating new record
	if len(errors) > 0 {
		w.Header().Set("HX-Retarget", "#form-modal")
		w.Header().Set("HX-Reswap", "innerHTML")
		h.Renderer.Render(w, r, "model_form.partial.html", &TemplateData{
			Title:  "New " + config.Name,
			Errors: errors,
			Data: map[string]interface{}{
				"Config":   config,
				"Action":   "create",
				"FormData": data,
			},
		})
		return
	}

	// Check for duplicate email when creating a User
	if config.Name == "User" {
		if email, ok := data["Email"].(string); ok && email != "" {
			exists, err := h.DB.User.Query().Where(user.EmailEQ(email)).Exist(r.Context())
			if err != nil {
				utils.Errorw("admin.check_email_failed", "error", err)
				h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to check email")
				return
			}
			if exists {
				w.Header().Set("HX-Retarget", "#form-modal")
				w.Header().Set("HX-Reswap", "innerHTML")
				h.Renderer.Render(w, r, "model_form.partial.html", &TemplateData{
					Title:  "New " + config.Name,
					Errors: map[string]string{"Email": "This email address is already registered"},
					Data: map[string]interface{}{
						"Config":   config,
						"Action":   "create",
						"FormData": data,
					},
				})
				return
			}
		}
	}

	// Create the record
	_, err = config.CreateFunc(r.Context(), data)
	if err != nil {
		utils.Errorw("admin.create_failed", "model", config.Name, "error", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, fmt.Sprintf("Failed to create %s", config.Name))
		return
	}

	// Parse pagination params for the list response
	page := 1
	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	perPage := 20
	if v := r.URL.Query().Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && (pp == 20 || pp == 50 || pp == 100) {
			perPage = pp
		}
	}
	offset := (page - 1) * perPage

	totalCount, err := config.CountAll(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load records")
		return
	}

	records, err := config.QueryAllPaginated(r.Context(), perPage, offset)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load records")
		return
	}

	totalPages := (totalCount + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	w.Header().Set("HX-Trigger", "closeFormModal")

	h.Renderer.Render(w, r, "model_list.partial.html", &TemplateData{
		Data: map[string]interface{}{
			"Config":     config,
			"Records":    records,
			"Page":       page,
			"PerPage":    perPage,
			"TotalPages": totalPages,
			"TotalCount": totalCount,
		},
	})
}

// Edit shows the edit form
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")
	idStr := chi.URLParam(r, "id")

	config, err := h.Registry.Get(modelName)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Model not found")
		return
	}

	// Prevent direct access - modal forms must be loaded via HTMX
	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, "/admin/"+modelName, http.StatusSeeOther)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid ID")
		return
	}

	record, err := config.QueryByID(r.Context(), id)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, config.Name+" not found")
		return
	}

	// Get pagination params to pass to template for form submission
	page := 1
	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	perPage := 20
	if v := r.URL.Query().Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && (pp == 20 || pp == 50 || pp == 100) {
			perPage = pp
		}
	}

	h.Renderer.Render(w, r, "model_form.partial.html", &TemplateData{
		Title: "Edit " + config.Name,
		Data: map[string]interface{}{
			"Config":  config,
			"Record":  record,
			"Page":    page,
			"PerPage": perPage,
		},
	})
}

// Update updates a record
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")
	idStr := chi.URLParam(r, "id")

	config, err := h.Registry.Get(modelName)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Model not found")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	// Extract form data
	data := make(map[string]interface{})
	for _, field := range config.Fields {
		if field.Readonly || field.Hidden {
			continue
		}

		// Special handling for checkboxes: unchecked boxes don't appear in form data
		if field.Type == FieldTypeBool {
			_, exists := r.Form[field.Name]
			data[field.Name] = exists
		} else {
			value := r.Form.Get(field.Name)
			data[field.Name] = h.parseFieldValue(field, value)
		}
	}

	// Validate required fields
	errors := h.validateFields(config, data, false) // false = not creating, it's an update
	if len(errors) > 0 {
		record, err := config.QueryByID(r.Context(), id)
		if err != nil {
			h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load record")
			return
		}
		w.Header().Set("HX-Retarget", "#form-modal")
		w.Header().Set("HX-Reswap", "innerHTML")
		h.Renderer.Render(w, r, "model_form.partial.html", &TemplateData{
			Title:  "Edit " + config.Name,
			Errors: errors,
			Data: map[string]interface{}{
				"Config":   config,
				"Action":   "edit",
				"Record":   record,
				"ID":       id,
				"FormData": data,
			},
		})
		return
	}

	// Check for duplicate email when updating a User (excluding the current user)
	if config.Name == "User" {
		if email, ok := data["Email"].(string); ok && email != "" {
			exists, err := h.DB.User.Query().
				Where(user.EmailEQ(email)).
				Where(user.IDNEQ(id)).
				Exist(r.Context())
			if err != nil {
				utils.Errorw("admin.check_email_failed", "error", err)
				h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to check email")
				return
			}
			if exists {
				record, err := config.QueryByID(r.Context(), id)
				if err != nil {
					h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load record")
					return
				}
				w.Header().Set("HX-Retarget", "#form-modal")
				w.Header().Set("HX-Reswap", "innerHTML")
				h.Renderer.Render(w, r, "model_form.partial.html", &TemplateData{
					Title:  "Edit " + config.Name,
					Errors: map[string]string{"Email": "This email address is already registered"},
					Data: map[string]interface{}{
						"Config":   config,
						"Action":   "edit",
						"Record":   record,
						"ID":       id,
						"FormData": data,
					},
				})
				return
			}
		}
	}

	// Update
	err = config.UpdateFunc(r.Context(), id, data)
	if err != nil {
		utils.Errorw("admin.update_failed", "model", config.Name, "error", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, fmt.Sprintf("Failed to update %s", config.Name))
		return
	}

	// Parse pagination params for the list response
	page := 1
	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	perPage := 20
	if v := r.URL.Query().Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && (pp == 20 || pp == 50 || pp == 100) {
			perPage = pp
		}
	}
	offset := (page - 1) * perPage

	totalCount, err := config.CountAll(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load records")
		return
	}

	records, err := config.QueryAllPaginated(r.Context(), perPage, offset)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load records")
		return
	}

	totalPages := (totalCount + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	w.Header().Set("HX-Trigger", "closeFormModal")

	h.Renderer.Render(w, r, "model_list.partial.html", &TemplateData{
		Data: map[string]interface{}{
			"Config":     config,
			"Records":    records,
			"Page":       page,
			"PerPage":    perPage,
			"TotalPages": totalPages,
			"TotalCount": totalCount,
		},
	})
}

// DeleteConfirm shows delete confirmation
func (h *Handler) DeleteConfirm(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")
	idStr := chi.URLParam(r, "id")

	config, err := h.Registry.Get(modelName)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Model not found")
		return
	}

	// Prevent direct access - modal forms must be loaded via HTMX
	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, "/admin/"+modelName, http.StatusSeeOther)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid ID")
		return
	}

	record, err := config.QueryByID(r.Context(), id)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, config.Name+" not found")
		return
	}

	// Get pagination params to pass to template for form submission
	page := 1
	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	perPage := 20
	if v := r.URL.Query().Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && (pp == 20 || pp == 50 || pp == 100) {
			perPage = pp
		}
	}

	h.Renderer.Render(w, r, "model_delete.partial.html", &TemplateData{
		Title: "Delete " + config.Name,
		Data: map[string]interface{}{
			"Config":  config,
			"Record":  record,
			"ID":      id,
			"Page":    page,
			"PerPage": perPage,
		},
	})
}

// Delete deletes a record
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")
	idStr := chi.URLParam(r, "id")

	config, err := h.Registry.Get(modelName)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Model not found")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid ID")
		return
	}

	err = config.DeleteFunc(r.Context(), id)
	if err != nil {
		utils.Errorw("admin.delete_failed", "model", config.Name, "error", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, fmt.Sprintf("Failed to delete %s", config.Name))
		return
	}

	// Parse pagination params for the list response
	page := 1
	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	perPage := 20
	if v := r.URL.Query().Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && (pp == 20 || pp == 50 || pp == 100) {
			perPage = pp
		}
	}
	offset := (page - 1) * perPage

	totalCount, err := config.CountAll(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load records")
		return
	}

	records, err := config.QueryAllPaginated(r.Context(), perPage, offset)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load records")
		return
	}

	totalPages := (totalCount + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	// Trigger modal close via HTMX event
	w.Header().Set("HX-Trigger", "closeDeleteModal")

	h.Renderer.Render(w, r, "model_list.partial.html", &TemplateData{
		Data: map[string]interface{}{
			"Config":     config,
			"Records":    records,
			"Page":       page,
			"PerPage":    perPage,
			"TotalPages": totalPages,
			"TotalCount": totalCount,
		},
	})
}

// parseFieldValue parses a form value based on field type
func (h *Handler) parseFieldValue(field FieldConfig, value string) interface{} {
	switch field.Type {
	case FieldTypeBool:
		return value == "on" || value == "true" || value == "1"
	case FieldTypeInt:
		if value == "" {
			return 0
		}
		i, _ := strconv.Atoi(value)
		return i
	case FieldTypeFloat:
		if value == "" {
			return 0.0
		}
		f, _ := strconv.ParseFloat(value, 64)
		return f
	case FieldTypeTime:
		// Expect value from <input type="datetime-local"> with layout 2006-01-02T15:04
		if value == "" {
			return nil
		}
		// Try parsing in local time first
		if t, err := time.Parse("2006-01-02T15:04", value); err == nil {
			return t
		}
		// Fallbacks for potential seconds precision
		if t, err := time.Parse("2006-01-02T15:04:05", value); err == nil {
			return t
		}
		// If parsing fails, return the raw string; validator may catch it later
		return value
	default:
		return value
	}
}

// validateFields validates form data
func (h *Handler) validateFields(config *ModelConfig, data map[string]interface{}, isCreate bool) map[string]string {
	errors := make(map[string]string)

	for _, field := range config.Fields {
		if !field.Required || field.Readonly || field.Hidden {
			continue
		}

		// For password fields on update, skip validation if empty (optional on edit)
		if !isCreate && field.Type == FieldTypePassword {
			continue
		}

		value, ok := data[field.Name]
		if !ok || value == "" || value == nil {
			errors[field.Name] = field.Label + " is required"
		}
	}

	return errors
}

// SaveModelOrderSetting saves the model order preference
func (h *Handler) SaveModelOrderSetting(w http.ResponseWriter, r *http.Request) {
	// Parse JSON body
	var request struct {
		Order []string `json:"order"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Save order to database
	if err := h.Registry.SaveModelOrder(request.Order); err != nil {
		utils.Errorf("Failed to save model order: %v", err)
		http.Error(w, "Failed to save order", http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}
