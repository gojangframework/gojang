package pages

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gojangframework/gojang/app/gojang/views/renderers"
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

// Dashboard renders the user dashboard
func (h *PageHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "dashboard.html", &renderers.TemplateData{
		Title: "Dashboard",
	})
}

// NotFound renders the 404 page
func (h *PageHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	h.Renderer.Render(w, r, "404.html", &renderers.TemplateData{
		Title: "404 Not Found",
	})
}

// SetLanguage stores the user's language preference and returns them to the previous page.
func (h *PageHandler) SetLanguage(w http.ResponseWriter, r *http.Request) {
	lang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))

	supportedLangs := map[string]bool{
		"en":      true,
		"es":      true,
		"ko":      true,
		"ja":      true,
		"zh-hans": true,
		"zh-hant": true,
		"th":      true,
	}
	if !supportedLangs[lang] {
		lang = "en"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "lang",
		Value:    lang,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		SameSite: http.SameSiteLaxMode,
	})

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/"
	}
	if refererURL, err := url.Parse(referer); err == nil {
		query := refererURL.Query()
		query.Del("lang")
		refererURL.RawQuery = query.Encode()
		referer = refererURL.String()
	}

	http.Redirect(w, r, referer, http.StatusSeeOther)
}
