package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gojangframework/gojang/app/gojang/utils"
	"github.com/google/uuid"

	"github.com/go-chi/chi/v5"
	"github.com/gojangframework/gojang/app/gojang/models"
)

// Handler handles all admin panel requests
type Handler struct {
	Registry *Registry
	Renderer *AdminRenderer
	DB       *models.Client
}

type resourcePageData struct {
	Config         *ModelConfig
	Records        []interface{}
	View           string
	Page           int
	PerPage        int
	TotalPages     int
	TotalCount     int
	Resources      []*ModelConfig
	Fields         []FieldConfig
	AllFields      []FieldConfig
	SelectedFields map[string]bool
	Filter         GridFilter
	Sort           GridSort
	Related        RelatedFilter
}

type GridFilter struct {
	Field FieldConfig
	Op    string
	Value string
	Valid bool
}

type GridSort struct {
	Field FieldConfig
	Dir   string
	Valid bool
}

type RelatedFilter struct {
	SourceResource string
	SourceField    FieldConfig
	SourceID       uuid.UUID
	SourceLabel    string
	Valid          bool
}

const (
	workspaceViewOverview = "overview"
	workspaceViewGrid     = "grid"
)

// NewHandler creates a new admin handler
func NewHandler(registry *Registry, renderer *AdminRenderer, db *models.Client) *Handler {
	return &Handler{
		Registry: registry,
		Renderer: renderer,
		DB:       db,
	}
}

// Dashboard opens the admin workspace at the first available resource.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	models := h.Registry.List()

	if len(models) > 0 {
		if r.Header.Get("HX-Request") == "true" {
			h.renderWorkspace(w, r, models[0].Name)
			return
		}
		h.renderWorkspace(w, r, models[0].Name)
		return
	}

	h.Renderer.Render(w, r, "admin_main.html", &TemplateData{
		Title: "Admin Workspace",
		Data: map[string]interface{}{
			"Models": models,
		},
	})
}

// Workspace renders the Airtable-style admin workspace for a resource.
func (h *Handler) Workspace(w http.ResponseWriter, r *http.Request) {
	h.renderWorkspace(w, r, chi.URLParam(r, "resource"))
}

// Grid renders only the data grid partial for a resource.
func (h *Handler) Grid(w http.ResponseWriter, r *http.Request) {
	data, err := h.resourcePageData(r, chi.URLParam(r, "resource"))
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, err.Error())
		return
	}
	data.View = workspaceViewGrid
	h.Renderer.Render(w, r, "workspace_grid.partial.html", &TemplateData{
		Title: data.Config.NamePlural,
		Data:  map[string]interface{}{"Grid": data},
	})
}

// RecordDrawer renders the full record editor drawer.
func (h *Handler) RecordDrawer(w http.ResponseWriter, r *http.Request) {
	config, ok := h.publicResourceConfig(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid ID")
		return
	}
	record, err := config.QueryByID(r.Context(), id)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, config.Name+" not found")
		return
	}
	h.Renderer.Render(w, r, "record_drawer.partial.html", &TemplateData{
		Title: "Edit " + config.Name,
		Data: map[string]interface{}{
			"Config":    config,
			"Record":    record,
			"Page":      currentGridPage(r),
			"PerPage":   currentGridPerPage(r),
			"GridQuery": currentGridQuery(r),
		},
	})
}

// UpdateCell updates one editable scalar field and returns the rendered cell.
func (h *Handler) UpdateCell(w http.ResponseWriter, r *http.Request) {
	config, field, id, ok := h.lookupMutationTarget(w, r)
	if !ok {
		return
	}
	record, err := config.QueryByID(r.Context(), id)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, config.Name+" not found")
		return
	}
	if !canEditRecordField(config, record, field) {
		h.Renderer.RenderError(w, r, http.StatusForbidden, "Field is protected")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid form data")
		return
	}
	value, err := h.parseFieldValueStrict(field, submittedFieldValue(r, "value"))
	if err != nil {
		h.renderCellError(w, r, config, field, id, err.Error())
		return
	}
	if err := config.UpdateFunc(r.Context(), id, map[string]interface{}{field.Name: value}); err != nil {
		h.renderCellError(w, r, config, field, id, err.Error())
		return
	}
	record, err = config.QueryByID(r.Context(), id)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, "Failed to reload record")
		return
	}
	h.Renderer.Render(w, r, "grid_cell.partial.html", &TemplateData{
		Data: map[string]interface{}{
			"Config": config,
			"Field":  field,
			"Record": record,
		},
	})
}

// SaveRecord updates a record from the drawer and refreshes the grid.
func (h *Handler) SaveRecord(w http.ResponseWriter, r *http.Request) {
	config, ok := h.publicResourceConfig(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid ID")
		return
	}
	record, err := config.QueryByID(r.Context(), id)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, config.Name+" not found")
		return
	}
	data, errors := h.editableFormData(r, config, false)
	if len(errors) > 0 {
		w.Header().Set("HX-Retarget", "#record-drawer")
		w.Header().Set("HX-Reswap", "innerHTML")
		h.Renderer.Render(w, r, "record_drawer.partial.html", &TemplateData{
			Title:  "Edit " + config.Name,
			Errors: errors,
			Data: map[string]interface{}{
				"Config":    config,
				"Record":    record,
				"Page":      currentGridPage(r),
				"PerPage":   currentGridPerPage(r),
				"GridQuery": currentGridQuery(r),
			},
		})
		return
	}
	if err := h.rejectProtectedRecordMutations(config, record, data); err != nil {
		w.Header().Set("HX-Retarget", "#record-drawer")
		w.Header().Set("HX-Reswap", "innerHTML")
		h.Renderer.Render(w, r, "record_drawer.partial.html", &TemplateData{
			Title:  "Edit " + config.Name,
			Errors: map[string]string{"_general": err.Error()},
			Data: map[string]interface{}{
				"Config":    config,
				"Record":    record,
				"Page":      currentGridPage(r),
				"PerPage":   currentGridPerPage(r),
				"GridQuery": currentGridQuery(r),
			},
		})
		return
	}
	if err := config.UpdateFunc(r.Context(), id, data); err != nil {
		w.Header().Set("HX-Retarget", "#record-drawer")
		w.Header().Set("HX-Reswap", "innerHTML")
		h.Renderer.Render(w, r, "record_drawer.partial.html", &TemplateData{
			Title:  "Edit " + config.Name,
			Errors: map[string]string{"_general": err.Error()},
			Data: map[string]interface{}{
				"Config":    config,
				"Record":    record,
				"Page":      currentGridPage(r),
				"PerPage":   currentGridPerPage(r),
				"GridQuery": currentGridQuery(r),
			},
		})
		return
	}
	w.Header().Set("HX-Trigger", "closeRecordDrawer")
	h.Grid(w, r)
}

// CreateRecord creates a record from the drawer/new-row form and refreshes the grid.
func (h *Handler) CreateRecord(w http.ResponseWriter, r *http.Request) {
	config, ok := h.publicResourceConfig(w, r)
	if !ok {
		return
	}
	data, errors := h.editableFormData(r, config, true)
	if len(errors) > 0 {
		w.Header().Set("HX-Retarget", "#record-drawer")
		w.Header().Set("HX-Reswap", "innerHTML")
		h.Renderer.Render(w, r, "record_drawer.partial.html", &TemplateData{
			Title:  "New " + config.Name,
			Errors: errors,
			Data: map[string]interface{}{
				"Config":    config,
				"Page":      currentGridPage(r),
				"PerPage":   currentGridPerPage(r),
				"GridQuery": currentGridQuery(r),
			},
		})
		return
	}
	if err := h.rejectProtectedRecordMutations(config, nil, data); err != nil {
		w.Header().Set("HX-Retarget", "#record-drawer")
		w.Header().Set("HX-Reswap", "innerHTML")
		h.Renderer.Render(w, r, "record_drawer.partial.html", &TemplateData{
			Title:  "New " + config.Name,
			Errors: map[string]string{"_general": err.Error()},
			Data: map[string]interface{}{
				"Config":    config,
				"Page":      currentGridPage(r),
				"PerPage":   currentGridPerPage(r),
				"GridQuery": currentGridQuery(r),
			},
		})
		return
	}
	if _, err := config.CreateFunc(r.Context(), data); err != nil {
		w.Header().Set("HX-Retarget", "#record-drawer")
		w.Header().Set("HX-Reswap", "innerHTML")
		h.Renderer.Render(w, r, "record_drawer.partial.html", &TemplateData{
			Title:  "New " + config.Name,
			Errors: map[string]string{"_general": err.Error()},
			Data: map[string]interface{}{
				"Config":    config,
				"Page":      currentGridPage(r),
				"PerPage":   currentGridPerPage(r),
				"GridQuery": currentGridQuery(r),
			},
		})
		return
	}
	w.Header().Set("HX-Trigger", "closeRecordDrawer")
	h.Grid(w, r)
}

// NewRecordDrawer renders an empty drawer for record creation.
func (h *Handler) NewRecordDrawer(w http.ResponseWriter, r *http.Request) {
	config, ok := h.publicResourceConfig(w, r)
	if !ok {
		return
	}
	h.Renderer.Render(w, r, "record_drawer.partial.html", &TemplateData{
		Title: "New " + config.Name,
		Data: map[string]interface{}{
			"Config":    config,
			"Page":      currentGridPage(r),
			"PerPage":   currentGridPerPage(r),
			"GridQuery": currentGridQuery(r),
		},
	})
}

// DeleteRecord deletes a record and refreshes the grid.
func (h *Handler) DeleteRecord(w http.ResponseWriter, r *http.Request) {
	config, ok := h.publicResourceConfig(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid ID")
		return
	}
	record, err := config.QueryByID(r.Context(), id)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, config.Name+" not found")
		return
	}
	if isProtectedAdminSettingRecord(config, record) {
		h.Renderer.RenderError(w, r, http.StatusForbidden, "Protected admin preferences cannot be deleted")
		return
	}
	if err := config.DeleteFunc(r.Context(), id); err != nil {
		h.Renderer.RenderError(w, r, http.StatusInternalServerError, fmt.Sprintf("Failed to delete %s", config.Name))
		return
	}
	w.Header().Set("HX-Trigger", "closeRecordDrawer")
	h.Grid(w, r)
}

// LegacyRedirect redirects old /admin/{model} URLs into the workspace route.
func (h *Handler) LegacyRedirect(w http.ResponseWriter, r *http.Request) {
	modelName := strings.Trim(chi.URLParam(r, "model"), "/")
	if modelName == "" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/t/"+modelName, http.StatusSeeOther)
}

func (h *Handler) renderWorkspace(w http.ResponseWriter, r *http.Request, resourceName string) {
	data, err := h.resourcePageData(r, resourceName)
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusNotFound, err.Error())
		return
	}
	h.Renderer.Render(w, r, "workspace.html", &TemplateData{
		Title: data.Config.NamePlural,
		Data: map[string]interface{}{
			"Grid":      data,
			"Resources": data.Resources,
			"Config":    data.Config,
		},
	})
}

func (h *Handler) resourcePageData(r *http.Request, resourceName string) (*resourcePageData, error) {
	config, err := h.Registry.Get(resourceName)
	if err != nil {
		return nil, err
	}
	if config.Internal {
		return nil, fmt.Errorf("model %s not found", resourceName)
	}
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	perPage := parseAllowedPerPage(r.URL.Query().Get("per_page"), 50)
	offset := (page - 1) * perPage
	allFields := allWorkspaceFields(config)
	defaultFields := workspaceFields(config)
	selectedFields := parseSelectedGridFields(r.URL.Query()["fields"], allFields, defaultFields)
	filter := parseGridFilter(r, allFields)
	sortState := parseGridSort(r, allFields)
	related := h.parseRelatedFilter(r, config)

	records, totalCount, err := h.queryWorkspaceRecords(r, config, filter, sortState, related, perPage, offset)
	if err != nil {
		return nil, err
	}
	totalPages := (totalCount + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	return &resourcePageData{
		Config:         config,
		Records:        records,
		View:           parseWorkspaceView(r.URL.Query().Get("view")),
		Page:           page,
		PerPage:        perPage,
		TotalPages:     totalPages,
		TotalCount:     totalCount,
		Resources:      h.Registry.List(),
		Fields:         fieldsBySelection(allFields, selectedFields),
		AllFields:      allFields,
		SelectedFields: selectedFields,
		Filter:         filter,
		Sort:           sortState,
		Related:        related,
	}, nil
}

func (h *Handler) queryWorkspaceRecords(r *http.Request, config *ModelConfig, filter GridFilter, sortState GridSort, related RelatedFilter, limit, offset int) ([]interface{}, int, error) {
	if related.Valid {
		records, totalCount, pushedDown, err := h.tryQueryRelatedRecordsPaginated(r, related, filter, sortState, limit, offset)
		if err != nil {
			return nil, 0, err
		}
		if pushedDown {
			return records, totalCount, nil
		}
		records, err = h.queryRelatedRecords(r, related)
		if err != nil {
			return nil, 0, err
		}
		if filter.Valid {
			records = filterRecords(records, filter)
		}
		if sortState.Valid {
			sortRecords(records, sortState)
		}
		totalCount = len(records)
		return paginateRecords(records, totalCount, limit, offset), totalCount, nil
	}

	if !filter.Valid && !sortState.Valid {
		totalCount, err := config.CountAll(r.Context())
		if err != nil {
			return nil, 0, err
		}
		records, err := config.QueryAllPaginated(r.Context(), limit, offset)
		return records, totalCount, err
	}
	if config.QueryAll == nil {
		return nil, 0, fmt.Errorf("resource %s does not support filtered grid queries", config.Name)
	}
	records, err := config.QueryAll(r.Context())
	if err != nil {
		return nil, 0, err
	}
	if filter.Valid {
		records = filterRecords(records, filter)
	}
	if sortState.Valid {
		sortRecords(records, sortState)
	}
	totalCount := len(records)
	return paginateRecords(records, totalCount, limit, offset), totalCount, nil
}

func paginateRecords(records []interface{}, totalCount, limit, offset int) []interface{} {
	end := offset + limit
	if offset > totalCount {
		return []interface{}{}
	}
	if end > totalCount {
		end = totalCount
	}
	return records[offset:end]
}

func (h *Handler) parseRelatedFilter(r *http.Request, targetConfig *ModelConfig) RelatedFilter {
	sourceResource := strings.TrimSpace(r.URL.Query().Get("related_from"))
	sourceFieldName := strings.TrimSpace(r.URL.Query().Get("related_field"))
	sourceIDRaw := strings.TrimSpace(r.URL.Query().Get("related_id"))
	if sourceResource == "" || sourceFieldName == "" || sourceIDRaw == "" || targetConfig == nil {
		return RelatedFilter{}
	}
	sourceID, err := uuid.Parse(sourceIDRaw)
	if err != nil {
		return RelatedFilter{}
	}
	sourceConfig, err := h.Registry.Get(sourceResource)
	if err != nil || sourceConfig.Internal {
		return RelatedFilter{}
	}
	sourceField, ok := findField(sourceConfig, sourceFieldName)
	if !ok || !sourceField.Relation || sourceField.RelationTarget != targetConfig.Name {
		return RelatedFilter{}
	}
	sourceLabel := ""
	if sourceConfig.QueryByID != nil {
		if sourceRecord, err := sourceConfig.QueryByID(r.Context(), sourceID); err == nil {
			sourceLabel = cellLabel(sourceRecord)
		}
	}
	return RelatedFilter{
		SourceResource: sourceConfig.Name,
		SourceField:    sourceField,
		SourceID:       sourceID,
		SourceLabel:    sourceLabel,
		Valid:          true,
	}
}

func (h *Handler) queryRelatedRecords(r *http.Request, related RelatedFilter) ([]interface{}, error) {
	records, _, err := h.queryRelatedRecordsWithMode(r, related, 0, 0, false)
	return records, err
}

func (h *Handler) queryRelatedRecordsPaginated(r *http.Request, related RelatedFilter, limit, offset int) ([]interface{}, int, error) {
	records, totalCount, _, err := h.tryQueryRelatedRecordsPaginated(r, related, GridFilter{}, GridSort{}, limit, offset)
	return records, totalCount, err
}

func (h *Handler) queryRelatedRecordsWithMode(r *http.Request, related RelatedFilter, limit, offset int, paginated bool) ([]interface{}, int, error) {
	sourceRecord, targetConfig, err := h.relatedQueryContext(r, related)
	if err != nil {
		return nil, 0, err
	}
	newQuery := h.relatedQueryFactory(r, related, sourceRecord, targetConfig)
	query, err := newQuery()
	if err != nil {
		return nil, 0, err
	}
	records, err := callQueryAll(r.Context(), query, related)
	if err != nil {
		return nil, 0, err
	}
	return records, len(records), nil
}

func (h *Handler) tryQueryRelatedRecordsPaginated(r *http.Request, related RelatedFilter, filter GridFilter, sortState GridSort, limit, offset int) ([]interface{}, int, bool, error) {
	sourceRecord, targetConfig, err := h.relatedQueryContext(r, related)
	if err != nil {
		return nil, 0, false, err
	}
	newQuery := h.relatedQueryFactory(r, related, sourceRecord, targetConfig)

	countQuery, err := newQuery()
	if err != nil {
		return nil, 0, false, err
	}
	countQuery, supported, err := applyRelatedQueryFilter(countQuery, filter, related)
	if err != nil || !supported {
		return nil, 0, supported, err
	}
	totalCount, err := callQueryCount(r.Context(), countQuery, related)
	if err != nil {
		return nil, 0, false, err
	}

	pageQuery, err := newQuery()
	if err != nil {
		return nil, 0, false, err
	}
	pageQuery, supported, err = applyRelatedQueryFilter(pageQuery, filter, related)
	if err != nil || !supported {
		return nil, 0, supported, err
	}
	pageQuery, supported, err = applyRelatedQuerySort(pageQuery, sortState, related)
	if err != nil || !supported {
		return nil, 0, supported, err
	}
	if limit > 0 {
		pageQuery, err = callQueryIntMethod(pageQuery, "Limit", limit, related)
		if err != nil {
			return nil, 0, false, err
		}
	}
	if offset > 0 {
		pageQuery, err = callQueryIntMethod(pageQuery, "Offset", offset, related)
		if err != nil {
			return nil, 0, false, err
		}
	}
	records, err := callQueryAll(r.Context(), pageQuery, related)
	if err != nil {
		return nil, 0, false, err
	}
	return records, totalCount, true, nil
}

func (h *Handler) relatedQueryContext(r *http.Request, related RelatedFilter) (interface{}, *ModelConfig, error) {
	sourceConfig, err := h.Registry.Get(related.SourceResource)
	if err != nil {
		return nil, nil, err
	}
	targetConfig, err := h.Registry.Get(related.SourceField.RelationTarget)
	if err != nil {
		return nil, nil, err
	}
	sourceRecord, err := sourceConfig.QueryByID(r.Context(), related.SourceID)
	if err != nil {
		return nil, nil, err
	}
	return sourceRecord, targetConfig, nil
}

func (h *Handler) relatedQueryFactory(r *http.Request, related RelatedFilter, sourceRecord interface{}, targetConfig *ModelConfig) func() (interface{}, error) {
	return func() (interface{}, error) {
		recordValue := reflect.ValueOf(sourceRecord)
		queryMethod := recordValue.MethodByName("Query" + related.SourceField.Name)
		if !queryMethod.IsValid() {
			return nil, fmt.Errorf("related query %s.%s is not available", related.SourceResource, related.SourceField.Name)
		}
		queryResults := queryMethod.Call(nil)
		if len(queryResults) != 1 {
			return nil, fmt.Errorf("related query %s.%s returned unexpected values", related.SourceResource, related.SourceField.Name)
		}
		query := queryResults[0].Interface()
		if targetConfig.QueryModifier != nil {
			query = targetConfig.QueryModifier(r.Context(), query)
		}
		if query == nil {
			return nil, fmt.Errorf("related query %s.%s modifier returned nil", related.SourceResource, related.SourceField.Name)
		}
		return query, nil
	}
}

func applyRelatedQueryFilter(query interface{}, filter GridFilter, related RelatedFilter) (interface{}, bool, error) {
	if !filter.Valid {
		return query, true, nil
	}
	predicate, supported, err := relatedQueryFilterPredicate(filter, related)
	if err != nil {
		return nil, false, err
	}
	if !supported {
		return query, false, nil
	}
	query, err = callQueryWhere(query, predicate, related)
	if err != nil {
		return nil, false, err
	}
	return query, true, nil
}

func applyRelatedQuerySort(query interface{}, sortState GridSort, related RelatedFilter) (interface{}, bool, error) {
	if !sortState.Valid {
		return query, true, nil
	}
	column := fieldColumnName(sortState.Field)
	if column == "" || sortState.Field.Relation || sortState.Field.Virtual {
		return query, false, nil
	}
	query, err := callQueryOrder(query, column, sortState.Dir, related)
	if err != nil {
		return nil, false, err
	}
	return query, true, nil
}

func relatedQueryFilterPredicate(filter GridFilter, related RelatedFilter) (func(*entsql.Selector), bool, error) {
	column := fieldColumnName(filter.Field)
	if column == "" || filter.Field.Relation || filter.Field.Virtual {
		return nil, false, nil
	}
	switch filter.Op {
	case "equals":
		value, supported, err := queryFilterEqualValue(filter.Field, filter.Value)
		if err != nil || !supported {
			return nil, supported, err
		}
		return entsql.FieldEQ(column, value), true, nil
	default:
		if !fieldSupportsContains(filter.Field) {
			return nil, false, nil
		}
		return entsql.FieldContainsFold(column, filter.Value), true, nil
	}
}

func fieldSupportsContains(field FieldConfig) bool {
	switch field.Type {
	case FieldTypeString, FieldTypeText, FieldTypeEmail, FieldTypePassword:
		return true
	default:
		return false
	}
}

func queryFilterEqualValue(field FieldConfig, raw string) (interface{}, bool, error) {
	if field.Name == "ID" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, true, fmt.Errorf("%s must be a valid UUID", field.Label)
		}
		return id, true, nil
	}
	switch field.Type {
	case FieldTypeString, FieldTypeText, FieldTypeEmail, FieldTypePassword:
		return raw, true, nil
	case FieldTypeBool:
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "true", "1", "yes", "on", "✓":
			return true, true, nil
		case "false", "0", "no", "off", "✗":
			return false, true, nil
		default:
			return nil, true, fmt.Errorf("%s must be true or false", field.Label)
		}
	case FieldTypeInt:
		value, err := strconv.Atoi(raw)
		if err != nil {
			return nil, true, fmt.Errorf("%s must be a whole number", field.Label)
		}
		return value, true, nil
	case FieldTypeFloat:
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, true, fmt.Errorf("%s must be a number", field.Label)
		}
		return value, true, nil
	case FieldTypeTime:
		for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02", time.RFC3339, "Jan 2, 2006 3:04 PM"} {
			if value, err := time.Parse(layout, raw); err == nil {
				return value, true, nil
			}
		}
		return nil, true, fmt.Errorf("%s must be a valid date/time", field.Label)
	default:
		return nil, false, nil
	}
}

func fieldColumnName(field FieldConfig) string {
	if field.Column != "" {
		return field.Column
	}
	if field.Name == "ID" {
		return "id"
	}
	return camelToSnake(field.Name)
}

func callQueryCount(ctx context.Context, query interface{}, related RelatedFilter) (int, error) {
	queryVal := reflect.ValueOf(query)
	countMethod := queryVal.MethodByName("Count")
	if !countMethod.IsValid() {
		return 0, fmt.Errorf("related query %s.%s cannot count records", related.SourceResource, related.SourceField.Name)
	}
	countResults := countMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(countResults) != 2 {
		return 0, fmt.Errorf("related query %s.%s returned unexpected count values", related.SourceResource, related.SourceField.Name)
	}
	if !countResults[1].IsNil() {
		return 0, countResults[1].Interface().(error)
	}
	return int(countResults[0].Int()), nil
}

func callQueryWhere(query interface{}, predicate func(*entsql.Selector), related RelatedFilter) (interface{}, error) {
	queryVal := reflect.ValueOf(query)
	whereMethod := queryVal.MethodByName("Where")
	if !whereMethod.IsValid() {
		return nil, fmt.Errorf("related query %s.%s cannot apply filter", related.SourceResource, related.SourceField.Name)
	}
	predicateValue, err := makeQueryFuncArg(whereMethod, predicate)
	if err != nil {
		return nil, fmt.Errorf("related query %s.%s filter predicate: %w", related.SourceResource, related.SourceField.Name, err)
	}
	whereResults := whereMethod.Call([]reflect.Value{predicateValue})
	if len(whereResults) != 1 {
		return nil, fmt.Errorf("related query %s.%s returned unexpected filter values", related.SourceResource, related.SourceField.Name)
	}
	return whereResults[0].Interface(), nil
}

func callQueryOrder(query interface{}, column, direction string, related RelatedFilter) (interface{}, error) {
	queryVal := reflect.ValueOf(query)
	orderMethod := queryVal.MethodByName("Order")
	if !orderMethod.IsValid() {
		return nil, fmt.Errorf("related query %s.%s cannot apply sort", related.SourceResource, related.SourceField.Name)
	}
	orderFn := func(selector *entsql.Selector) {
		opts := []entsql.OrderTermOption{entsql.OrderAsc()}
		if direction == "desc" {
			opts = []entsql.OrderTermOption{entsql.OrderDesc()}
		}
		entsql.OrderByField(column, opts...).ToFunc()(selector)
	}
	orderValue, err := makeQueryFuncArg(orderMethod, orderFn)
	if err != nil {
		return nil, fmt.Errorf("related query %s.%s sort option: %w", related.SourceResource, related.SourceField.Name, err)
	}
	orderResults := orderMethod.Call([]reflect.Value{orderValue})
	if len(orderResults) != 1 {
		return nil, fmt.Errorf("related query %s.%s returned unexpected sort values", related.SourceResource, related.SourceField.Name)
	}
	return orderResults[0].Interface(), nil
}

func makeQueryFuncArg(method reflect.Value, fn interface{}) (reflect.Value, error) {
	methodType := method.Type()
	if !methodType.IsVariadic() || methodType.NumIn() != 1 {
		return reflect.Value{}, fmt.Errorf("unexpected method signature")
	}
	sliceType := methodType.In(0)
	if sliceType.Kind() != reflect.Slice {
		return reflect.Value{}, fmt.Errorf("unexpected variadic argument type %s", sliceType)
	}
	funcType := sliceType.Elem()
	if funcType.Kind() != reflect.Func || funcType.NumIn() != 1 || funcType.NumOut() != 0 {
		return reflect.Value{}, fmt.Errorf("unexpected function argument type %s", funcType)
	}
	sourceValue := reflect.ValueOf(fn)
	if sourceValue.Type().AssignableTo(funcType) {
		return sourceValue, nil
	}
	if sourceValue.Type().ConvertibleTo(funcType) {
		return sourceValue.Convert(funcType), nil
	}
	return reflect.MakeFunc(funcType, func(args []reflect.Value) []reflect.Value {
		if len(args) == 1 && !args[0].IsNil() {
			if selector, ok := args[0].Interface().(*entsql.Selector); ok {
				sourceValue.Call([]reflect.Value{reflect.ValueOf(selector)})
			}
		}
		return nil
	}), nil
}

func callQueryIntMethod(query interface{}, methodName string, value int, related RelatedFilter) (interface{}, error) {
	queryVal := reflect.ValueOf(query)
	method := queryVal.MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("related query %s.%s cannot apply %s", related.SourceResource, related.SourceField.Name, methodName)
	}
	methodResults := method.Call([]reflect.Value{reflect.ValueOf(value)})
	if len(methodResults) != 1 {
		return nil, fmt.Errorf("related query %s.%s returned unexpected %s values", related.SourceResource, related.SourceField.Name, methodName)
	}
	return methodResults[0].Interface(), nil
}

func callQueryAll(ctx context.Context, query interface{}, related RelatedFilter) ([]interface{}, error) {
	queryVal := reflect.ValueOf(query)
	allMethod := queryVal.MethodByName("All")
	if !allMethod.IsValid() {
		return nil, fmt.Errorf("related query %s.%s cannot list records", related.SourceResource, related.SourceField.Name)
	}
	allResults := allMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(allResults) != 2 {
		return nil, fmt.Errorf("related query %s.%s returned unexpected list values", related.SourceResource, related.SourceField.Name)
	}
	if !allResults[1].IsNil() {
		return nil, allResults[1].Interface().(error)
	}

	recordsVal := allResults[0]
	records := make([]interface{}, recordsVal.Len())
	for i := 0; i < recordsVal.Len(); i++ {
		records[i] = recordsVal.Index(i).Interface()
	}
	return records, nil
}

func (h *Handler) publicResourceConfig(w http.ResponseWriter, r *http.Request) (*ModelConfig, bool) {
	config, err := h.Registry.Get(chi.URLParam(r, "resource"))
	if err != nil || config.Internal {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Model not found")
		return nil, false
	}
	return config, true
}

func parseWorkspaceView(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case workspaceViewGrid:
		return workspaceViewGrid
	default:
		return workspaceViewOverview
	}
}

func allWorkspaceFields(config *ModelConfig) []FieldConfig {
	if config == nil {
		return nil
	}
	fields := make([]FieldConfig, 0, len(config.Fields))
	for _, field := range config.Fields {
		if field.Visible && !field.Hidden && !field.Sensitive {
			fields = append(fields, field)
		}
	}
	return fields
}

func parseSelectedGridFields(raw []string, allFields, defaultFields []FieldConfig) map[string]bool {
	allowed := make(map[string]bool, len(allFields))
	for _, field := range allFields {
		allowed[field.Name] = true
	}
	names := flattenGridParamValues(raw)
	selected := make(map[string]bool, len(names))
	for _, name := range names {
		if allowed[name] {
			selected[name] = true
		}
	}
	if len(selected) == 0 {
		for _, field := range defaultFields {
			if allowed[field.Name] {
				selected[field.Name] = true
			}
		}
	}
	if len(selected) == 0 {
		for _, field := range allFields {
			selected[field.Name] = true
		}
	}
	return selected
}

func flattenGridParamValues(values []string) []string {
	var result []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
	}
	return result
}

func fieldsBySelection(allFields []FieldConfig, selected map[string]bool) []FieldConfig {
	fields := make([]FieldConfig, 0, len(allFields))
	for _, field := range allFields {
		if selected[field.Name] {
			fields = append(fields, field)
		}
	}
	return fields
}

func parseGridFilter(r *http.Request, fields []FieldConfig) GridFilter {
	value := strings.TrimSpace(r.URL.Query().Get("filter_value"))
	if value == "" {
		return GridFilter{}
	}
	field, ok := findGridField(fields, r.URL.Query().Get("filter_field"), func(field FieldConfig) bool {
		return field.Filterable
	})
	if !ok {
		return GridFilter{}
	}
	op := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("filter_op")))
	if op != "equals" {
		op = "contains"
	}
	return GridFilter{Field: field, Op: op, Value: value, Valid: true}
}

func parseGridSort(r *http.Request, fields []FieldConfig) GridSort {
	field, ok := findGridField(fields, r.URL.Query().Get("sort_field"), func(field FieldConfig) bool {
		return field.Sortable
	})
	if !ok {
		return GridSort{}
	}
	dir := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort_dir")))
	if dir != "desc" {
		dir = "asc"
	}
	return GridSort{Field: field, Dir: dir, Valid: true}
}

func findGridField(fields []FieldConfig, name string, accept func(FieldConfig) bool) (FieldConfig, bool) {
	for _, field := range fields {
		if field.Name == name && accept(field) {
			return field, true
		}
	}
	return FieldConfig{}, false
}

func filterRecords(records []interface{}, filter GridFilter) []interface{} {
	filtered := make([]interface{}, 0, len(records))
	needle := strings.ToLower(filter.Value)
	for _, record := range records {
		value := strings.ToLower(strings.TrimSpace(cellValue(record, filter.Field)))
		switch filter.Op {
		case "equals":
			if value == needle {
				filtered = append(filtered, record)
			}
		default:
			if strings.Contains(value, needle) {
				filtered = append(filtered, record)
			}
		}
	}
	return filtered
}

func sortRecords(records []interface{}, sortState GridSort) {
	sort.SliceStable(records, func(i, j int) bool {
		cmp := compareRecordField(records[i], records[j], sortState.Field)
		if sortState.Dir == "desc" {
			return cmp > 0
		}
		return cmp < 0
	})
}

func compareRecordField(left, right interface{}, field FieldConfig) int {
	leftValue := rawFieldValue(left, field.Name)
	rightValue := rawFieldValue(right, field.Name)
	switch field.Type {
	case FieldTypeInt:
		return compareFloat(fieldFloatValue(leftValue), fieldFloatValue(rightValue))
	case FieldTypeFloat:
		return compareFloat(fieldFloatValue(leftValue), fieldFloatValue(rightValue))
	case FieldTypeBool:
		return compareString(strconv.FormatBool(fieldBoolValue(leftValue)), strconv.FormatBool(fieldBoolValue(rightValue)))
	case FieldTypeTime:
		return compareTime(fieldTimeValue(leftValue), fieldTimeValue(rightValue))
	default:
		return compareString(fmt.Sprint(leftValue), fmt.Sprint(rightValue))
	}
}

func fieldFloatValue(value interface{}) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		parsed, _ := strconv.ParseFloat(fmt.Sprint(value), 64)
		return parsed
	}
}

func fieldBoolValue(value interface{}) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	parsed, _ := strconv.ParseBool(fmt.Sprint(value))
	return parsed
}

func fieldTimeValue(value interface{}) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v
	case *time.Time:
		if v != nil {
			return *v
		}
	}
	parsed, _ := time.Parse(time.RFC3339, fmt.Sprint(value))
	return parsed
}

func compareFloat(left, right float64) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func compareTime(left, right time.Time) int {
	switch {
	case left.Before(right):
		return -1
	case left.After(right):
		return 1
	default:
		return 0
	}
}

func compareString(left, right string) int {
	left = strings.ToLower(left)
	right = strings.ToLower(right)
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func parseAllowedPerPage(raw string, fallback int) int {
	value := parsePositiveInt(raw, fallback)
	switch value {
	case 20, 50, 100:
		return value
	default:
		return fallback
	}
}

func currentGridPage(r *http.Request) int {
	return parsePositiveInt(r.URL.Query().Get("page"), 1)
}

func currentGridPerPage(r *http.Request) int {
	return parseAllowedPerPage(r.URL.Query().Get("per_page"), 50)
}

func currentGridQuery(r *http.Request) template.URL {
	source := r.URL.Query()
	values := url.Values{}
	values.Set("view", workspaceViewGrid)
	values.Set("page", strconv.Itoa(currentGridPage(r)))
	values.Set("per_page", strconv.Itoa(currentGridPerPage(r)))
	for _, field := range flattenGridParamValues(source["fields"]) {
		values.Add("fields", field)
	}
	if field := strings.TrimSpace(source.Get("filter_field")); field != "" {
		values.Set("filter_field", field)
	}
	if op := strings.TrimSpace(source.Get("filter_op")); op != "" {
		values.Set("filter_op", op)
	}
	if value := strings.TrimSpace(source.Get("filter_value")); value != "" {
		values.Set("filter_value", value)
	}
	if field := strings.TrimSpace(source.Get("sort_field")); field != "" {
		values.Set("sort_field", field)
	}
	if dir := strings.TrimSpace(source.Get("sort_dir")); dir != "" {
		values.Set("sort_dir", dir)
	}
	if relatedFrom := strings.TrimSpace(source.Get("related_from")); relatedFrom != "" {
		values.Set("related_from", relatedFrom)
	}
	if relatedField := strings.TrimSpace(source.Get("related_field")); relatedField != "" {
		values.Set("related_field", relatedField)
	}
	if relatedID := strings.TrimSpace(source.Get("related_id")); relatedID != "" {
		values.Set("related_id", relatedID)
	}
	return template.URL(values.Encode())
}

func submittedFieldValue(r *http.Request, name string) string {
	values := r.Form[name]
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] == "true" || values[i] == "on" || values[i] == "1" {
			return values[i]
		}
	}
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

func (h *Handler) lookupMutationTarget(w http.ResponseWriter, r *http.Request) (*ModelConfig, FieldConfig, uuid.UUID, bool) {
	config, ok := h.publicResourceConfig(w, r)
	if !ok {
		return nil, FieldConfig{}, uuid.UUID{}, false
	}
	fieldName := chi.URLParam(r, "field")
	field, ok := findField(config, fieldName)
	if !ok {
		h.Renderer.RenderError(w, r, http.StatusNotFound, "Field not found")
		return nil, FieldConfig{}, uuid.UUID{}, false
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.Renderer.RenderError(w, r, http.StatusBadRequest, "Invalid ID")
		return nil, FieldConfig{}, uuid.UUID{}, false
	}
	return config, field, id, true
}

func findField(config *ModelConfig, name string) (FieldConfig, bool) {
	for _, field := range config.Fields {
		if field.Name == name {
			return field, true
		}
	}
	return FieldConfig{}, false
}

func canInlineEdit(field FieldConfig) bool {
	if !field.Editable || field.Readonly || field.Hidden || field.Virtual || field.Sensitive {
		return false
	}
	switch field.Type {
	case FieldTypePassword, FieldTypeSelect:
		return false
	default:
		return true
	}
}

func (h *Handler) parseFieldValueStrict(field FieldConfig, value string) (interface{}, error) {
	switch field.Type {
	case FieldTypeBool:
		return value == "on" || value == "true" || value == "1", nil
	case FieldTypeInt:
		if value == "" {
			return 0, nil
		}
		i, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("%s must be a whole number", field.Label)
		}
		return i, nil
	case FieldTypeFloat:
		if value == "" {
			return 0.0, nil
		}
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("%s must be a number", field.Label)
		}
		return f, nil
	case FieldTypeTime:
		if value == "" {
			return clearFieldValue{}, nil
		}
		for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", time.RFC3339} {
			if t, err := time.Parse(layout, value); err == nil {
				return t, nil
			}
		}
		return nil, fmt.Errorf("%s must be a valid date/time", field.Label)
	default:
		return value, nil
	}
}

func (h *Handler) editableFormData(r *http.Request, config *ModelConfig, isCreate bool) (map[string]interface{}, map[string]string) {
	if err := r.ParseForm(); err != nil {
		return nil, map[string]string{"_general": "Invalid form data"}
	}
	data := make(map[string]interface{})
	errors := make(map[string]string)
	for _, field := range config.Fields {
		if field.Readonly || field.Hidden || !field.Editable {
			continue
		}
		if !isCreate && field.Type == FieldTypePassword && r.Form.Get(field.Name) == "" {
			continue
		}
		if field.Type == FieldTypeBool {
			_, exists := r.Form[field.Name]
			_, present := r.Form[field.Name+"__present"]
			if isCreate && !exists && !present {
				continue
			}
			data[field.Name] = exists
			continue
		}
		value, err := h.parseFieldValueStrict(field, r.Form.Get(field.Name))
		if err != nil {
			errors[field.Name] = err.Error()
			continue
		}
		if _, clear := value.(clearFieldValue); clear && isCreate {
			continue
		}
		data[field.Name] = value
		if isCreate && field.Required && (value == nil || value == "") {
			errors[field.Name] = field.Label + " is required"
		}
	}
	return data, errors
}

func (h *Handler) renderCellError(w http.ResponseWriter, r *http.Request, config *ModelConfig, field FieldConfig, id uuid.UUID, message string) {
	record, _ := config.QueryByID(r.Context(), id)
	h.Renderer.Render(w, r, "grid_cell.partial.html", &TemplateData{
		Errors: map[string]string{field.Name: message},
		Data: map[string]interface{}{
			"Config": config,
			"Errors": map[string]string{field.Name: message},
			"Field":  field,
			"Record": record,
		},
	})
}

func (h *Handler) rejectProtectedRecordMutations(config *ModelConfig, record interface{}, data map[string]interface{}) error {
	if config == nil || config.Name != "AdminSetting" {
		return nil
	}
	for fieldName := range data {
		if isProtectedRecordField(config, record, fieldName) {
			return fmt.Errorf("field %s is protected", fieldName)
		}
	}
	if record == nil {
		key, _ := data["Key"].(string)
		if strings.HasPrefix(key, "admin.") {
			return fmt.Errorf("admin setting keys are protected")
		}
	}
	return nil
}

func isProtectedAdminSettingRecord(config *ModelConfig, record interface{}) bool {
	return config != nil && config.Name == "AdminSetting" && isAdminSettingRecord(record)
}

// SaveModelOrderPreference saves the model order preference.
func (h *Handler) SaveModelOrderPreference(w http.ResponseWriter, r *http.Request) {
	// Parse JSON body
	var request struct {
		Order []string `json:"order"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Save order to database
	if err := h.Registry.SaveModelOrder(request.Order); err != nil {
		utils.Errorf("Failed to save model order: %v", err)
		http.Error(w, "Failed to save order", http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}
