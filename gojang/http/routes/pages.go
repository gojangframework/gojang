package routes

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
