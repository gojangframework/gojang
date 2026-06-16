package renderers

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"sync"

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

	templates := make(map[string]*template.Template)

	baseContent, err := views.TemplateFiles.ReadFile("templates/base.html")
	if err != nil {
		return nil, fmt.Errorf("reading base.html: %w", err)
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

// Render renders a template
func (r *Renderer) Render(w http.ResponseWriter, req *http.Request, name string, data *TemplateData) error {
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
	if data.Lang == "" {
		data.Lang = r.translator.DetectLanguage(req)
	}

	// Reload templates in debug mode
	if r.debug {
		translator, terr := i18n.NewTranslator()
		if terr == nil {
			r.translator = translator
		}
		tmpl, err := parseTemplates(r.translator)
		if err == nil {
			r.mu.Lock()
			r.templates = tmpl
			r.mu.Unlock()
		}
	}

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
