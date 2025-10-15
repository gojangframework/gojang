package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsValidPageName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple name", "About", true},
		{"name with space", "About Us", true},
		{"name with multiple spaces", "Terms of Service", true},
		{"name with numbers", "About123", false},
		{"name with special chars", "About!", false},
		{"name with dash", "About-Us", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidPageName(tt.input)
			if got != tt.want {
				t.Errorf("isValidPageName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"About", "About"},
		{"about", "About"},
		{"About Us", "AboutUs"},
		{"about us", "AboutUs"},
		{"terms of service", "TermsOfService"},
		{"CONTACT", "Contact"},
		{"Contact Us", "ContactUs"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toPascalCase(tt.input)
			if got != tt.want {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCreateTemplateFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test-page.html")

	// Create the template file
	err := createTemplateFile(templatePath, "Test Page", "Test")
	if err != nil {
		t.Fatalf("createTemplateFile failed: %v", err)
	}

	// Read the file
	content, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("Failed to read template file: %v", err)
	}

	// Verify content
	contentStr := string(content)
	expectedStrings := []string{
		`{{define "title"}}Test Page{{end}}`,
		`{{define "content"}}`,
		`<h1>Test Page</h1>`,
		`Welcome to the Test page`,
		`test-page.html`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Template content missing expected string: %q", expected)
		}
	}
}

func TestAddHandler(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	handlerPath := filepath.Join(tmpDir, "pages.go")

	// Create a basic pages.go file
	initialContent := `package handlers

import (
	"net/http"

	"github.com/gojangframework/gojang/gojang/views/renderers"
)

type PageHandler struct {
	Renderer *renderers.Renderer
}

func NewPageHandler(renderer *renderers.Renderer) *PageHandler {
	return &PageHandler{
		Renderer: renderer,
	}
}

// Home renders the home page
func (h *PageHandler) Home(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "home.html", nil)
}

// NotFound renders the 404 page
func (h *PageHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	h.Renderer.Render(w, r, "404.html", &renderers.TemplateData{
		Title: "404 Not Found",
	})
}
`

	err := os.WriteFile(handlerPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add a handler
	err = addHandler(handlerPath, "About", "About Us", "about.html")
	if err != nil {
		t.Fatalf("addHandler failed: %v", err)
	}

	// Read the modified file
	content, err := os.ReadFile(handlerPath)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	contentStr := string(content)

	// Verify the handler was added
	expectedStrings := []string{
		"// About renders the about page",
		"func (h *PageHandler) About(w http.ResponseWriter, r *http.Request) {",
		`h.Renderer.Render(w, r, "about.html"`,
		`Title: "About Us"`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Handler content missing expected string: %q", expected)
		}
	}

	// Verify the handler is before NotFound
	aboutPos := strings.Index(contentStr, "func (h *PageHandler) About")
	notFoundPos := strings.Index(contentStr, "func (h *PageHandler) NotFound")
	if aboutPos >= notFoundPos {
		t.Error("About handler should be placed before NotFound handler")
	}
}

func TestAddHandler_Duplicate(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	handlerPath := filepath.Join(tmpDir, "pages.go")

	// Create a file with an existing handler
	initialContent := `package handlers

import (
	"net/http"

	"github.com/gojangframework/gojang/gojang/views/renderers"
)

type PageHandler struct {
	Renderer *renderers.Renderer
}

// About renders the about page
func (h *PageHandler) About(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "about.html", nil)
}

// NotFound renders the 404 page
func (h *PageHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	h.Renderer.Render(w, r, "404.html", nil)
}
`

	err := os.WriteFile(handlerPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Try to add the same handler again
	err = addHandler(handlerPath, "About", "About Us", "about.html")
	if err == nil {
		t.Error("Expected error when adding duplicate handler, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' error, got: %v", err)
	}
}

func TestAddRoute_Public(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	routesPath := filepath.Join(tmpDir, "pages.go")

	// Create a basic pages.go file
	initialContent := `package routes

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/gojang/http/handlers"
	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/justinas/nosurf"
)

func PageRoutes(handler *handlers.PageHandler, sm *scs.SessionManager, client *models.Client) chi.Router {
	r := chi.NewRouter()
	r.Use(nosurf.NewPure)

	r.Get("/", handler.Home)

	// Protected pages
	r.Group(func(auth chi.Router) {
		auth.Use(middleware.RequireAuth(sm, client))
		auth.Get("/dashboard", handler.Dashboard)
	})

	return r
}
`

	err := os.WriteFile(routesPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add a public route
	err = addRoute(routesPath, "/about", "About", false)
	if err != nil {
		t.Fatalf("addRoute failed: %v", err)
	}

	// Read the modified file
	content, err := os.ReadFile(routesPath)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	contentStr := string(content)

	// Verify the route was added
	if !strings.Contains(contentStr, `r.Get("/about", handler.About)`) {
		t.Error("Route not added correctly")
	}

	// Verify the route is after the home route
	homePos := strings.Index(contentStr, `r.Get("/", handler.Home)`)
	aboutPos := strings.Index(contentStr, `r.Get("/about", handler.About)`)
	if aboutPos <= homePos {
		t.Error("About route should be placed after Home route")
	}
}

func TestAddRoute_Protected(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	routesPath := filepath.Join(tmpDir, "pages.go")

	// Create a basic pages.go file
	initialContent := `package routes

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/gojang/http/handlers"
	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/justinas/nosurf"
)

func PageRoutes(handler *handlers.PageHandler, sm *scs.SessionManager, client *models.Client) chi.Router {
	r := chi.NewRouter()
	r.Use(nosurf.NewPure)

	r.Get("/", handler.Home)

	// Protected pages
	r.Group(func(auth chi.Router) {
		auth.Use(middleware.RequireAuth(sm, client))
		auth.Get("/dashboard", handler.Dashboard)
	})

	return r
}
`

	err := os.WriteFile(routesPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add a protected route
	err = addRoute(routesPath, "/settings", "Settings", true)
	if err != nil {
		t.Fatalf("addRoute failed: %v", err)
	}

	// Read the modified file
	content, err := os.ReadFile(routesPath)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	contentStr := string(content)

	// Verify the route was added
	if !strings.Contains(contentStr, `auth.Get("/settings", handler.Settings)`) {
		t.Error("Protected route not added correctly")
	}

	// Verify the route is in the protected section
	protectedSectionPos := strings.Index(contentStr, "// Protected pages")
	settingsPos := strings.Index(contentStr, `auth.Get("/settings", handler.Settings)`)
	if settingsPos <= protectedSectionPos {
		t.Error("Settings route should be in the protected section")
	}

	// Verify the route is after the dashboard route
	dashboardPos := strings.Index(contentStr, `auth.Get("/dashboard", handler.Dashboard)`)
	if settingsPos <= dashboardPos {
		t.Error("Settings route should be placed after Dashboard route")
	}
}

func TestAddRoute_Duplicate(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	routesPath := filepath.Join(tmpDir, "pages.go")

	// Create a file with an existing route
	initialContent := `package routes

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/gojang/http/handlers"
	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/justinas/nosurf"
)

func PageRoutes(handler *handlers.PageHandler, sm *scs.SessionManager, client *models.Client) chi.Router {
	r := chi.NewRouter()
	r.Use(nosurf.NewPure)

	r.Get("/", handler.Home)
	r.Get("/about", handler.About)

	// Protected pages
	r.Group(func(auth chi.Router) {
		auth.Use(middleware.RequireAuth(sm, client))
		auth.Get("/dashboard", handler.Dashboard)
	})

	return r
}
`

	err := os.WriteFile(routesPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Try to add the same route again
	err = addRoute(routesPath, "/about", "About", false)
	if err == nil {
		t.Error("Expected error when adding duplicate route, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' error, got: %v", err)
	}
}
