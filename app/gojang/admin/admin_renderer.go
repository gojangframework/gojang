package admin

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gojangframework/gojang/app/gojang/utils"

	"github.com/gojangframework/gojang/app/gojang/http/middleware"
	"github.com/gojangframework/gojang/app/gojang/models"

	"github.com/justinas/nosurf"
)

//go:embed views
var ViewFiles embed.FS

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
		"iterate": func(start, end int) []int {
			if start > end {
				return []int{}
			}
			result := make([]int, end-start+1)
			for i := range result {
				result[i] = start + i
			}
			return result
		},
		"fieldValue":     extractFieldValue,
		"formatField":    formatFieldForDisplay,
		"getID":          getIDValue,
		"formatDateTime": formatDateTimeField,
	}
	for name, fn := range workspaceTemplateFuncs() {
		funcMap[name] = fn
	}

	templates := make(map[string]*template.Template)

	baseContent, err := ViewFiles.ReadFile("views/admin_base.html")
	if err != nil {
		return nil, fmt.Errorf("reading admin_base.html: %w", err)
	}

	partialContents := map[string]string{}
	err = fs.WalkDir(ViewFiles, "views", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".partial.html") {
			return nil
		}
		content, err := ViewFiles.ReadFile(path)
		if err != nil {
			return err
		}
		partialContents[strings.TrimPrefix(path, "views/")] = string(content)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading admin partials: %w", err)
	}

	// Walk the embedded template directory to find all .html files
	err = fs.WalkDir(ViewFiles, "views", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-html files
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		// Get relative path from views directory
		relPath := strings.TrimPrefix(path, "views/")

		// Skip admin_base.html itself and CSS directory
		if relPath == "admin_base.html" || strings.Contains(relPath, "css/") {
			return nil
		}

		// Determine if this is a fragment (any file with .partial.html)
		isFragment := strings.Contains(relPath, ".partial.html")

		var tmpl *template.Template
		if isFragment {
			// Parse fragment standalone, with sibling partials available for nested templates.
			content, err := ViewFiles.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading admin fragment %s: %w", relPath, err)
			}
			tmpl = template.New(relPath).Funcs(funcMap)
			for partialName, partialContent := range partialContents {
				if partialName == relPath {
					continue
				}
				_, err = tmpl.New(partialName).Parse(partialContent)
				if err != nil {
					return fmt.Errorf("parsing %s: %w", partialName, err)
				}
			}
			tmpl, err = tmpl.Parse(string(content))
			if err != nil {
				return fmt.Errorf("parsing admin fragment %s: %w", relPath, err)
			}
		} else {
			// Parse with admin_base.html
			content, err := ViewFiles.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading admin page %s: %w", relPath, err)
			}

			tmpl, err = template.New("admin_base.html").Funcs(funcMap).Parse(string(baseContent))
			if err != nil {
				return fmt.Errorf("parsing admin_base.html: %w", err)
			}

			for partialName, partialContent := range partialContents {
				_, err = tmpl.New(partialName).Parse(partialContent)
				if err != nil {
					return fmt.Errorf("parsing %s: %w", partialName, err)
				}
			}

			_, err = tmpl.New(relPath).Parse(string(content))
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
		var err error
		if tmpl.Lookup("content") != nil {
			err = tmpl.ExecuteTemplate(w, "content", data)
		} else {
			err = tmpl.ExecuteTemplate(w, name, data)
			if err != nil {
				err = tmpl.Execute(w, data)
			}
		}
		if err != nil {
			utils.Errorf("Partial template execution failed: %v", err)
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
		return field.Bool()
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

// getIDValue extracts the ID field from a struct and returns it as a string
func getIDValue(obj interface{}) string {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}

	idField := v.FieldByName("ID")
	if !idField.IsValid() {
		return ""
	}

	// Handle int IDs (legacy or other models)
	if idField.Kind() == reflect.Int || idField.Kind() == reflect.Int64 {
		return fmt.Sprintf("%d", idField.Int())
	}

	// Handle UUID IDs
	if idField.Type().String() == "uuid.UUID" {
		return fmt.Sprintf("%v", idField.Interface())
	}

	// Fallback: convert to string
	return fmt.Sprintf("%v", idField.Interface())
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

// formatFieldForDisplay formats a field value for display in tables
func formatFieldForDisplay(obj interface{}, fieldName string) string {
	val := extractFieldValue(obj, fieldName)

	// Format boolean values
	if b, ok := val.(bool); ok {
		if b {
			return "✓ Yes"
		}
		return "✗ No"
	}

	// Format time values
	if t, ok := val.(time.Time); ok {
		if t.IsZero() {
			return "-"
		}
		return t.Format("2006-01-02 15:04:05")
	}

	// Return string representation for everything else
	return fmt.Sprintf("%v", val)
}
