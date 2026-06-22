package pages

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/app/gojang/http/middleware"
	"github.com/gojangframework/gojang/app/gojang/models"
)

func PageRoutes(handler *PageHandler, sm *scs.SessionManager, client *models.Client) chi.Router {
	r := chi.NewRouter()

	r.Get("/", handler.Home)
	r.Get("/set-language", handler.SetLanguage)

	// Protected pages
	r.Group(func(auth chi.Router) {
		auth.Use(middleware.RequireAuth(sm, client))
		auth.Get("/dashboard", handler.Dashboard)
	})

	return r
}
