package admin

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/app/gojang/http/middleware"
	"github.com/gojangframework/gojang/app/gojang/models"
)

func AdminRoutes(adminHandler *Handler, sm *scs.SessionManager, client *models.Client) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequireAuth(sm, client))
	r.Use(middleware.RequireStaff)
	r.Use(middleware.AuditMiddleware) // Log all admin actions

	// Admin dashboard
	r.Get("/", adminHandler.Dashboard)

	// Admin settings
	r.Post("/settings/model-order", adminHandler.SaveModelOrderSetting)

	// Airtable-style workspace routes
	r.Route("/t/{resource}", func(resource chi.Router) {
		resource.Get("/", adminHandler.Workspace)
		resource.Get("/grid", adminHandler.Grid)
		resource.Get("/records/new", adminHandler.NewRecordDrawer)
		resource.Post("/records", adminHandler.CreateRecord)
		resource.Get("/records/{id}", adminHandler.RecordDrawer)
		resource.Put("/records/{id}", adminHandler.SaveRecord)
		resource.Delete("/records/{id}", adminHandler.DeleteRecord)
		resource.Patch("/records/{id}/fields/{field}", adminHandler.UpdateCell)
	})

	// Legacy model routes redirect into the workspace.
	r.Route("/{model}", func(model chi.Router) {
		model.Get("/", adminHandler.LegacyRedirect)
		model.Get("/new", adminHandler.LegacyRedirect)
		model.Post("/", adminHandler.LegacyRedirect)
		model.Get("/{id}/edit", adminHandler.LegacyRedirect)
		model.Put("/{id}", adminHandler.LegacyRedirect)
		model.Get("/{id}/delete", adminHandler.LegacyRedirect)
		model.Delete("/{id}", adminHandler.LegacyRedirect)
	})

	return r
}
