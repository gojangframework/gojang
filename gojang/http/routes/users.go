package routes

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/gojang/http/handlers"
	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/justinas/nosurf"
)

func UserRoutes(handler *handlers.UserHandler, sm *scs.SessionManager, client *models.Client) chi.Router {
	r := chi.NewRouter()
	r.Use(nosurf.NewPure)
	r.Use(middleware.RequireAuth(sm, client))
	r.Use(middleware.RequireStaffOrAdmin)

	// All user management routes check permissions in handler
	r.Get("/", handler.Index)
	r.Get("/new", handler.New)
	r.Post("/", handler.Create)
	r.Get("/{id}/edit", handler.Edit)
	r.Get("/{id}/delete", handler.DeleteConfirm)
	r.Put("/{id}", handler.Update)
	r.Delete("/{id}", handler.Delete)

	return r
}
