package admin

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gojangframework/gojang/gojang/utils"

	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"

	"github.com/justinas/nosurf"
)

// TemplateData holds data for admin template rendering
type TemplateData struct {
	Title       string
	Data        map[string]interface{}
	User        *models.User
	CSRFToken   string
	IsHX        bool
	Errors      map[string]string
	CurrentPath string
	Flash       string
	FlashType   string
}

type AdminRenderer struct {
	templates map[string]*template.Template
	mu        sync.RWMutex // Protects templates map
	debug     bool
}

// NewAdminRenderer creates a new template renderer for admin panel
// Admin templates are ALWAYS rendered as fragments (no base.html wrapper)
func NewAdminRenderer(debug bool) (*AdminRenderer, error) {
	tmpl, err := parseAdminTemplates()
	if err != nil {
		return nil, err
	}

	return &AdminRenderer{
		templates: tmpl,
		debug:     debug,
	}, nil
}

func parseAdminTemplates() (map[string]*template.Template, error) {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"contains": func(slice []string, item string) bool {
			for _, s := range slice {
				if s == item {
					return true
				}
			}
			return false
		},
		"fieldValue":     extractFieldValue,
		"getID":          getIDValue,
		"formatDateTime": formatDateTimeField,
	}

	templates := make(map[string]*template.Template)
	templateDir := "./gojang/admin/views"
	basePath := filepath.Join(templateDir, "admin_base.html")

	// Walk the template directory to find all .html files
	err := filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-html files
		if info.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		// Get relative path from templateDir
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}

		// Normalize path separators to forward slashes for cross-platform compatibility
		relPath = filepath.ToSlash(relPath)

		// Skip admin_base.html itself and CSS directory
		if relPath == "admin_base.html" || strings.Contains(relPath, "css/") {
			return nil
		}

		// Determine if this is a fragment (any file with .partial.html)
		isFragment := strings.Contains(relPath, ".partial.html")

		var tmpl *template.Template
		if isFragment {
			// Parse fragment standalone
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading admin fragment %s: %w", relPath, err)
			}
			tmpl, err = template.New(relPath).Funcs(funcMap).Parse(string(content))
			if err != nil {
				return fmt.Errorf("parsing admin fragment %s: %w", relPath, err)
			}
		} else {
			// Parse with admin_base.html
			files := []string{basePath, path}

			// For model_index.html, also include the partial
			if relPath == "model_index.html" {
				partialPath := filepath.Join(templateDir, "model_list.partial.html")
				if _, err := os.Stat(partialPath); err == nil {
					files = append(files, partialPath)
				}
			}

			tmpl, err = template.New(filepath.Base(basePath)).Funcs(funcMap).ParseFiles(files...)
			if err != nil {
				return fmt.Errorf("parsing admin page %s: %w", relPath, err)
			}
		}

		templates[relPath] = tmpl
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking admin template directory: %w", err)
	}

	return templates, nil
}

// Render renders an admin template
func (r *AdminRenderer) Render(w http.ResponseWriter, req *http.Request, name string, data *TemplateData) error {
	if data == nil {
		data = &TemplateData{}
	}

	// Add CSRF token
	data.CSRFToken = nosurf.Token(req)

	// Add user if authenticated
	data.User = middleware.GetUser(req.Context())

	// Check if htmx request
	data.IsHX = req.Header.Get("HX-Request") == "true"
	data.CurrentPath = req.URL.Path

	// Reload templates in debug mode
	if r.debug {
		tmpl, err := parseAdminTemplates()
		if err == nil {
			r.mu.Lock()
			r.templates = tmpl
			r.mu.Unlock()
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Get the template
	r.mu.RLock()
	tmpl, ok := r.templates[name]
	r.mu.RUnlock()
	if !ok {
		utils.Errorf("Admin template '%s' not found", name)
		return fmt.Errorf("admin template %s not found", name)
	}

	// Fragment templates (partials) render directly
	isFragment := strings.Contains(name, ".partial.html")

	if isFragment {
		// Partials can define a "content" block or just render directly
		// Try content block first, fallback to direct execution
		err := tmpl.ExecuteTemplate(w, "content", data)
		if err != nil {
			// If content block doesn't exist, execute directly
			err = tmpl.Execute(w, data)
			if err != nil {
				utils.Errorf("Partial template execution failed: %v", err)
			}
		}
		return err
	}

	// Full page templates render with admin_base.html
	err := tmpl.ExecuteTemplate(w, "admin_base.html", data)
	if err != nil {
		utils.Errorf("Admin template execution failed: %v", err)
	}
	return err
}

// RenderError renders an error message (as a simple fragment)
func (r *AdminRenderer) RenderError(w http.ResponseWriter, req *http.Request, status int, message string) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<div class="error">
		<h2>Error %d</h2>
		<p>%s</p>
	</div>`, status, message)
}

// extractFieldValue extracts a field value from a struct using reflection
func extractFieldValue(obj interface{}, fieldName string) interface{} {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return ""
	}

	// Handle different types
	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint()
	case reflect.Bool:
		if field.Bool() {
			return "Yes"
		}
		return "No"
	case reflect.Struct:
		// Handle time.Time
		if field.Type().String() == "time.Time" {
			t := field.Interface().(time.Time)
			if t.IsZero() {
				return "-"
			}
			return t.Format("2006-01-02 15:04:05")
		}
		return field.Interface()
	case reflect.Ptr:
		if field.IsNil() {
			return "-"
		}
		return extractFieldValue(field.Interface(), fieldName)
	default:
		return field.Interface()
	}
}

// getIDValue extracts the ID field from a struct
func getIDValue(obj interface{}) int {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return 0
	}

	idField := v.FieldByName("ID")
	if !idField.IsValid() {
		return 0
	}

	if idField.Kind() == reflect.Int || idField.Kind() == reflect.Int64 {
		return int(idField.Int())
	}

	return 0
}

// formatDateTimeField extracts a time field and formats it for datetime-local input
func formatDateTimeField(obj interface{}, fieldName string) string {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return ""
	}

	// Check if it's a time.Time field
	if field.Type().String() == "time.Time" {
		t := field.Interface().(time.Time)
		if t.IsZero() {
			return ""
		}
		return t.Format("2006-01-02T15:04")
	}

	return ""
}
