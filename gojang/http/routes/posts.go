package routes

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/gojang/http/handlers"
	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/justinas/nosurf"
)

func PostRoutes(handler *handlers.PostHandler, sm *scs.SessionManager, client *models.Client) chi.Router {
	r := chi.NewRouter()
	r.Use(nosurf.NewPure)

	// Public routes
	r.Get("/", handler.Index) // Lists all posts (public)

	// Protected routes - auth required
	r.Group(func(auth chi.Router) {
		auth.Use(middleware.RequireAuth(sm, client))

		auth.Get("/new", handler.New)
		auth.Post("/", handler.Create)
		auth.Get("/{id}/edit", handler.Edit)            // Handler checks ownership
		auth.Get("/{id}/delete", handler.DeleteConfirm) // Handler checks ownership
		auth.Put("/{id}", handler.Update)               // Handler checks ownership
		auth.Delete("/{id}", handler.Delete)            // Handler checks ownership
	})

	return r
}
