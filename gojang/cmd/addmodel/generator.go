package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// generateEntCode runs go generate in the models directory
func generateEntCode(modelsDir string) error {
	if dryRun {
		fmt.Printf("  [DRY-RUN] Would run: go generate in %s\n", modelsDir)
		return nil
	}
	cmd := exec.Command("go", "generate", "./...")
	cmd.Dir = modelsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// createSchema creates the Ent schema file
func createSchema(path, modelName string, fields []Field, includeTimestamps bool) error {
	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("schema file already exists: %s", path)
	}

	imports := `"time"
	
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"`

	// Build fields code
	var fieldsCode strings.Builder
	
	// Add UUID ID field as the first field
	fieldsCode.WriteString("\t\tfield.UUID(\"id\", uuid.UUID{}).\n\t\t\tDefault(uuid.New),\n\t\t\n")
	
	for _, field := range fields {
		fieldsCode.WriteString(fmt.Sprintf("\t\tfield.%s(\"%s\")", getEntFieldType(field.Type), field.Name))

		// Add field modifiers based on type and requirements
		if field.Required {
			// Required field modifiers
			if field.Type == "string" || field.Type == "text" {
				fieldsCode.WriteString(".\n\t\t\tNotEmpty()")
			}
			if field.Type == "float" {
				fieldsCode.WriteString(".\n\t\t\tPositive()")
			}
		} else {
			// Optional field modifiers
			if field.Type == "int" {
				fieldsCode.WriteString(".\n\t\t\tDefault(0)")
			} else if field.Type == "bool" {
				fieldsCode.WriteString(".\n\t\t\tDefault(false)")
			} else {
				// For string, text, float, time - mark as optional
				fieldsCode.WriteString(".\n\t\t\tOptional()")
				if field.Type == "float" {
					// Keep positive constraint even if optional
					fieldsCode.WriteString(".\n\t\t\tPositive()")
				}
			}
		}

		fieldsCode.WriteString(",\n\t\t\n")
	}

	// Add timestamp fields if requested
	if includeTimestamps {
		fieldsCode.WriteString("\t\tfield.Time(\"created_at\").\n\t\t\tDefault(time.Now).\n\t\t\tImmutable(),\n")
	}

	content := fmt.Sprintf(`package schema

import (
	%s
)

type %s struct {
	ent.Schema
}

func (%s) Fields() []ent.Field {
	return []ent.Field{
%s	}
}
`, imports, modelName, modelName, fieldsCode.String())

	return writeFile(path, []byte(content), 0644)
}

// addFormStruct adds a form validation struct to forms.go
func addFormStruct(path, modelName string, fields []Field) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Check if form already exists
	formName := modelName + "Form"
	if strings.Contains(string(content), fmt.Sprintf("type %s struct", formName)) {
		return fmt.Errorf("form struct %s already exists", formName)
	}

	// Build form struct
	var formStruct strings.Builder
	formStruct.WriteString(fmt.Sprintf("\n// %s represents %s create/update form\n", formName, strings.ToLower(modelName)))
	formStruct.WriteString(fmt.Sprintf("type %s struct {\n", formName))

	for _, field := range fields {
		goType := getGoType(field.Type)
		fieldName := toCamelCase(field.Name)
		validation := getValidationTag(field)
		formStruct.WriteString(fmt.Sprintf("\t%s %s `form:\"%s\" validate:\"%s\"`\n", fieldName, goType, field.Name, validation))
	}

	formStruct.WriteString("}\n")

	// Find position to insert (before Validate function)
	validatePos := strings.Index(string(content), "// Validate validates a form struct")
	if validatePos == -1 {
		validatePos = len(content)
	}

	// Insert the form struct
	newContent := string(content[:validatePos]) + formStruct.String() + "\n" + string(content[validatePos:])

	return writeFile(path, []byte(newContent), 0644)
}

// createHandler creates the handler file
func createHandler(path, modelName string, fields []Field) error {
	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("handler file already exists: %s", path)
	}

	modelLower := strings.ToLower(modelName)
	modelPlural := modelLower + "s"
	handlerName := modelName + "Handler"
	modelCamelCase := toCamelCase(modelName)

	// Build form field extraction
	formFieldExtraction := buildFormFieldExtraction(fields)

	// Build field setters for Create
	var createSetters strings.Builder
	for _, field := range fields {
		fieldName := toCamelCase(field.Name)
		setter := fmt.Sprintf("\t\tSet%s(form.%s)", fieldName, fieldName)
		createSetters.WriteString(setter + ".\n")
	}

	// Build field setters for Update
	updateSetters := createSetters.String()

	// Check if we need strconv import (for int or float fields)
	needsStrconv := false
	for _, field := range fields {
		if field.Type == "int" || field.Type == "float" {
			needsStrconv = true
			break
		}
	}

	// Build imports
	var importsBuilder strings.Builder
	importsBuilder.WriteString(`"log"` + "\n\t")
	importsBuilder.WriteString(`"net/http"`)
	if needsStrconv {
		importsBuilder.WriteString("\n\t" + `"strconv"`)
	}
	importsBuilder.WriteString("\n\t" + `"time"` + "\n\n\t")
	importsBuilder.WriteString(`"github.com/go-chi/chi/v5"` + "\n\t")
	importsBuilder.WriteString(`"github.com/google/uuid"` + "\n\t")
	importsBuilder.WriteString(`"github.com/gojangframework/gojang/gojang/models"` + "\n\t")
	importsBuilder.WriteString(`"github.com/gojangframework/gojang/gojang/views/forms"` + "\n\t")
	importsBuilder.WriteString(`"github.com/gojangframework/gojang/gojang/views/renderers"`)
	imports := importsBuilder.String()

	content := fmt.Sprintf(`package handlers

import (
	%s
)

var _ time.Time // to avoid unused import

type %s struct {
	Client   *models.Client
	Renderer *renderers.Renderer
}

func New%s(client *models.Client, renderer *renderers.Renderer) *%s {
	return &%s{
		Client:   client,
		Renderer: renderer,
	}
}

// Index lists all %s
func (h *%s) Index(w http.ResponseWriter, r *http.Request) {
	%s, err := h.Client.%s.Query().
		Order(models.Desc("created_at")).
		All(r.Context())
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to load %s")
		return
	}

	h.Renderer.Render(w, r, "%s/index.html", &renderers.TemplateData{
		Title: "%s",
		Data: map[string]interface{}{
			"%s": %s,
		},
	})
}

// New shows the create form
func (h *%s) New(w http.ResponseWriter, r *http.Request) {
	h.Renderer.Render(w, r, "%s/new.partial.html", &renderers.TemplateData{
		Title: "New %s",
	})
}

// Create creates a new %s
func (h *%s) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form")
		return
	}

	form := forms.%sForm{
%s	}

	if errors := forms.Validate(&form); len(errors) > 0 {
		h.Renderer.Render(w, r, "%s/new.partial.html", &renderers.TemplateData{
			Title: "New %s",
			Data: map[string]interface{}{
				"Form":   form,
				"Errors": errors,
			},
		})
		return
	}

	_, err := h.Client.%s.Create().
%s		Save(r.Context())

	if err != nil {
		log.Printf("Error creating %s: %%v", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to create %s")
		return
	}

	http.Redirect(w, r, "/%s", http.StatusSeeOther)
}

// Edit shows the edit form
func (h *%s) Edit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid ID")
		return
	}

	%s, err := h.Client.%s.Get(r.Context(), id)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "%s not found")
		return
	}

	h.Renderer.Render(w, r, "%s/edit.partial.html", &renderers.TemplateData{
		Title: "Edit %s",
		Data: map[string]interface{}{
			"%s": %s,
		},
	})
}

// Update updates a %s
func (h *%s) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form")
		return
	}

	form := forms.%sForm{
%s	}

	if errors := forms.Validate(&form); len(errors) > 0 {
		%s, _ := h.Client.%s.Get(r.Context(), id)
		h.Renderer.Render(w, r, "%s/edit.partial.html", &renderers.TemplateData{
			Title: "Edit %s",
			Data: map[string]interface{}{
				"%s": %s,
				"Form":     form,
				"Errors":   errors,
			},
		})
		return
	}

	_, err = h.Client.%s.UpdateOneID(id).
%s		Save(r.Context())

	if err != nil {
		log.Printf("Error updating %s: %%v", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to update %s")
		return
	}

	http.Redirect(w, r, "/%s", http.StatusSeeOther)
}

// Delete deletes a %s
func (h *%s) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := h.Client.%s.DeleteOneID(id).Exec(r.Context()); err != nil {
		log.Printf("Error deleting %s: %%v", err)
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to delete %s")
		return
	}

	http.Redirect(w, r, "/%s", http.StatusSeeOther)
}
`,
		// Imports
		imports,
		// Handler struct
		handlerName, handlerName, handlerName, handlerName,
		// Index
		modelPlural, handlerName, modelPlural, modelName, modelPlural, modelPlural, modelName, modelCamelCase, modelPlural,
		// New
		handlerName, modelPlural, modelName,
		// Create
		modelLower, handlerName, modelName, formFieldExtraction, modelPlural, modelName, modelName, createSetters.String(), modelLower, modelLower, modelPlural,
		// Edit
		handlerName, modelLower, modelName, modelName, modelPlural, modelName, modelName, modelLower,
		// Update
		modelLower, handlerName, modelName, formFieldExtraction, modelLower, modelName, modelPlural, modelName, modelName, modelLower, modelName, updateSetters, modelLower, modelLower, modelPlural,
		// Delete
		modelLower, handlerName, modelName, modelLower, modelLower, modelPlural)

	return writeFile(path, []byte(content), 0644)
}

// createRoutes creates the routes file
func createRoutes(path, modelName string) error {
	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("routes file already exists: %s", path)
	}

	handlerName := modelName + "Handler"

	content := fmt.Sprintf(`package routes

import (
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/gojang/http/handlers"
	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/models"
	"github.com/justinas/nosurf"
)

func %sRoutes(handler *handlers.%s, sm *scs.SessionManager, client *models.Client) chi.Router {
	r := chi.NewRouter()
	r.Use(nosurf.NewPure)

	// Public routes
	r.Get("/", handler.Index)

	// Protected routes - auth required
	r.Group(func(auth chi.Router) {
		auth.Use(middleware.RequireAuth(sm, client))

		auth.Get("/new", handler.New)
		auth.Post("/", handler.Create)
		auth.Get("/{id}/edit", handler.Edit)
		auth.Put("/{id}", handler.Update)
		auth.Delete("/{id}", handler.Delete)
	})

	return r
}
`, modelName, handlerName)

	return writeFile(path, []byte(content), 0644)
}

// updateMainGo adds route registration to main.go
func updateMainGo(path, modelName string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	modelLower := strings.ToLower(modelName)
	modelPlural := modelLower + "s"
	handlerName := modelLower + "Handler"

	// Check if handler already registered
	if strings.Contains(string(content), handlerName+" := handlers.New"+modelName+"Handler") {
		return fmt.Errorf("handler %s already registered in main.go", handlerName)
	}

	// Find position to add handler initialization (after postHandler)
	postHandlerPos := strings.Index(string(content), "postHandler := handlers.NewPostHandler")
	if postHandlerPos == -1 {
		return fmt.Errorf("could not find handler initialization section")
	}
	endOfLine := strings.Index(string(content[postHandlerPos:]), "\n")
	insertPos := postHandlerPos + endOfLine + 1

	// Add handler initialization
	handlerCode := fmt.Sprintf("\t%s := handlers.New%sHandler(client, publicRenderer)\n", handlerName, modelName)
	newContent := string(content[:insertPos]) + handlerCode + string(content[insertPos:])

	// Find position to add route mounting (after posts routes)
	postsRoutePos := strings.Index(newContent, `r.Mount("/posts", routes.PostRoutes`)
	if postsRoutePos == -1 {
		return fmt.Errorf("could not find route mounting section")
	}
	endOfLine = strings.Index(newContent[postsRoutePos:], "\n")
	insertPos = postsRoutePos + endOfLine + 1

	// Add route mounting
	routeCode := fmt.Sprintf("\tr.Mount(\"/%s\", routes.%sRoutes(%s, sessionManager, client))\n", modelPlural, modelName, handlerName)
	newContent = newContent[:insertPos] + routeCode + newContent[insertPos:]

	return writeFile(path, []byte(newContent), 0644)
}

// registerWithAdmin registers the model with the admin panel
func registerWithAdmin(path, modelName, modelIcon string, fields []Field) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Check if model already registered
	if strings.Contains(string(content), fmt.Sprintf("ModelType:      &models.%s{}", modelName)) {
		return fmt.Errorf("model %s already registered in admin", modelName)
	}

	// Build list fields
	listFields := []string{"ID"}
	for i, field := range fields {
		if i < 4 { // Show first 4 fields
			listFields = append(listFields, toCamelCase(field.Name))
		}
	}

	listFieldsStr := `"` + strings.Join(listFields, `", "`) + `"`

	// Build optional fields (non-required in generator input)
	optionalFields := []string{}
	for _, f := range fields {
		if !f.Required {
			optionalFields = append(optionalFields, toCamelCase(f.Name))
		}
	}

	// Build registration code with OptionalFields if any
	var b strings.Builder
	b.WriteString("\n\t// Register ")
	b.WriteString(modelName)
	b.WriteString(" model\n\tregistry.RegisterModel(ModelRegistration{\n")
	b.WriteString("\t\tModelType:      &models.")
	b.WriteString(modelName)
	b.WriteString("{},\n")
	b.WriteString("\t\tIcon:           \"")
	b.WriteString(modelIcon)
	b.WriteString("\",\n")
	b.WriteString("\t\tNamePlural:     \"")
	b.WriteString(modelName)
	b.WriteString("s\",\n")
	b.WriteString("\t\tListFields:     []string{")
	b.WriteString(listFieldsStr)
	b.WriteString("},\n")
	b.WriteString("\t\tReadonlyFields: []string{\"ID\", \"CreatedAt\"},\n")
	if len(optionalFields) > 0 {
		b.WriteString("\t\tOptionalFields: []string{")
		b.WriteString("\"")
		b.WriteString(strings.Join(optionalFields, "\", \""))
		b.WriteString("\"},\n")
	}
	b.WriteString("\t})\n")

	registrationCode := b.String()

	// Find position to insert (before closing brace of RegisterModels function)
	closingBracePos := strings.LastIndex(string(content), "}")
	if closingBracePos == -1 {
		return fmt.Errorf("could not find closing brace in models.go")
	}

	// Find the previous line to maintain formatting
	prevNewline := strings.LastIndex(string(content[:closingBracePos]), "\n")
	insertPos := prevNewline + 1

	// Insert the registration code
	newContent := string(content[:insertPos]) + registrationCode + "\n" + string(content[insertPos:])

	return writeFile(path, []byte(newContent), 0644)
}
