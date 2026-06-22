package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestWorkspaceRendersOverviewByDefault(t *testing.T) {
	registry := &Registry{models: map[string]*ModelConfig{}}
	config := &ModelConfig{
		Name:       "Thing",
		NamePlural: "Things",
		Icon:       "▦",
		Fields: []FieldConfig{
			{Name: "ID", Label: "ID", Type: FieldTypeString, Readonly: true, Visible: true, Width: 200, System: true},
			{Name: "Name", Label: "Name", Type: FieldTypeString, Editable: true, Visible: true, Width: 220, Primary: true},
		},
		ListFields: []string{"ID", "Name"},
		CountAll: func(ctx context.Context) (int, error) {
			return 0, nil
		},
		QueryAllPaginated: func(ctx context.Context, limit, offset int) ([]interface{}, error) {
			return nil, nil
		},
	}
	registry.register(config)

	renderer, err := NewAdminRenderer(false)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(registry, renderer, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/t/thing", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("resource", "thing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.Workspace(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"workspace-shell", "resource-overview", "Open grid"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected rendered workspace to contain %q", want)
		}
	}
	if strings.Contains(body, "sheet-table") {
		t.Fatalf("expected default workspace to render overview instead of grid, got %s", body)
	}
	if strings.Contains(body, "Rows per page") {
		t.Fatalf("expected overview to hide rows per page controls, got %s", body)
	}
}

func TestWorkspaceViewGridRendersGridShell(t *testing.T) {
	registry := &Registry{models: map[string]*ModelConfig{}}
	config := &ModelConfig{
		Name:       "Thing",
		NamePlural: "Things",
		Icon:       "▦",
		Fields: []FieldConfig{
			{Name: "ID", Label: "ID", Type: FieldTypeString, Readonly: true, Visible: true, Width: 200, System: true},
			{Name: "Name", Label: "Name", Type: FieldTypeString, Editable: true, Visible: true, Width: 220, Primary: true},
		},
		ListFields: []string{"ID", "Name"},
		CountAll: func(ctx context.Context) (int, error) {
			return 0, nil
		},
		QueryAllPaginated: func(ctx context.Context, limit, offset int) ([]interface{}, error) {
			return nil, nil
		},
	}
	registry.register(config)

	renderer, err := NewAdminRenderer(false)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(registry, renderer, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/t/thing?view=grid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("resource", "thing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.Workspace(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"workspace-shell", "sheet-table", "Add a new thing", `aria-current="page"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected rendered grid workspace to contain %q", want)
		}
	}
	if strings.Contains(body, `class="toolbar-actions"`) {
		t.Fatalf("expected grid actions to render inside the grid, not the page header, got %s", body)
	}
	if strings.Contains(body, "resource-overview") {
		t.Fatalf("expected grid view to render grid instead of overview, got %s", body)
	}
}

func TestInternalResourceWorkspaceIsNotAccessible(t *testing.T) {
	registry := &Registry{models: map[string]*ModelConfig{}}
	registry.register(&ModelConfig{
		Name:       "AdminSetting",
		NamePlural: "AdminSettings",
		Internal:   true,
	})

	renderer, err := NewAdminRenderer(false)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(registry, renderer, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/t/adminsetting?view=grid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("resource", "adminsetting")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.Workspace(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for internal admin resource, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLegacyRedirectUsesWorkspaceRoute(t *testing.T) {
	handler := NewHandler(nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/user", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("model", "user")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.LegacyRedirect(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/admin/t/user" {
		t.Fatalf("expected /admin/t/user, got %s", got)
	}
}

func TestUpdateCellRejectsProtectedField(t *testing.T) {
	protected := FieldConfig{Name: "ID", Readonly: true, Editable: false, System: true}
	if canInlineEdit(protected) {
		t.Fatal("protected ID field should not be inline editable")
	}
	editable := FieldConfig{Name: "Name", Type: FieldTypeString, Editable: true, Visible: true}
	if !canInlineEdit(editable) {
		t.Fatal("editable string field should be inline editable")
	}
}

func TestRecordDrawerPreservesGridPagination(t *testing.T) {
	id := uuid.New()
	registry := &Registry{models: map[string]*ModelConfig{}}
	config := &ModelConfig{
		Name:       "Thing",
		NamePlural: "Things",
		Fields: []FieldConfig{
			{Name: "ID", Label: "ID", Type: FieldTypeString, Readonly: true, Visible: true, Width: 200, System: true},
			{Name: "Name", Label: "Name", Type: FieldTypeString, Editable: true, Visible: true, Width: 220},
		},
		QueryByID: func(ctx context.Context, got uuid.UUID) (interface{}, error) {
			if got != id {
				t.Fatalf("expected ID %s, got %s", id, got)
			}
			return struct {
				ID   uuid.UUID
				Name string
			}{ID: id, Name: "Row"}, nil
		},
	}
	registry.register(config)

	renderer, err := NewAdminRenderer(false)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(registry, renderer, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/t/thing/records/"+id.String()+"?page=3&per_page=20", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("resource", "thing")
	rctx.URLParams.Add("id", id.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.RecordDrawer(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	for _, want := range []string{"records/" + id.String() + "?", "page=3", "per_page=20"} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("expected drawer mutation URL to preserve %q, got %s", want, w.Body.String())
		}
	}
}

func TestReadonlyGridCellsPreserveGridPagination(t *testing.T) {
	id := uuid.New()
	registry := &Registry{models: map[string]*ModelConfig{}}
	config := &ModelConfig{
		Name:       "Thing",
		NamePlural: "Things",
		Icon:       "▦",
		Fields: []FieldConfig{
			{Name: "ID", Label: "ID", Type: FieldTypeString, Readonly: true, Visible: true, Width: 200, System: true},
			{Name: "Name", Label: "Name", Type: FieldTypeString, Editable: true, Visible: true, Width: 220},
		},
		ListFields: []string{"ID", "Name"},
		CountAll: func(ctx context.Context) (int, error) {
			return 25, nil
		},
		QueryAllPaginated: func(ctx context.Context, limit, offset int) ([]interface{}, error) {
			return []interface{}{
				struct {
					ID   uuid.UUID
					Name string
				}{ID: id, Name: "Row"},
			}, nil
		},
	}
	registry.register(config)

	renderer, err := NewAdminRenderer(false)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(registry, renderer, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/t/thing/grid?page=2&per_page=20", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("resource", "thing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.Grid(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	for _, want := range []string{"records/" + id.String() + "?", "page=2", "per_page=20"} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("expected locked cell drawer URL to preserve %q, got %s", want, w.Body.String())
		}
	}
	for _, want := range []string{`href="/admin/t/thing?`, `view=grid`, `page=2`, `per_page=20`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("expected Grid view tab to preserve %q, got %s", want, w.Body.String())
		}
	}
	if !strings.Contains(w.Body.String(), `aria-current="page"`) {
		t.Fatalf("expected Grid view tab to mark the current layout, got %s", w.Body.String())
	}
	for _, want := range []string{
		`name="per_page"`,
		`<option value="20" selected>20</option>`,
		`<option value="50" >50</option>`,
		`<option value="100" >100</option>`,
		`<div class="grid-action-row">`,
		`<details id="fields-control" class="grid-control-details" name="grid-control">`,
		`<details id="filter-control" class="grid-control-details" name="grid-control">`,
		`<details id="sort-control" class="grid-control-details" name="grid-control">`,
		`>+ Add record</button>`,
		`class="sheet-footer"`,
		`page=1`,
		`class="sheet-page-button disabled" aria-disabled="true">Next</span>`,
	} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("expected rendered grid to contain %q, got %s", want, w.Body.String())
		}
	}
	if strings.Contains(w.Body.String(), `<details id="fields-control" class="grid-control-details" name="grid-control" open`) ||
		strings.Contains(w.Body.String(), `<details id="filter-control" class="grid-control-details" name="grid-control" open`) ||
		strings.Contains(w.Body.String(), `<details id="sort-control" class="grid-control-details" name="grid-control" open`) {
		t.Fatalf("expected grid control sections to be closed by default, got %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), `control-indicator`) {
		t.Fatalf("expected filter indicator to be hidden when no filter is active, got %s", w.Body.String())
	}
	if strings.Index(w.Body.String(), `<div class="grid-action-row">`) > strings.Index(w.Body.String(), `<form class="sheet-page-size"`) {
		t.Fatalf("expected row count dropdown below grid control buttons, got %s", w.Body.String())
	}
}

func TestGridFieldSelectionFilterAndSortAffectRows(t *testing.T) {
	registry := &Registry{models: map[string]*ModelConfig{}}
	records := []interface{}{
		struct {
			ID    uuid.UUID
			Name  string
			Score int
		}{ID: uuid.New(), Name: "beta", Score: 2},
		struct {
			ID    uuid.UUID
			Name  string
			Score int
		}{ID: uuid.New(), Name: "alpha", Score: 1},
		struct {
			ID    uuid.UUID
			Name  string
			Score int
		}{ID: uuid.New(), Name: "gamma", Score: 3},
	}
	config := &ModelConfig{
		Name:       "Thing",
		NamePlural: "Things",
		Icon:       "▦",
		Fields: []FieldConfig{
			{Name: "ID", Label: "ID", Type: FieldTypeString, Readonly: true, Visible: true, Width: 200, System: true, Sortable: true, Filterable: true},
			{Name: "Name", Label: "Name", Type: FieldTypeString, Editable: true, Visible: true, Width: 220, Sortable: true, Filterable: true},
			{Name: "Score", Label: "Score", Type: FieldTypeInt, Editable: true, Visible: true, Width: 120, Sortable: true, Filterable: true},
		},
		ListFields: []string{"ID", "Name", "Score"},
		QueryAll: func(ctx context.Context) ([]interface{}, error) {
			return records, nil
		},
		CountAll: func(ctx context.Context) (int, error) {
			return len(records), nil
		},
		QueryAllPaginated: func(ctx context.Context, limit, offset int) ([]interface{}, error) {
			return records, nil
		},
	}
	registry.register(config)

	renderer, err := NewAdminRenderer(false)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(registry, renderer, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/t/thing/grid?fields=Name&filter_field=Name&filter_op=contains&filter_value=a&sort_field=Name&sort_dir=desc&page=1&per_page=20", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("resource", "thing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.Grid(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, `<span class="field-pill">ID`) || strings.Contains(body, `<span class="field-pill">Score`) {
		t.Fatalf("expected selected fields to hide ID and Score columns, got %s", body)
	}
	if !strings.Contains(body, `<span class="field-pill">Name`) {
		t.Fatalf("expected Name column to render, got %s", body)
	}
	if !strings.Contains(body, `value="Name" checked`) {
		t.Fatalf("expected field selector to preserve Name selection, got %s", body)
	}
	for _, want := range []string{`filter_value" value="a"`, `sort_field"`, `sort_dir" value="desc"`, `fields" value="Name"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected grid state to preserve %q, got %s", want, body)
		}
	}
	for _, want := range []string{`has-active-filter`, `<span>Filter</span>`, `class="control-indicator"`, `aria-label="Filter active"`, `>Reset</a>`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected active filter indicator %q, got %s", want, body)
		}
	}
	alphaIndex := strings.Index(body, "alpha")
	betaIndex := strings.Index(body, "beta")
	gammaIndex := strings.Index(body, "gamma")
	if gammaIndex < 0 || betaIndex < 0 || alphaIndex < 0 {
		t.Fatalf("expected filtered rows to render, got %s", body)
	}
	if !(gammaIndex < betaIndex && betaIndex < alphaIndex) {
		t.Fatalf("expected rows sorted by Name descending, got %s", body)
	}
}

func TestRenderCellErrorReturnsSwappablePartial(t *testing.T) {
	id := uuid.New()
	config := &ModelConfig{
		Name: "Thing",
		QueryByID: func(ctx context.Context, got uuid.UUID) (interface{}, error) {
			return struct {
				ID   uuid.UUID
				Name string
			}{ID: got, Name: "Row"}, nil
		},
	}
	renderer, err := NewAdminRenderer(false)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(nil, renderer, nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/t/thing/records/"+id.String()+"/fields/Name", nil)
	w := httptest.NewRecorder()

	handler.renderCellError(w, req, config, FieldConfig{Name: "Name", Label: "Name", Type: FieldTypeString, Editable: true, Visible: true, Width: 120}, id, "Name is invalid")

	if w.Code != http.StatusOK {
		t.Fatalf("expected swappable 200 response, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Name is invalid") {
		t.Fatalf("expected rendered cell error, got %s", w.Body.String())
	}
}

func TestSubmittedFieldValuePrefersCheckedBoolValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/admin/t/user/records/1/fields/IsActive", strings.NewReader("value=false&value=true"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatal(err)
	}
	if got := submittedFieldValue(req, "value"); got != "true" {
		t.Fatalf("expected checked bool value true, got %q", got)
	}
}

func TestSetFieldsOnBuilderClearsNullableTime(t *testing.T) {
	builder := &fakeTimeUpdateBuilder{}
	if err := setFieldsOnBuilder(builder, map[string]interface{}{"LastLogin": clearFieldValue{}}); err != nil {
		t.Fatal(err)
	}
	if !builder.clearedLastLogin {
		t.Fatal("expected LastLogin to be cleared")
	}

	now := time.Now()
	if err := setFieldsOnBuilder(builder, map[string]interface{}{"LastLogin": now}); err != nil {
		t.Fatal(err)
	}
	if builder.lastLogin == nil || !builder.lastLogin.Equal(now) {
		t.Fatal("expected LastLogin to be set")
	}
}

func TestEditableFormDataOmitsAbsentBoolOnCreate(t *testing.T) {
	handler := NewHandler(nil, nil, nil)
	config := &ModelConfig{
		Fields: []FieldConfig{
			{Name: "IsActive", Label: "Active", Type: FieldTypeBool, Editable: true, Visible: true},
		},
	}

	createReq := httptest.NewRequest(http.MethodPost, "/admin/t/user/records", nil)
	createData, createErrors := handler.editableFormData(createReq, config, true)
	if len(createErrors) != 0 {
		t.Fatalf("expected no create errors, got %v", createErrors)
	}
	if _, ok := createData["IsActive"]; ok {
		t.Fatal("expected absent create bool to be omitted so Ent defaults can apply")
	}

	presentCreateReq := httptest.NewRequest(http.MethodPost, "/admin/t/user/records", strings.NewReader("IsActive__present=true"))
	presentCreateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	presentCreateData, presentCreateErrors := handler.editableFormData(presentCreateReq, config, true)
	if len(presentCreateErrors) != 0 {
		t.Fatalf("expected no present create errors, got %v", presentCreateErrors)
	}
	if got, ok := presentCreateData["IsActive"].(bool); !ok || got {
		t.Fatalf("expected present unchecked create bool to be false, got %#v", presentCreateData["IsActive"])
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/admin/t/user/records/1", nil)
	updateData, updateErrors := handler.editableFormData(updateReq, config, false)
	if len(updateErrors) != 0 {
		t.Fatalf("expected no update errors, got %v", updateErrors)
	}
	if got, ok := updateData["IsActive"].(bool); !ok || got {
		t.Fatalf("expected absent update bool to be false, got %#v", updateData["IsActive"])
	}
}

func TestRecordDrawerUsesBoolCreateDefaults(t *testing.T) {
	registry := &Registry{models: map[string]*ModelConfig{}}
	config := &ModelConfig{
		Name:       "Thing",
		NamePlural: "Things",
		Fields: []FieldConfig{
			{Name: "IsActive", Label: "Active", Type: FieldTypeBool, Editable: true, Visible: true, Default: true},
		},
	}
	registry.register(config)

	renderer, err := NewAdminRenderer(false)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(registry, renderer, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/t/thing/records/new", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("resource", "thing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.NewRecordDrawer(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `name="IsActive" value="true" checked`) {
		t.Fatalf("expected bool default to render checked, got %s", w.Body.String())
	}
}

func TestIDPredicateForWhereSupportsGenericEntPredicateTypes(t *testing.T) {
	query := &fakePredicateQuery{}
	whereMethod := reflectValueOfWhere(query)

	predicate, err := idPredicateForWhere(whereMethod, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if !predicate.Type().AssignableTo(whereMethod.Type().In(0).Elem()) {
		t.Fatalf("predicate type %s is not assignable to %s", predicate.Type(), whereMethod.Type().In(0).Elem())
	}
}

type fakePredicate func(*sql.Selector)

type fakePredicateQuery struct{}

func (q *fakePredicateQuery) Where(predicates ...fakePredicate) *fakePredicateQuery {
	return q
}

func reflectValueOfWhere(query *fakePredicateQuery) reflect.Value {
	return reflect.ValueOf(query).MethodByName("Where")
}

type fakeTimeUpdateBuilder struct {
	clearedLastLogin bool
	lastLogin        *time.Time
}

func (b *fakeTimeUpdateBuilder) ClearLastLogin() *fakeTimeUpdateBuilder {
	b.clearedLastLogin = true
	return b
}

func (b *fakeTimeUpdateBuilder) SetLastLogin(value time.Time) *fakeTimeUpdateBuilder {
	b.lastLogin = &value
	return b
}
