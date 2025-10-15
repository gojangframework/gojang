package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/gojangframework/gojang/gojang/models/post"
	"github.com/gojangframework/gojang/gojang/views/forms"
	"github.com/gojangframework/gojang/gojang/views/renderers"
)

type PostHandler struct {
	Client   *models.Client
	Renderer *renderers.Renderer
}

func NewPostHandler(client *models.Client, renderer *renderers.Renderer) *PostHandler {
	return &PostHandler{
		Client:   client,
		Renderer: renderer,
	}
}

// Index lists all posts
func (h *PostHandler) Index(w http.ResponseWriter, r *http.Request) {
	posts, err := h.Client.Post.Query().
		WithAuthor().
		Order(models.Desc(post.FieldCreatedAt)).
		All(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load posts")
		return
	}

	// Render full page with posts
	h.Renderer.Render(w, r, "posts/index.html", &renderers.TemplateData{
		Title: "Posts",
		Data: map[string]interface{}{
			"Posts": posts,
		},
	})
}

// New shows the create post form
func (h *PostHandler) New(w http.ResponseWriter, r *http.Request) {
	// Prevent direct access - modal forms must be loaded via HTMX
	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, "/posts", http.StatusSeeOther)
		return
	}

	h.Renderer.Render(w, r, "posts/new.partial.html", nil)
}

// Create creates a new post
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	form := forms.PostForm{
		Subject: r.Form.Get("subject"),
		Body:    r.Form.Get("body"),
	}

	// Validate
	errors := forms.Validate(form)
	if len(errors) > 0 {
		h.Renderer.Render(w, r, "posts/new.partial.html", &renderers.TemplateData{
			Errors: errors,
		})
		return
	}

	// Get current user
	user := middleware.GetUser(r.Context())
	if user == nil {
		h.Renderer.RenderError(w, r, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Create post
	_, err := h.Client.Post.Create().
		SetSubject(form.Subject).
		SetBody(form.Body).
		SetAuthor(user).
		Save(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to create post")
		return
	}

	// Close modal and return updated posts list
	w.Header().Set("HX-Trigger", "closeModal")

	// Query all posts to return updated list
	posts, err := h.Client.Post.Query().
		WithAuthor().
		Order(models.Desc(post.FieldCreatedAt)).
		All(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load posts")
		return
	}

	// Return the updated posts list (user site only)
	w.Header().Set("HX-Retarget", "#posts-list")
	w.Header().Set("HX-Reswap", "innerHTML")
	h.Renderer.Render(w, r, "posts/list.partial.html", &renderers.TemplateData{
		Data: map[string]interface{}{
			"Posts": posts,
		},
	})
}

// Edit shows the edit post form
func (h *PostHandler) Edit(w http.ResponseWriter, r *http.Request) {
	// Prevent direct access - modal forms must be loaded via HTMX
	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, "/posts", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid post ID")
		return
	}

	p, err := h.Client.Post.Query().
		Where(post.IDEQ(id)).
		WithAuthor().
		Only(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Post not found")
		return
	}

	// Check if user owns the post or is staff
	// Use already-loaded author edge for better performance
	if p.Edges.Author == nil {
		h.Renderer.RenderError(w, r, http.StatusForbidden, "Unable to verify post ownership")
		return
	}

	if !middleware.OwnsResource(r, p.Edges.Author.ID) {
		h.Renderer.RenderError(w, r, http.StatusForbidden, "You don't have permission to edit this post")
		return
	}

	h.Renderer.Render(w, r, "posts/edit.partial.html", &renderers.TemplateData{
		Data: map[string]interface{}{
			"Post": p,
		},
	})
}

// Update updates a post
func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid post ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}

	form := forms.PostForm{
		Subject: r.Form.Get("subject"),
		Body:    r.Form.Get("body"),
	}

	// Validate
	errors := forms.Validate(form)
	if len(errors) > 0 {
		h.Renderer.Render(w, r, "posts/edit.partial.html", &renderers.TemplateData{
			Errors: errors,
			Data: map[string]interface{}{
				"Post": map[string]interface{}{
					"ID":      id,
					"Subject": form.Subject,
					"Body":    form.Body,
				},
			},
		})
		return
	}

	// Check if user owns the post or is staff
	p, err := h.Client.Post.Query().Where(post.IDEQ(id)).WithAuthor().Only(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Post not found")
		return
	}

	// Use already-loaded author edge
	if p.Edges.Author == nil {
		h.Renderer.RenderError(w, r, http.StatusForbidden, "Unable to verify post ownership")
		return
	}

	if !middleware.OwnsResource(r, p.Edges.Author.ID) {
		h.Renderer.RenderError(w, r, http.StatusForbidden, "You don't have permission to edit this post")
		return
	}

	// Update post
	_, err = h.Client.Post.UpdateOneID(id).
		SetSubject(form.Subject).
		SetBody(form.Body).
		Save(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to update post")
		return
	}

	// Close modal and return updated posts list (user site only)
	w.Header().Set("HX-Trigger", "closeModal")

	// Query all posts to return updated list
	posts, err := h.Client.Post.Query().
		WithAuthor().
		Order(models.Desc(post.FieldCreatedAt)).
		All(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load posts")
		return
	}

	// Return the updated posts list
	w.Header().Set("HX-Retarget", "#posts-list")
	w.Header().Set("HX-Reswap", "innerHTML")
	h.Renderer.Render(w, r, "posts/list.partial.html", &renderers.TemplateData{
		Data: map[string]interface{}{
			"Posts": posts,
		},
	})
}

// DeleteConfirm shows the delete confirmation modal
func (h *PostHandler) DeleteConfirm(w http.ResponseWriter, r *http.Request) {
	// Prevent direct access - modal forms must be loaded via HTMX
	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, "/posts", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid post ID")
		return
	}

	p, err := h.Client.Post.Query().
		Where(post.IDEQ(id)).
		WithAuthor().
		Only(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Post not found")
		return
	}

	// Check if user owns the post or is staff
	// Use already-loaded author edge for better performance
	if p.Edges.Author == nil {
		h.Renderer.RenderError(w, r, http.StatusForbidden, "Unable to verify post ownership")
		return
	}

	if !middleware.OwnsResource(r, p.Edges.Author.ID) {
		h.Renderer.RenderError(w, r, http.StatusForbidden, "You don't have permission to delete this post")
		return
	}

	h.Renderer.Render(w, r, "posts/delete.partial.html", &renderers.TemplateData{
		Data: map[string]interface{}{
			"Post": p,
		},
	})
}

// Delete deletes a post
func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid post ID")
		return
	}

	// Check if user owns the post or is staff
	p, err := h.Client.Post.Query().Where(post.IDEQ(id)).WithAuthor().Only(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Post not found")
		return
	}

	// Use already-loaded author edge
	if p.Edges.Author == nil {
		h.Renderer.RenderError(w, r, http.StatusForbidden, "Unable to verify post ownership")
		return
	}

	if !middleware.OwnsResource(r, p.Edges.Author.ID) {
		h.Renderer.RenderError(w, r, http.StatusForbidden, "You don't have permission to delete this post")
		return
	}

	err = h.Client.Post.DeleteOneID(id).Exec(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to delete post")
		return
	}

	// Close modal and return updated posts list (user site only)
	w.Header().Set("HX-Trigger", "closeModal")

	// Query all posts to return updated list
	posts, err := h.Client.Post.Query().
		WithAuthor().
		Order(models.Desc(post.FieldCreatedAt)).
		All(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load posts")
		return
	}

	// Return the updated posts list
	w.Header().Set("HX-Retarget", "#posts-list")
	w.Header().Set("HX-Reswap", "innerHTML")
	h.Renderer.Render(w, r, "posts/list.partial.html", &renderers.TemplateData{
		Data: map[string]interface{}{
			"Posts": posts,
		},
	})
}
