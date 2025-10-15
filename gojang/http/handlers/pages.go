package handlers

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

// Dashboard renders the user dashboard
func (h *PageHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "dashboard.html", &renderers.TemplateData{
		Title: "Dashboard",
	})
}

// Example of a page handler
// func (h *PageHandler) Sample(w http.ResponseWriter, r *http.Request) {
// 	h.Renderer.Render(w, r, "sample-page.html", nil)
// }

// NotFound renders the 404 page
func (h *PageHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	h.Renderer.Render(w, r, "404.html", &renderers.TemplateData{
		Title: "404 Not Found",
	})
}
