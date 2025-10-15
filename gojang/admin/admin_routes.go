package admin

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/justinas/nosurf"
)

func AdminRoutes(adminHandler *Handler, sm *scs.SessionManager, client *models.Client) chi.Router {
	r := chi.NewRouter()
	r.Use(nosurf.NewPure)
	r.Use(middleware.RequireAuth(sm, client))
	r.Use(middleware.RequireStaff)
	r.Use(middleware.AuditMiddleware) // Log all admin actions

	// Admin dashboard
	r.Get("/", adminHandler.Dashboard)

	// Admin settings
	r.Post("/settings/model-order", adminHandler.SaveModelOrderSetting)

	// Generic model routes
	r.Route("/{model}", func(model chi.Router) {
		model.Get("/", adminHandler.Index)                    // List records
		model.Get("/new", adminHandler.New)                   // Show create form
		model.Post("/", adminHandler.Create)                  // Create record
		model.Get("/{id}/edit", adminHandler.Edit)            // Show edit form
		model.Put("/{id}", adminHandler.Update)               // Update record
		model.Get("/{id}/delete", adminHandler.DeleteConfirm) // Show delete confirmation
		model.Delete("/{id}", adminHandler.Delete)            // Delete record
	})

	return r
}
