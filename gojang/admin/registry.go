package admin

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/gojangframework/gojang/gojang/models"
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
	return &Registry{
		models: make(map[string]*ModelConfig),
		client: client,
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

	// Append custom fields if provided
	if reg.CustomFields != nil {
		fields = append(fields, reg.CustomFields...)
	}

	// Create config with generic CRUD operations
	config := &ModelConfig{
		Name:           modelName,
		NamePlural:     reg.NamePlural,
		Icon:           reg.Icon,
		Fields:         fields,
		ListFields:     reg.ListFields,
		HiddenFields:   reg.HiddenFields,
		ReadonlyFields: reg.ReadonlyFields,

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
			return r.genericCreate(ctx, modelName, data)
		},

		UpdateFunc: func(ctx context.Context, id uuid.UUID, data map[string]interface{}) error {
			if reg.BeforeSave != nil {
				if err := reg.BeforeSave(ctx, data); err != nil {
					return err
				}
			}
			return r.genericUpdate(ctx, modelName, id, data)
		},

		DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
			return r.genericDelete(ctx, modelName, id)
		},
	}

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
			configs = append(configs, config)
		}
	}
	return configs
}

// register adds a model to the registry
func (r *Registry) register(config *ModelConfig) {
	key := strings.ToLower(config.Name)
	r.models[key] = config
	// Track registration order
	r.modelKeys = append(r.modelKeys, key)
}
