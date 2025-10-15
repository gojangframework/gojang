/*
Sample Product Routes - Demonstration of route configuration for a new data model

This file shows how to configure routes for the SampleProducts resource.
All code is commented out. To use these routes:

1. Ensure the SampleProduct handler is set up (see sample_products.go in handlers)
2. Uncomment all the code in this file
3. Rename this file to "sampleproducts.go" (remove "sample_" prefix)
4. Register these routes in main.go (see comments at the bottom of this file)

See SAMPLE_PRODUCTS_INTEGRATION.md for detailed instructions.
*/

package routes

// Uncomment below to use SampleProduct routes
//
// import (
// 	"github.com/alexedwards/scs/v2"
// 	"github.com/go-chi/chi/v5"
// 	"github.com/gojangframework/gojang/gojang/http/handlers"
// 	"github.com/gojangframework/gojang/gojang/http/middleware"
// 	"github.com/gojangframework/gojang/gojang/models"
// 	"github.com/justinas/nosurf"
// )
//
// // SampleProductRoutes sets up routes for sample product operations
// func SampleProductRoutes(handler *handlers.SampleProductHandler, sm *scs.SessionManager, client *models.Client) chi.Router {
// 	r := chi.NewRouter()
// 	r.Use(nosurf.NewPure)
//
// 	// Public routes
// 	r.Get("/", handler.Index) // Lists all sample products (public)
//
// 	// Protected routes - authentication required
// 	r.Group(func(auth chi.Router) {
// 		auth.Use(middleware.RequireAuth(sm, client))
//
// 		auth.Get("/new", handler.New)       // Show create form
// 		auth.Post("/", handler.Create)      // Create new sample product
// 		auth.Get("/{id}/edit", handler.Edit) // Show edit form
// 		auth.Put("/{id}", handler.Update)    // Update sample product
// 		auth.Delete("/{id}", handler.Delete) // Delete sample product
// 	})
//
// 	return r
// }
//
// // To register these routes in main.go, add:
// // sampleProductHandler := handlers.NewSampleProductHandler(client, publicRenderer)
// // r.Mount("/sampleproducts", routes.SampleProductRoutes(sampleProductHandler, sessionManager, client))
