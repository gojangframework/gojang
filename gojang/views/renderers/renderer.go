package renderers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gojangframework/gojang/gojang/utils"

	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/gojangframework/gojang/gojang/views"
	"github.com/gojangframework/gojang/gojang/views/i18n"

	"github.com/justinas/nosurf"
)

type Renderer struct {
	templates  map[string]*template.Template
	mu         sync.RWMutex // Protects templates map
	debug      bool
	translator *i18n.Translator
}

// TableData is a generic data shape for the reusable "table" component.
type TableData struct {
	Columns      []string
	Rows         []TableRow
	EmptyMessage string
	Pagination   *PaginationData
}

// TableRow contains a single row for the reusable "table" component.
type TableRow struct {
	Cells []interface{}
}

// PaginationData contains pagination metadata for the reusable "table" component.
type PaginationData struct {
	Page       int
	TotalPages int
	TotalCount int
	HasPrev    bool
	HasNext    bool
	PrevURL    string
	NextURL    string
}

// TemplateData holds data for template rendering
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
	Lang        string
}

// NewRenderer creates a new template renderer for public site
func NewRenderer(debug bool) (*Renderer, error) {
	translator, err := i18n.NewTranslator()
	if err != nil {
		return nil, fmt.Errorf("initializing translator: %w", err)
	}

	tmpl, err := parseTemplates(translator)
	if err != nil {
		return nil, err
	}

	return &Renderer{
		templates:  tmpl,
		debug:      debug,
		translator: translator,
	}, nil
}

func parseTemplates(translator *i18n.Translator) (map[string]*template.Template, error) {
	funcMap := TemplateFuncMap(translator)

	templates := make(map[string]*template.Template)

	baseContent, err := views.TemplateFiles.ReadFile("templates/base.html")
	if err != nil {
		return nil, fmt.Errorf("reading base.html: %w", err)
	}
	componentContents, err := readComponentTemplates()
	if err != nil {
		return nil, err
	}

	// Walk the embedded template directory to find all .html files
	err = fs.WalkDir(views.TemplateFiles, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-html files
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		// Get relative path from templates directory
		relPath := strings.TrimPrefix(path, "templates/")

		// Components are parsed into pages and fragments, not rendered as pages.
		if strings.HasPrefix(relPath, "components/") {
			return nil
		}

		// Skip base.html itself
		if relPath == "base.html" {
			return nil
		}

		// Determine if this is a fragment (any file with .partial.html)
		isFragment := strings.Contains(relPath, ".partial.html")

		var tmpl *template.Template
		if isFragment {
			// Parse fragment standalone
			content, err := views.TemplateFiles.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading fragment %s: %w", relPath, err)
			}
			tmpl, err = template.New(relPath).Funcs(funcMap).Parse(string(content))
			if err != nil {
				return fmt.Errorf("parsing fragment %s: %w", relPath, err)
			}
			tmpl, err = parseComponentTemplates(tmpl, componentContents)
			if err != nil {
				return fmt.Errorf("parsing component templates for fragment %s: %w", relPath, err)
			}
		} else {
			// Parse with base.html
			content, err := views.TemplateFiles.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading %s: %w", relPath, err)
			}
			tmpl, err = template.New("base.html").Funcs(funcMap).Parse(string(baseContent))
			if err != nil {
				return fmt.Errorf("parsing base.html: %w", err)
			}
			tmpl, err = parseComponentTemplates(tmpl, componentContents)
			if err != nil {
				return fmt.Errorf("parsing component templates for %s: %w", relPath, err)
			}
			tmpl, err = tmpl.New(relPath).Parse(string(content))
			if err != nil {
				return fmt.Errorf("parsing %s: %w", relPath, err)
			}
		}

		templates[relPath] = tmpl
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking template directory: %w", err)
	}

	return templates, nil
}

// TemplateFuncMap returns the common template functions used by Gojang views.
func TemplateFuncMap(translator *i18n.Translator) template.FuncMap {
	return template.FuncMap{
		"add":       func(a, b int) int { return a + b },
		"sub":       func(a, b int) int { return a - b },
		"subtract":  func(a, b int) int { return a - b },
		"mul":       func(a, b int) int { return a * b },
		"div":       safeDiv,
		"lower":     strings.ToLower,
		"join":      joinStrings,
		"contains":  containsString,
		"hasPrefix": strings.HasPrefix,
		"until":     until,
		"iterate":   iterate,
		"formatDate": func(layout string, t time.Time) string {
			return t.Format(layout)
		},
		"formatNumber": formatNumber,
		"toJSON":       toJSON,
		"t": func(data *TemplateData, key string, args ...interface{}) string {
			lang := "en"
			if data != nil && data.Lang != "" {
				lang = data.Lang
			}
			return translator.Translate(lang, key, args...)
		},
		"tArray": func(data *TemplateData, key string) []string {
			lang := "en"
			if data != nil && data.Lang != "" {
				lang = data.Lang
			}
			return translator.TranslateArray(lang, key)
		},
	}
}

func readComponentTemplates() ([]string, error) {
	componentContents := []string{}
	err := fs.WalkDir(views.TemplateFiles, "templates/components", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == "templates/components" {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		content, err := views.TemplateFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading component %s: %w", strings.TrimPrefix(path, "templates/"), err)
		}
		componentContents = append(componentContents, string(content))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking component templates: %w", err)
	}
	return componentContents, nil
}

func parseComponentTemplates(tmpl *template.Template, componentContents []string) (*template.Template, error) {
	var err error
	for _, content := range componentContents {
		tmpl, err = tmpl.Parse(content)
		if err != nil {
			return nil, err
		}
	}
	return tmpl, nil
}

func safeDiv(a, b int) int {
	if b == 0 {
		return 0
	}
	return a / b
}

func joinStrings(values []string, separator string) string {
	return strings.Join(values, separator)
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func until(count int) []int {
	if count <= 0 {
		return []int{}
	}
	result := make([]int, count)
	for i := range result {
		result[i] = i
	}
	return result
}

func iterate(start, end int) []int {
	if end <= start {
		return []int{}
	}
	result := make([]int, end-start)
	for i := range result {
		result[i] = start + i
	}
	return result
}

func toJSON(v interface{}) template.JS {
	data, err := json.Marshal(v)
	if err != nil {
		return template.JS("null")
	}
	return template.JS(data)
}

func formatNumber(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case float64:
		return fmt.Sprintf("%.2f", v)
	case *float64:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%.2f", *v)
	case float32:
		return fmt.Sprintf("%.2f", v)
	case *float32:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%.2f", *v)
	}

	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%.2f", rv.Float())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%.2f", float64(rv.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%.2f", float64(rv.Uint()))
	default:
		return fmt.Sprint(value)
	}
}

// Render renders a template
func (r *Renderer) Render(w http.ResponseWriter, req *http.Request, name string, data *TemplateData) error {
	data = r.prepareTemplateData(req, data, false)
	r.reloadTemplates()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check if htmx request for partial
	partialName := name + ".partial.html"
	if data.IsHX {
		r.mu.RLock()
		tmpl, ok := r.templates[partialName]
		r.mu.RUnlock()
		if ok {
			return tmpl.ExecuteTemplate(w, partialName, data)
		}
	}

	// Get the template for this page
	r.mu.RLock()
	tmpl, ok := r.templates[name]
	r.mu.RUnlock()
	if !ok {
		utils.Errorf("Template '%s' not found", name)
		return fmt.Errorf("template %s not found", name)
	}

	// Fragment templates (partials) render directly
	isFragment := data.IsHX && strings.Contains(name, ".partial.html")

	if isFragment {
		// Execute the fragment template directly (no base.html wrapper)
		return tmpl.Execute(w, data)
	}

	// For htmx requests to full pages, render only the content block
	if data.IsHX {
		// Execute just the "content" block without base.html wrapper
		return tmpl.ExecuteTemplate(w, "content", data)
	}

	// Execute base.html which will use the blocks defined in the specific template
	err := tmpl.ExecuteTemplate(w, "base.html", data)
	if err != nil {
		utils.Errorf("Template execution failed: %v", err)
	}
	return err
}

// RenderPartial renders a fragment template directly, without requiring an
// HTMX request header. The name may be supplied with or without ".partial.html".
func (r *Renderer) RenderPartial(w http.ResponseWriter, req *http.Request, name string, data *TemplateData) error {
	if !strings.HasSuffix(name, ".partial.html") {
		name += ".partial.html"
	}

	data = r.prepareTemplateData(req, data, true)
	r.reloadTemplates()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	r.mu.RLock()
	tmpl, ok := r.templates[name]
	r.mu.RUnlock()
	if !ok {
		utils.Errorf("Template '%s' not found", name)
		return fmt.Errorf("template %s not found", name)
	}

	return tmpl.ExecuteTemplate(w, name, data)
}

// RenderComponent renders a named template component, such as "table".
func (r *Renderer) RenderComponent(w http.ResponseWriter, req *http.Request, componentName string, data interface{}) error {
	r.reloadTemplates()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	r.mu.RLock()
	var componentTemplate *template.Template
	for _, tmpl := range r.templates {
		if tmpl.Lookup(componentName) != nil {
			componentTemplate = tmpl
			break
		}
	}
	r.mu.RUnlock()

	if componentTemplate != nil {
		return componentTemplate.ExecuteTemplate(w, componentName, data)
	}
	utils.Errorf("Component template '%s' not found", componentName)
	return fmt.Errorf("component template %s not found", componentName)
}

func (r *Renderer) prepareTemplateData(req *http.Request, data *TemplateData, forceHX bool) *TemplateData {
	if data == nil {
		data = &TemplateData{}
	}

	data.CSRFToken = nosurf.Token(req)
	data.User = middleware.GetUser(req.Context())
	data.IsHX = forceHX || req.Header.Get("HX-Request") == "true"
	data.CurrentPath = req.URL.Path
	if data.Lang == "" {
		data.Lang = r.translator.DetectLanguage(req)
	}
	return data
}

func (r *Renderer) reloadTemplates() {
	if !r.debug {
		return
	}

	translator, terr := i18n.NewTranslator()
	if terr != nil {
		return
	}
	tmpl, err := parseTemplates(translator)
	if err != nil {
		return
	}

	r.mu.Lock()
	r.translator = translator
	r.templates = tmpl
	r.mu.Unlock()
}

// RenderError renders an error page
func (r *Renderer) RenderError(w http.ResponseWriter, req *http.Request, status int, message string) {
	w.WriteHeader(status)
	data := &TemplateData{
		Title: fmt.Sprintf("Error %d", status),
		Data: map[string]interface{}{
			"Status":  status,
			"Message": message,
		},
	}
	_ = r.Render(w, req, "error.html", data)
}
