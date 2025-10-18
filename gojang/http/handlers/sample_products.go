/*
Sample Product Handler - Demonstration of adding a new data model

This file contains a complete example of a handler for managing SampleProducts.
All code is commented out. To use this handler:

1. Create the SampleProduct schema in gojang/models/schema/sampleproduct.go
2. Run `go generate ./...` in the models directory
3. Uncomment all the code in this file
4. Rename this file to "sampleproducts.go" (remove "sample_" prefix)
5. Register routes in main.go

See SAMPLE_PRODUCTS_INTEGRATION.md for detailed instructions.
*/

package handlers

// Uncomment below to use the SampleProduct handler
//
// import (
// 	"net/http"
// 	"strconv"
//
// 	"github.com/go-chi/chi/v5"
// 	"github.com/gojangframework/gojang/gojang/models"
// 	"github.com/gojangframework/gojang/gojang/views/forms"
// 	"github.com/gojangframework/gojang/gojang/views/renderers"
// )
//
// SampleProductHandler handles sample product-related requests
// type SampleProductHandler struct {
// 	Client   *models.Client
// 	Renderer *renderers.Renderer
// }
//
// NewSampleProductHandler creates a new sample product handler
// func NewSampleProductHandler(client *models.Client, renderer *renderers.Renderer) *SampleProductHandler {
// 	return &SampleProductHandler{
// 		Client:   client,
// 		Renderer: renderer,
// 	}
// }
//
// Index lists all sample products
// func (h *SampleProductHandler) Index(w http.ResponseWriter, r *http.Request) {
// Query all sample products from database
// sampleproducts, err := h.Client.SampleProduct.Query().
// 	Order(models.Desc("created_at")).
// 	All(r.Context())
// if err != nil {
// 	h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load sample products")
// 	return
// }
//
// Render sample products index page
// h.Renderer.Render(w, r, "sampleproducts/index.html", &renderers.TemplateData{
// 	Title: "Sample Products",
// 	Data: map[string]interface{}{
// 		"SampleProducts": sampleproducts,
// 	},
// })
// }
//
// New shows the form to create a new sample product
// func (h *SampleProductHandler) New(w http.ResponseWriter, r *http.Request) {
// Render new sample product form (modal)
// h.Renderer.Render(w, r, "sampleproducts/new.partial.html", nil)
// }
//
// Create creates a new sample product
// func (h *SampleProductHandler) Create(w http.ResponseWriter, r *http.Request) {
// Parse form data
// if err := r.ParseForm(); err != nil {
// 	h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
// 	return
// }
//
// Validate form using SampleProductForm
// form := forms.SampleProductForm{
// 	Name:        r.Form.Get("name"),
// 	Description: r.Form.Get("description"),
// 	Price:       r.Form.Get("price"),
// 	Stock:       r.Form.Get("stock"),
// }
//
// Validate form
// errors := forms.Validate(form)
// if len(errors) > 0 {
// 	h.Renderer.Render(w, r, "sampleproducts/new.partial.html", &renderers.TemplateData{
// 		Errors: errors,
// 	})
// 	return
// }
//
// Create sample product in database
// _, err := h.Client.SampleProduct.Create().
// 	SetName(form.Name).
// 	SetDescription(form.Description).
// 	SetPrice(form.Price).
// 	SetStock(form.Stock).
// 	Save(r.Context())
// if err != nil {
// 	h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to create sample product")
// 	return
// }
//
// Redirect to sample products list
// http.Redirect(w, r, "/sampleproducts", http.StatusSeeOther)
// }
//
// Edit shows the form to edit a sample product
// func (h *SampleProductHandler) Edit(w http.ResponseWriter, r *http.Request) {
// Get sample product ID from URL
// idStr := chi.URLParam(r, "id")
// id, err := strconv.Atoi(idStr)
// if err != nil {
// 	h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid sample product ID")
// 	return
// }
//
// Query sample product from database
// sampleproduct, err := h.Client.SampleProduct.Get(r.Context(), id)
// if err != nil {
// 	h.Renderer.RenderError(w, r, http.StatusNotFound, "Sample product not found")
// 	return
// }
//
// Render edit form
// h.Renderer.Render(w, r, "sampleproducts/edit.partial.html", &renderers.TemplateData{
// 	Data: map[string]interface{}{
// 		"SampleProduct": sampleproduct,
// 	},
// })
// }
//
// Update updates a sample product
// func (h *SampleProductHandler) Update(w http.ResponseWriter, r *http.Request) {
// Get sample product ID from URL
// idStr := chi.URLParam(r, "id")
// id, err := strconv.Atoi(idStr)
// if err != nil {
// 	h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid sample product ID")
// 	return
// }
//
// Parse and validate form
// Similar to Create
//
// Update sample product in database
// _, err = h.Client.SampleProduct.UpdateOneID(id).
// 	SetName(form.Name).
// 	SetDescription(form.Description).
// 	SetPrice(form.Price).
// 	SetStock(form.Stock).
// 	Save(r.Context())
// if err != nil {
// 	h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to update sample product")
// 	return
// }
//
// Redirect to sample products list
// http.Redirect(w, r, "/sampleproducts", http.StatusSeeOther)
// }
//
// Delete deletes a sample product
// func (h *SampleProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
// Get sample product ID from URL
// idStr := chi.URLParam(r, "id")
// id, err := strconv.Atoi(idStr)
// if err != nil {
// 	h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid sample product ID")
// 	return
// }
//
// Delete sample product from database
// err = h.Client.SampleProduct.DeleteOneID(id).Exec(r.Context())
// if err != nil {
// 	h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to delete sample product")
// 	return
// }
//
// Redirect to sample products list
// http.Redirect(w, r, "/sampleproducts", http.StatusSeeOther)
// }
