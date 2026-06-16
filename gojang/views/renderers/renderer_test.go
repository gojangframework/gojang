package renderers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewRendererLoadsEmbeddedTemplates(t *testing.T) {
	renderer, err := NewRenderer(false)
	if err != nil {
		t.Fatalf("NewRenderer(false) returned error: %v", err)
	}

	for _, name := range []string{
		"home.html",
		"posts/list.partial.html",
	} {
		if renderer.templates[name] == nil {
			t.Fatalf("expected embedded template %q to be loaded", name)
		}
	}
}

func TestNewRendererInitializesTranslator(t *testing.T) {
	renderer, err := NewRenderer(false)
	if err != nil {
		t.Fatalf("NewRenderer(false) returned error: %v", err)
	}

	if renderer.translator == nil {
		t.Fatal("expected renderer translator to be initialized")
	}
}

func TestRenderUsesAcceptLanguage(t *testing.T) {
	renderer, err := NewRenderer(false)
	if err != nil {
		t.Fatalf("NewRenderer(false) returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9")
	rec := httptest.NewRecorder()

	if err := renderer.Render(rec, req, "home.html", nil); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `<html lang="es">`) {
		t.Fatalf("expected rendered body to include Spanish lang attribute, got: %s", body)
	}
	if !strings.Contains(body, "Bienvenido a Gojang") {
		t.Fatalf("expected rendered body to include Spanish translation, got: %s", body)
	}
}

func TestRenderTranslatedHTMXPartial(t *testing.T) {
	renderer, err := NewRenderer(false)
	if err != nil {
		t.Fatalf("NewRenderer(false) returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/posts/new", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9")
	rec := httptest.NewRecorder()

	if err := renderer.Render(rec, req, "posts/new.partial.html", nil); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Crear nueva publicacion") {
		t.Fatalf("expected translated partial content, got: %s", body)
	}
}

func TestRenderPartialDoesNotRequireHTMXHeader(t *testing.T) {
	renderer, err := NewRenderer(false)
	if err != nil {
		t.Fatalf("NewRenderer(false) returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/posts/new", nil)
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9")
	rec := httptest.NewRecorder()

	if err := renderer.RenderPartial(rec, req, "posts/new", nil); err != nil {
		t.Fatalf("RenderPartial returned error: %v", err)
	}

	body := rec.Body.String()
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("expected partial-only output, got full page: %s", body)
	}
	if !strings.Contains(body, "Crear nueva publicacion") {
		t.Fatalf("expected translated partial content, got: %s", body)
	}
}

func TestRenderComponentTable(t *testing.T) {
	renderer, err := NewRenderer(false)
	if err != nil {
		t.Fatalf("NewRenderer(false) returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	table := TableData{
		Columns: []string{"Name", "Status"},
		Rows: []TableRow{
			{Cells: []interface{}{"Gojang", "Ready"}},
		},
		Pagination: &PaginationData{
			Page:       1,
			TotalPages: 2,
			TotalCount: 3,
			HasNext:    true,
			NextURL:    "?page=2",
		},
	}

	if err := renderer.RenderComponent(rec, req, "table", table); err != nil {
		t.Fatalf("RenderComponent returned error: %v", err)
	}

	body := rec.Body.String()
	for _, want := range []string{"<table", "Gojang", "Ready", "Page 1 of 2", `href="?page=2"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected rendered table to contain %q, got: %s", want, body)
		}
	}
}

func TestTemplateFuncMapReusableFunctions(t *testing.T) {
	renderer, err := NewRenderer(false)
	if err != nil {
		t.Fatalf("NewRenderer(false) returned error: %v", err)
	}

	tmpl, err := template.New("funcs").Funcs(TemplateFuncMap(renderer.translator)).Parse(`
{{join .Words "/"}}
{{hasPrefix .Path "/admin"}}
{{range until 3}}{{.}}{{end}}
{{range iterate 2 5}}{{.}}{{end}}
{{formatDate "2006-01-02" .Date}}
{{formatNumber .Amount}}
<script>const payload = {{toJSON .Payload}};</script>
`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	var out strings.Builder
	err = tmpl.Execute(&out, map[string]interface{}{
		"Words":  []string{"alpha", "beta"},
		"Path":   "/admin/users",
		"Date":   time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC),
		"Amount": 12.5,
		"Payload": map[string]string{
			"name": "gojang",
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	body := out.String()
	for _, want := range []string{"alpha/beta", "true", "012", "234", "2026-06-16", "12.50", `"name":"gojang"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected output to contain %q, got: %s", want, body)
		}
	}
}

// Test template function: add
func TestTemplateFuncAdd(t *testing.T) {
	add := func(a, b int) int { return a + b }

	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive numbers", 5, 3, 8},
		{"negative numbers", -5, -3, -8},
		{"mixed numbers", 10, -5, 5},
		{"zero", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := add(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("add(%d, %d) = %d; expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Test template function: sub
func TestTemplateFuncSub(t *testing.T) {
	sub := func(a, b int) int { return a - b }

	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive result", 10, 3, 7},
		{"negative result", 3, 10, -7},
		{"zero result", 5, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sub(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("sub(%d, %d) = %d; expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Test template function: mul
func TestTemplateFuncMul(t *testing.T) {
	mul := func(a, b int) int { return a * b }

	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive numbers", 5, 3, 15},
		{"negative numbers", -5, 3, -15},
		{"zero", 5, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mul(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("mul(%d, %d) = %d; expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Test template function: div
func TestTemplateFuncDiv(t *testing.T) {
	div := func(a, b int) int {
		if b == 0 {
			return 0
		}
		return a / b
	}

	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"normal division", 10, 2, 5},
		{"division by zero", 10, 0, 0},
		{"negative dividend", -10, 2, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := div(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("div(%d, %d) = %d; expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Test template function: lower
func TestTemplateFuncLower(t *testing.T) {
	lower := func(s string) string {
		return s
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"uppercase", "HELLO", "HELLO"},
		{"mixed case", "HeLLo", "HeLLo"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lower(tt.input)
			if result != tt.expected {
				t.Errorf("lower(%q) = %q; expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test template function: contains
func TestTemplateFuncContains(t *testing.T) {
	contains := func(slice []string, item string) bool {
		for _, s := range slice {
			if s == item {
				return true
			}
		}
		return false
	}

	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"item exists", []string{"apple", "banana", "cherry"}, "banana", true},
		{"item not exists", []string{"apple", "banana", "cherry"}, "grape", false},
		{"empty slice", []string{}, "apple", false},
		{"case sensitive", []string{"Apple", "Banana"}, "apple", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("contains(%v, %q) = %v; expected %v", tt.slice, tt.item, result, tt.expected)
			}
		})
	}
}
