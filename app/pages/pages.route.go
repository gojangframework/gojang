package pages

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"
)

func PageRoutes(handler *PageHandler, sm *scs.SessionManager, client *models.Client) chi.Router {
	r := chi.NewRouter()

	r.Get("/", handler.Home)
	r.Get("/set-language", handler.SetLanguage)

	// Example of a public page route
	// r.Get("/sample", handler.Sample)

	// Protected pages
	r.Group(func(auth chi.Router) {
		auth.Use(middleware.RequireAuth(sm, client))
		auth.Get("/dashboard", handler.Dashboard)

		// Example of a protected page route
		// auth.Get("/sample", handler.Sample)
	})

	return r
}
