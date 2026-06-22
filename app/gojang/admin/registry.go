package admin

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/gojangframework/gojang/app/gojang/models"
	"github.com/google/uuid"
)

// Registry holds all registered models
type Registry struct {
	models    map[string]*ModelConfig
	modelKeys []string // Maintains order of registration
	client    *models.Client
}

// NewRegistry creates a new admin registry
func NewRegistry(client *models.Client) *Registry {
	r := &Registry{
		models: make(map[string]*ModelConfig),
		client: client,
	}
	if client != nil {
		r.DiscoverResources()
	}
	return r
}

// DiscoverResources auto-registers all Ent model clients exposed by models.Client.
func (r *Registry) DiscoverResources() {
	clientVal := reflect.ValueOf(r.client)
	if !clientVal.IsValid() || clientVal.Kind() != reflect.Ptr || clientVal.IsNil() {
		return
	}

	clientStruct := clientVal.Elem()
	clientType := clientStruct.Type()
	for i := 0; i < clientStruct.NumField(); i++ {
		field := clientType.Field(i)
		if !field.IsExported() || field.Name == "Schema" {
			continue
		}

		modelClient := clientStruct.Field(i)
		if !isEntResourceClient(modelClient) {
			continue
		}

		entityType, ok := entityTypeFromClient(modelClient)
		if !ok {
			continue
		}

		modelName := field.Name
		fields := extractFieldsFromType(entityType, AdminOverrides{})
		fields = append(fields, extractRelationFields(entityType)...)
		listFields := visibleFieldNames(fields)
		if len(listFields) > 7 {
			listFields = listFields[:7]
		}

		config := &ModelConfig{
			Name:           modelName,
			NamePlural:     pluralize(modelName),
			Icon:           defaultResourceIcon(modelName),
			Fields:         fields,
			ListFields:     listFields,
			HiddenFields:   hiddenFieldNames(fields),
			ReadonlyFields: readonlyFieldNames(fields),
		}
		applyResourceDefaults(config)
		config.QueryAll = func(ctx context.Context) ([]interface{}, error) {
			return r.queryAll(ctx, modelName, nil)
		}
		config.QueryAllPaginated = func(ctx context.Context, limit, offset int) ([]interface{}, error) {
			return r.queryAllPaginated(ctx, modelName, nil, limit, offset)
		}
		config.CountAll = func(ctx context.Context) (int, error) {
			return r.countAll(ctx, modelName)
		}
		config.QueryByID = func(ctx context.Context, id uuid.UUID) (interface{}, error) {
			return r.queryByID(ctx, modelName, id, nil)
		}
		config.CreateFunc = func(ctx context.Context, data map[string]interface{}) (interface{}, error) {
			return r.genericCreate(ctx, modelName, data)
		}
		config.UpdateFunc = func(ctx context.Context, id uuid.UUID, data map[string]interface{}) error {
			return r.genericUpdate(ctx, modelName, id, data)
		}
		config.DeleteFunc = func(ctx context.Context, id uuid.UUID) error {
			return r.genericDelete(ctx, modelName, id)
		}

		r.register(config)
	}
}

// RegisterModel registers a model using the simplified registration API
func (r *Registry) RegisterModel(reg ModelRegistration) error {
	// Extract model name from type
	modelType := reflect.TypeOf(reg.ModelType)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	modelName := modelType.Name()

	// Apply defaults
	if reg.NamePlural == "" {
		reg.NamePlural = pluralize(modelName)
	}

	// Build AdminOverrides from registration
	override := AdminOverrides{
		Icon:           reg.Icon,
		NamePlural:     reg.NamePlural,
		ListFields:     reg.ListFields,
		HiddenFields:   reg.HiddenFields,
		ReadonlyFields: reg.ReadonlyFields,
		OptionalFields: reg.OptionalFields,
	}

	// Use reflection to discover fields
	fields := extractFields(reg.ModelType, override)
	fields = appendMissingFields(fields, extractRelationFields(modelType)...)

	// Append custom fields if provided
	if reg.CustomFields != nil {
		fields = append(fields, reg.CustomFields...)
	}
	markPrimaryField(fields)

	listFields := reg.ListFields
	if len(listFields) == 0 {
		listFields = visibleFieldNames(fields)
		if len(listFields) > 7 {
			listFields = listFields[:7]
		}
	}
	hiddenFields := reg.HiddenFields
	if len(hiddenFields) == 0 {
		hiddenFields = hiddenFieldNames(fields)
	}
	readonlyFields := reg.ReadonlyFields
	if len(readonlyFields) == 0 {
		readonlyFields = readonlyFieldNames(fields)
	}

	// Create config with generic CRUD operations
	config := &ModelConfig{
		Name:           modelName,
		NamePlural:     reg.NamePlural,
		Icon:           reg.Icon,
		Fields:         fields,
		ListFields:     listFields,
		HiddenFields:   hiddenFields,
		ReadonlyFields: readonlyFields,

		QueryAll: func(ctx context.Context) ([]interface{}, error) {
			return r.queryAll(ctx, modelName, reg.QueryModifier)
		},

		QueryAllPaginated: func(ctx context.Context, limit, offset int) ([]interface{}, error) {
			return r.queryAllPaginated(ctx, modelName, reg.QueryModifier, limit, offset)
		},

		CountAll: func(ctx context.Context) (int, error) {
			return r.countAll(ctx, modelName)
		},

		QueryByID: func(ctx context.Context, id uuid.UUID) (interface{}, error) {
			return r.queryByID(ctx, modelName, id, reg.QueryModifier)
		},

		CreateFunc: func(ctx context.Context, data map[string]interface{}) (interface{}, error) {
			if reg.BeforeSave != nil {
				if err := reg.BeforeSave(ctx, data); err != nil {
					return nil, err
				}
			}
			if reg.BeforeCreate != nil {
				if err := reg.BeforeCreate(ctx, data); err != nil {
					return nil, err
				}
			}
			return r.genericCreate(ctx, modelName, data)
		},

		UpdateFunc: func(ctx context.Context, id uuid.UUID, data map[string]interface{}) error {
			if reg.BeforeSave != nil {
				if err := reg.BeforeSave(ctx, data); err != nil {
					return err
				}
			}
			if reg.BeforeUpdate != nil {
				if err := reg.BeforeUpdate(ctx, data); err != nil {
					return err
				}
			}
			return r.genericUpdate(ctx, modelName, id, data)
		},

		DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
			return r.genericDelete(ctx, modelName, id)
		},
	}
	applyResourceDefaults(config)

	r.register(config)
	return nil
}

// Get retrieves a model config by name
func (r *Registry) Get(name string) (*ModelConfig, error) {
	key := strings.ToLower(name)
	config, ok := r.models[key]
	if !ok {
		return nil, fmt.Errorf("model %s not found", name)
	}
	return config, nil
}

// List returns all registered models in saved/registration order
func (r *Registry) List() []*ModelConfig {
	// Load saved order from database
	r.LoadModelOrder()

	configs := make([]*ModelConfig, 0, len(r.modelKeys))
	for _, key := range r.modelKeys {
		if config, ok := r.models[key]; ok {
			if config.Internal {
				continue
			}
			configs = append(configs, config)
		}
	}
	return configs
}

// register adds a model to the registry
func (r *Registry) register(config *ModelConfig) {
	key := strings.ToLower(config.Name)
	if _, exists := r.models[key]; !exists {
		// Track registration order
		r.modelKeys = append(r.modelKeys, key)
	}
	r.models[key] = config
}

func isEntResourceClient(modelClient reflect.Value) bool {
	if !modelClient.IsValid() || (modelClient.Kind() == reflect.Ptr && modelClient.IsNil()) {
		return false
	}
	for _, method := range []string{"Query", "Create", "Get", "UpdateOneID", "DeleteOneID"} {
		if !modelClient.MethodByName(method).IsValid() {
			return false
		}
	}
	return true
}

func entityTypeFromClient(modelClient reflect.Value) (reflect.Type, bool) {
	queryMethod := modelClient.MethodByName("Query")
	if !queryMethod.IsValid() {
		return nil, false
	}
	queryResults := queryMethod.Call(nil)
	if len(queryResults) != 1 {
		return nil, false
	}
	allMethod := queryResults[0].MethodByName("All")
	if !allMethod.IsValid() {
		return nil, false
	}
	allType := allMethod.Type()
	if allType.NumOut() != 2 {
		return nil, false
	}
	sliceType := allType.Out(0)
	if sliceType.Kind() != reflect.Slice {
		return nil, false
	}
	entityType := sliceType.Elem()
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}
	return entityType, entityType.Kind() == reflect.Struct
}

func extractRelationFields(entityType reflect.Type) []FieldConfig {
	edgesField, ok := entityType.FieldByName("Edges")
	if !ok {
		return nil
	}

	edgeType := edgesField.Type
	fields := make([]FieldConfig, 0, edgeType.NumField())
	for i := 0; i < edgeType.NumField(); i++ {
		edge := edgeType.Field(i)
		if !edge.IsExported() || edge.Name == "loadedTypes" {
			continue
		}
		fields = append(fields, FieldConfig{
			Name:       edge.Name,
			Label:      formatLabel(edge.Name),
			Type:       FieldTypeSelect,
			Readonly:   true,
			Editable:   false,
			Visible:    true,
			Width:      220,
			Primary:    false,
			Sortable:   false,
			Filterable: false,
		})
	}
	return fields
}

func appendMissingFields(fields []FieldConfig, additions ...FieldConfig) []FieldConfig {
	seen := make(map[string]bool, len(fields))
	for _, field := range fields {
		seen[field.Name] = true
	}
	for _, field := range additions {
		if !seen[field.Name] {
			fields = append(fields, field)
			seen[field.Name] = true
		}
	}
	return fields
}

func applyResourceDefaults(config *ModelConfig) {
	if config == nil {
		return
	}
	if config.Name == "AdminSetting" {
		config.Internal = true
		for i := range config.Fields {
			if config.Fields[i].Name == "Key" {
				config.Fields[i].System = true
				config.Fields[i].Visible = true
			}
		}
	}
	if config.Name == "User" {
		for i := range config.Fields {
			if config.Fields[i].Name == "IsActive" {
				config.Fields[i].Default = true
			}
		}
	}
	config.HiddenFields = hiddenFieldNames(config.Fields)
	config.ReadonlyFields = readonlyFieldNames(config.Fields)
}

func defaultResourceIcon(modelName string) string {
	switch modelName {
	case "User":
		return "👤"
	case "Post":
		return "✎"
	case "AdminSetting":
		return "⚙"
	default:
		return "▦"
	}
}
