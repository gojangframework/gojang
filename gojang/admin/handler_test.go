package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestNew_WithoutHTMXRequest tests that direct access to New endpoint redirects
func TestNew_WithoutHTMXRequest(t *testing.T) {
	// Setup - use nil client for testing HTMX check only
	registry := &Registry{
		models: make(map[string]*ModelConfig),
	}
	// Use nil renderer since we're only testing the HTMX check logic
	handler := NewHandler(registry, nil, nil)

	// Register a test model directly
	config := &ModelConfig{
		Name:       "TestModel",
		NamePlural: "TestModels",
		Fields:     []FieldConfig{},
	}
	registry.register(config)

	// Create request without HX-Request header
	req := httptest.NewRequest(http.MethodGet, "/admin/testmodel/new", nil)
	w := httptest.NewRecorder()

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("model", "testmodel")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Execute
	handler.New(w, req)

	// Assert redirect
	if w.Code != http.StatusSeeOther {
		t.Errorf("Expected status %d, got %d", http.StatusSeeOther, w.Code)
	}

	location := w.Header().Get("Location")
	expectedLocation := "/admin/testmodel"
	if location != expectedLocation {
		t.Errorf("Expected redirect to %s, got %s", expectedLocation, location)
	}
}

// TestNew_WithHTMXRequest tests that HTMX requests pass through (not redirected)
func TestNew_WithHTMXRequest(t *testing.T) {
	// Setup - use nil client for testing HTMX check only
	registry := &Registry{
		models: make(map[string]*ModelConfig),
	}
	// Use nil renderer since we're only testing the HTMX check logic
	handler := NewHandler(registry, nil, nil)

	// Register a test model directly
	config := &ModelConfig{
		Name:       "TestModel",
		NamePlural: "TestModels",
		Fields:     []FieldConfig{},
	}
	registry.register(config)

	// Create request with HX-Request header
	req := httptest.NewRequest(http.MethodGet, "/admin/testmodel/new", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("model", "testmodel")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Execute with defer/recover to catch expected panic from nil renderer
	defer func() {
		if r := recover(); r != nil {
			// Expected panic from nil renderer - this means HTMX check passed
			// Verify no redirect happened before the panic
			if w.Code == http.StatusSeeOther {
				t.Error("HTMX request should not redirect")
			}
		}
	}()

	handler.New(w, req)

	// If we get here without panic, also check for no redirect
	if w.Code == http.StatusSeeOther {
		t.Error("HTMX request should not redirect")
	}
}

// TestEdit_WithoutHTMXRequest tests that direct access to Edit endpoint redirects
func TestEdit_WithoutHTMXRequest(t *testing.T) {
	// Setup - use nil client for testing HTMX check only
	registry := &Registry{
		models: make(map[string]*ModelConfig),
	}
	// Use nil renderer since we're only testing the HTMX check logic
	handler := NewHandler(registry, nil, nil)

	// Register a test model directly
	config := &ModelConfig{
		Name:       "TestModel",
		NamePlural: "TestModels",
		Fields:     []FieldConfig{},
	}
	registry.register(config)

	// Create request without HX-Request header
	req := httptest.NewRequest(http.MethodGet, "/admin/testmodel/1/edit", nil)
	w := httptest.NewRecorder()

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("model", "testmodel")
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Execute
	handler.Edit(w, req)

	// Assert redirect
	if w.Code != http.StatusSeeOther {
		t.Errorf("Expected status %d, got %d", http.StatusSeeOther, w.Code)
	}

	location := w.Header().Get("Location")
	expectedLocation := "/admin/testmodel"
	if location != expectedLocation {
		t.Errorf("Expected redirect to %s, got %s", expectedLocation, location)
	}
}

// TestEdit_WithHTMXRequest tests that HTMX requests pass through (not redirected)
func TestEdit_WithHTMXRequest(t *testing.T) {
	// Setup - use nil client for testing HTMX check only
	registry := &Registry{
		models: make(map[string]*ModelConfig),
	}
	// Use nil renderer since we're only testing the HTMX check logic
	handler := NewHandler(registry, nil, nil)

	// Register a test model with a query function that returns a mock record
	config := &ModelConfig{
		Name:       "TestModel",
		NamePlural: "TestModels",
		Fields:     []FieldConfig{},
		QueryByID: func(ctx context.Context, id int) (interface{}, error) {
			// Return a simple mock object
			return struct{ ID int }{ID: id}, nil
		},
	}
	registry.register(config)

	// Create request with HX-Request header
	req := httptest.NewRequest(http.MethodGet, "/admin/testmodel/1/edit", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("model", "testmodel")
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Execute with defer/recover to catch expected panic from nil renderer
	defer func() {
		if r := recover(); r != nil {
			// Expected panic from nil renderer - this means HTMX check passed
			// Verify no redirect happened before the panic
			if w.Code == http.StatusSeeOther {
				t.Error("HTMX request should not redirect")
			}
		}
	}()

	handler.Edit(w, req)

	// If we get here without panic, also check for no redirect
	if w.Code == http.StatusSeeOther {
		t.Error("HTMX request should not redirect")
	}
}

// TestDeleteConfirm_WithoutHTMXRequest tests that direct access to DeleteConfirm endpoint redirects
func TestDeleteConfirm_WithoutHTMXRequest(t *testing.T) {
	// Setup - use nil client for testing HTMX check only
	registry := &Registry{
		models: make(map[string]*ModelConfig),
	}
	// Use nil renderer since we're only testing the HTMX check logic
	handler := NewHandler(registry, nil, nil)

	// Register a test model directly
	config := &ModelConfig{
		Name:       "TestModel",
		NamePlural: "TestModels",
		Fields:     []FieldConfig{},
	}
	registry.register(config)

	// Create request without HX-Request header
	req := httptest.NewRequest(http.MethodGet, "/admin/testmodel/1/delete", nil)
	w := httptest.NewRecorder()

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("model", "testmodel")
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Execute
	handler.DeleteConfirm(w, req)

	// Assert redirect
	if w.Code != http.StatusSeeOther {
		t.Errorf("Expected status %d, got %d", http.StatusSeeOther, w.Code)
	}

	location := w.Header().Get("Location")
	expectedLocation := "/admin/testmodel"
	if location != expectedLocation {
		t.Errorf("Expected redirect to %s, got %s", expectedLocation, location)
	}
}

// TestDeleteConfirm_WithHTMXRequest tests that HTMX requests pass through (not redirected)
func TestDeleteConfirm_WithHTMXRequest(t *testing.T) {
	// Setup - use nil client for testing HTMX check only
	registry := &Registry{
		models: make(map[string]*ModelConfig),
	}
	// Use nil renderer since we're only testing the HTMX check logic
	handler := NewHandler(registry, nil, nil)

	// Register a test model with a query function
	config := &ModelConfig{
		Name:       "TestModel",
		NamePlural: "TestModels",
		Fields:     []FieldConfig{},
		ListFields: []string{"ID"},
		QueryByID: func(ctx context.Context, id int) (interface{}, error) {
			// Return a simple mock object
			return struct{ ID int }{ID: id}, nil
		},
	}
	registry.register(config)

	// Create request with HX-Request header
	req := httptest.NewRequest(http.MethodGet, "/admin/testmodel/1/delete", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("model", "testmodel")
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Execute with defer/recover to catch expected panic from nil renderer
	defer func() {
		if r := recover(); r != nil {
			// Expected panic from nil renderer - this means HTMX check passed
			// Verify no redirect happened before the panic
			if w.Code == http.StatusSeeOther {
				t.Error("HTMX request should not redirect")
			}
		}
	}()

	handler.DeleteConfirm(w, req)

	// If we get here without panic, also check for no redirect
	if w.Code == http.StatusSeeOther {
		t.Error("HTMX request should not redirect")
	}
}
