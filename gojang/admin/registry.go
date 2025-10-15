package admin

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gojangframework/gojang/gojang/models"
	"github.com/gojangframework/gojang/gojang/models/setting"
)

// ModelConfig defines how a model should be displayed and managed in the admin panel
type ModelConfig struct {
	Name           string        // Display name (e.g., "User", "Post")
	NamePlural     string        // Plural name (e.g., "Users", "Posts")
	Icon           string        // Icon for navigation
	Fields         []FieldConfig // Auto-discovered fields
	ListFields     []string      // Fields to show in list view
	HiddenFields   []string      // Fields to hide
	ReadonlyFields []string      // Fields that can't be edited

	// CRUD operations
	QueryAll   func(ctx context.Context) ([]interface{}, error)
	QueryByID  func(ctx context.Context, id int) (interface{}, error)
	CreateFunc func(ctx context.Context, data map[string]interface{}) (interface{}, error)
	UpdateFunc func(ctx context.Context, id int, data map[string]interface{}) error
	DeleteFunc func(ctx context.Context, id int) error
}

// FieldConfig defines configuration for a single field
type FieldConfig struct {
	Name      string    // Field name (database column)
	Label     string    // Display label
	Type      FieldType // Field type
	Required  bool      // Is field required?
	Readonly  bool      // Is field readonly?
	Sensitive bool      // Is field sensitive (e.g., password)?
	Hidden    bool      // Hide from forms
	Help      string    // Help text shown below field
}

// FieldType represents the type of field
type FieldType string

const (
	FieldTypeString   FieldType = "string"
	FieldTypeText     FieldType = "text"
	FieldTypeInt      FieldType = "int"
	FieldTypeFloat    FieldType = "float"
	FieldTypeBool     FieldType = "bool"
	FieldTypeTime     FieldType = "time"
	FieldTypePassword FieldType = "password"
	FieldTypeEmail    FieldType = "email"
	FieldTypeSelect   FieldType = "select"
)

// AdminOverrides allows customizing auto-discovered models
type AdminOverrides struct {
	Icon           string
	NamePlural     string
	ListFields     []string
	HiddenFields   []string
	ReadonlyFields []string
	FieldLabels    map[string]string
	FieldTypes     map[string]FieldType
	OptionalFields []string
}

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

		QueryByID: func(ctx context.Context, id int) (interface{}, error) {
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

		UpdateFunc: func(ctx context.Context, id int, data map[string]interface{}) error {
			if reg.BeforeSave != nil {
				if err := reg.BeforeSave(ctx, data); err != nil {
					return err
				}
			}
			return r.genericUpdate(ctx, modelName, id, data)
		},

		DeleteFunc: func(ctx context.Context, id int) error {
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

// extractFields uses reflection to discover fields from a struct
func extractFields(example interface{}, override AdminOverrides) []FieldConfig {
	var fields []FieldConfig

	v := reflect.ValueOf(example)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields and special fields
		if !field.IsExported() || field.Name == "Edges" || field.Name == "selectValues" {
			continue
		}

		fieldName := field.Name

		// Check if hidden
		isHidden := contains(override.HiddenFields, fieldName)
		// Readonly defaults: ID, CreatedAt, UpdatedAt. Do not blanket-match all *At fields.
		isReadonly := contains(override.ReadonlyFields, fieldName) ||
			fieldName == "CreatedAt" ||
			fieldName == "UpdatedAt" ||
			fieldName == "ID"

		// Determine field type
		fieldType := detectFieldType(field.Type, fieldName, override.FieldTypes)

		// Get label
		label := override.FieldLabels[fieldName]
		if label == "" {
			label = formatLabel(fieldName)
		}

		// Determine if required
		// By default, non-bool, non-readonly/hidden fields are required.
		// If field is listed in OptionalFields or is a pointer (nil-able), treat as optional.
		isOptional := contains(override.OptionalFields, fieldName)
		// Pointers are optional as they can be nil
		isPointer := field.Type.Kind() == reflect.Ptr
		required := !isReadonly && !isHidden && fieldType != FieldTypeBool && !isOptional && !isPointer

		fields = append(fields, FieldConfig{
			Name:      fieldName,
			Label:     label,
			Type:      fieldType,
			Required:  required,
			Readonly:  isReadonly,
			Hidden:    isHidden,
			Sensitive: fieldName == "PasswordHash",
		})
	}

	return fields
}

// detectFieldType determines the field type from reflection
func detectFieldType(t reflect.Type, fieldName string, overrides map[string]FieldType) FieldType {
	// Check overrides first
	if override, ok := overrides[fieldName]; ok {
		return override
	}

	// Handle pointers
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Special cases
	if fieldName == "PasswordHash" {
		return FieldTypePassword
	}
	if fieldName == "Email" {
		return FieldTypeEmail
	}
	if fieldName == "Body" || fieldName == "Description" {
		return FieldTypeText
	}

	// Type-based detection
	switch t.Kind() {
	case reflect.String:
		return FieldTypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return FieldTypeInt
	case reflect.Float32, reflect.Float64:
		return FieldTypeFloat
	case reflect.Bool:
		return FieldTypeBool
	case reflect.Struct:
		if t.String() == "time.Time" {
			return FieldTypeTime
		}
	}

	return FieldTypeString
}

// formatLabel converts field name to display label
func formatLabel(fieldName string) string {
	// Insert spaces before capital letters
	var result strings.Builder
	for i, r := range fieldName {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}
	return result.String()
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// pluralize adds 's' to make plural (simple version)
func pluralize(name string) string {
	if strings.HasSuffix(name, "s") {
		return name + "es"
	}
	return name + "s"
}

// ExtractFieldValue extracts a field value from a struct using reflection
func ExtractFieldValue(obj interface{}, fieldName string) interface{} {
	v := reflect.ValueOf(obj)

	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	// Try direct field access
	field := v.FieldByName(fieldName)
	if field.IsValid() && field.CanInterface() {
		return formatFieldValue(field.Interface())
	}

	// Try Edges for related fields
	edges := v.FieldByName("Edges")
	if edges.IsValid() && edges.CanInterface() {
		edgeField := edges.FieldByName(fieldName)
		if edgeField.IsValid() && edgeField.CanInterface() {
			return formatFieldValue(edgeField.Interface())
		}
	}

	return nil
}

// formatFieldValue formats a field value for display
func formatFieldValue(val interface{}) interface{} {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case time.Time:
		if v.IsZero() {
			return ""
		}
		return v.Format("Jan 2, 2006 3:04 PM")
	case *time.Time:
		if v == nil || v.IsZero() {
			return ""
		}
		return v.Format("Jan 2, 2006 3:04 PM")
	case bool:
		if v {
			return "✓"
		}
		return "✗"
	case *models.User:
		if v != nil {
			return v.Email
		}
		return ""
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// queryAll retrieves all records for a model using Ent client with reflection
func (r *Registry) queryAll(ctx context.Context, modelName string, modifier AfterLoadHook) ([]interface{}, error) {
	// Get the model client using reflection (e.g., r.client.User)
	clientVal := reflect.ValueOf(r.client).Elem()
	modelClient := clientVal.FieldByName(modelName)

	if !modelClient.IsValid() {
		return nil, fmt.Errorf("model %s not found on client", modelName)
	}

	// Call Query() method
	queryMethod := modelClient.MethodByName("Query")
	if !queryMethod.IsValid() {
		return nil, fmt.Errorf("query method not found for model %s", modelName)
	}

	queryResults := queryMethod.Call([]reflect.Value{})
	if len(queryResults) == 0 {
		return nil, fmt.Errorf("query method returned no results for model %s", modelName)
	}

	query := queryResults[0].Interface()

	// Apply modifier if provided
	if modifier != nil {
		query = modifier(ctx, query)
	}

	// Call All(ctx) method
	queryVal := reflect.ValueOf(query)
	allMethod := queryVal.MethodByName("All")
	if !allMethod.IsValid() {
		return nil, fmt.Errorf("all method not found for model %s", modelName)
	}

	allResults := allMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(allResults) != 2 {
		return nil, fmt.Errorf("all method returned unexpected number of values for model %s", modelName)
	}

	// Check for error
	if !allResults[1].IsNil() {
		return nil, allResults[1].Interface().(error)
	}

	// Convert slice to []interface{}
	recordsVal := allResults[0]
	results := make([]interface{}, recordsVal.Len())
	for i := 0; i < recordsVal.Len(); i++ {
		results[i] = recordsVal.Index(i).Interface()
	}

	return results, nil
}

// queryByID retrieves a single record by ID using reflection
func (r *Registry) queryByID(ctx context.Context, modelName string, id int, _ AfterLoadHook) (interface{}, error) {
	// Get the model client using reflection (e.g., r.client.User)
	clientVal := reflect.ValueOf(r.client).Elem()
	modelClient := clientVal.FieldByName(modelName)

	if !modelClient.IsValid() {
		return nil, fmt.Errorf("model %s not found on client", modelName)
	}

	// Call Get(ctx, id) method
	getMethod := modelClient.MethodByName("Get")
	if !getMethod.IsValid() {
		return nil, fmt.Errorf("Get method not found for model %s", modelName)
	}

	getResults := getMethod.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(id),
	})

	if len(getResults) != 2 {
		return nil, fmt.Errorf("Get method returned unexpected number of values for model %s", modelName)
	}

	// Check for error
	if !getResults[1].IsNil() {
		return nil, getResults[1].Interface().(error)
	}

	return getResults[0].Interface(), nil
}

// genericCreate creates a new record using reflection
func (r *Registry) genericCreate(ctx context.Context, modelName string, data map[string]interface{}) (interface{}, error) {
	// Get the model client using reflection (e.g., r.client.User)
	clientVal := reflect.ValueOf(r.client).Elem()
	modelClient := clientVal.FieldByName(modelName)

	if !modelClient.IsValid() {
		return nil, fmt.Errorf("model %s not found on client", modelName)
	}

	// Call Create() method
	createMethod := modelClient.MethodByName("Create")
	if !createMethod.IsValid() {
		return nil, fmt.Errorf("Create method not found for model %s", modelName)
	}

	createResults := createMethod.Call([]reflect.Value{})
	if len(createResults) == 0 {
		return nil, fmt.Errorf("Create method returned no results for model %s", modelName)
	}

	builder := createResults[0].Interface()

	// Set fields on builder
	if err := setFieldsOnBuilder(builder, data); err != nil {
		return nil, err
	}

	// Call Save(ctx) method
	builderVal := reflect.ValueOf(builder)
	saveMethod := builderVal.MethodByName("Save")
	if !saveMethod.IsValid() {
		return nil, fmt.Errorf("save method not found for model %s", modelName)
	}

	saveResults := saveMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(saveResults) != 2 {
		return nil, fmt.Errorf("save method returned unexpected number of values for model %s", modelName)
	}

	// Check for error
	if !saveResults[1].IsNil() {
		return nil, saveResults[1].Interface().(error)
	}

	return saveResults[0].Interface(), nil
}

// genericUpdate updates a record using reflection
func (r *Registry) genericUpdate(ctx context.Context, modelName string, id int, data map[string]interface{}) error {
	// Get the model client using reflection (e.g., r.client.User)
	clientVal := reflect.ValueOf(r.client).Elem()
	modelClient := clientVal.FieldByName(modelName)

	if !modelClient.IsValid() {
		return fmt.Errorf("model %s not found on client", modelName)
	}

	// Call UpdateOneID(id) method
	updateMethod := modelClient.MethodByName("UpdateOneID")
	if !updateMethod.IsValid() {
		return fmt.Errorf("UpdateOneID method not found for model %s", modelName)
	}

	updateResults := updateMethod.Call([]reflect.Value{reflect.ValueOf(id)})
	if len(updateResults) == 0 {
		return fmt.Errorf("UpdateOneID method returned no results for model %s", modelName)
	}

	builder := updateResults[0].Interface()

	// Set fields on builder
	if err := setFieldsOnBuilder(builder, data); err != nil {
		return err
	}

	// Call Save(ctx) method
	builderVal := reflect.ValueOf(builder)
	saveMethod := builderVal.MethodByName("Save")
	if !saveMethod.IsValid() {
		return fmt.Errorf("save method not found for model %s", modelName)
	}

	saveResults := saveMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(saveResults) != 2 {
		return fmt.Errorf("save method returned unexpected number of values for model %s", modelName)
	}

	// Check for error
	if !saveResults[1].IsNil() {
		return saveResults[1].Interface().(error)
	}

	return nil
}

// genericDelete deletes a record using reflection
func (r *Registry) genericDelete(ctx context.Context, modelName string, id int) error {
	// Get the model client using reflection (e.g., r.client.User)
	clientVal := reflect.ValueOf(r.client).Elem()
	modelClient := clientVal.FieldByName(modelName)

	if !modelClient.IsValid() {
		return fmt.Errorf("model %s not found on client", modelName)
	}

	// Call DeleteOneID(id) method
	deleteMethod := modelClient.MethodByName("DeleteOneID")
	if !deleteMethod.IsValid() {
		return fmt.Errorf("DeleteOneID method not found for model %s", modelName)
	}

	deleteResults := deleteMethod.Call([]reflect.Value{reflect.ValueOf(id)})
	if len(deleteResults) == 0 {
		return fmt.Errorf("DeleteOneID method returned no results for model %s", modelName)
	}

	deleter := deleteResults[0]

	// Call Exec(ctx) method
	execMethod := deleter.MethodByName("Exec")
	if !execMethod.IsValid() {
		return fmt.Errorf("exec method not found for model %s", modelName)
	}

	execResults := execMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(execResults) != 1 {
		return fmt.Errorf("exec method returned unexpected number of values for model %s", modelName)
	}

	// Check for error
	if !execResults[0].IsNil() {
		return execResults[0].Interface().(error)
	}

	return nil
}

// setFieldsOnBuilder sets fields on an Ent builder using reflection
func setFieldsOnBuilder(builder interface{}, data map[string]interface{}) error {
	builderVal := reflect.ValueOf(builder)

	// For each field in data, call the appropriate Set method
	for fieldName, value := range data {
		// Skip nil values and empty strings for non-required fields
		if value == nil || (value == "" && fieldName != "Body") {
			continue
		}

		// Build the setter method name (e.g., "SetEmail" for "Email")
		setterName := "Set" + fieldName
		method := builderVal.MethodByName(setterName)

		if !method.IsValid() {
			// Try with "ID" suffix for foreign keys (e.g., "SetAuthorID")
			setterName = "Set" + fieldName + "ID"
			method = builderVal.MethodByName(setterName)
			if !method.IsValid() {
				continue // Skip fields without setters
			}
		}

		// Call the setter method
		valueToSet := reflect.ValueOf(value)
		method.Call([]reflect.Value{valueToSet})
	}

	return nil
}

// LoadModelOrder loads the saved model order from database and applies it
func (r *Registry) LoadModelOrder() {
	ctx := context.Background()

	// Try to load saved order from settings
	settingRecord, err := r.client.Setting.Query().
		Where(setting.KeyEQ("admin_model_order")).
		Only(ctx)

	if err != nil || settingRecord == nil {
		// No saved order, keep registration order
		return
	}

	// Parse JSON array from value
	var savedOrder []string
	value := strings.Trim(settingRecord.Value, "[]\"")
	if value == "" {
		return
	}

	// Simple JSON array parsing
	savedOrder = strings.Split(value, "\",\"")
	for i := range savedOrder {
		savedOrder[i] = strings.Trim(savedOrder[i], "\"")
	}

	// Build new order based on saved preferences
	newKeys := make([]string, 0, len(r.modelKeys))
	usedKeys := make(map[string]bool)

	// First, add models in saved order
	for _, name := range savedOrder {
		key := strings.ToLower(name)
		if _, exists := r.models[key]; exists {
			newKeys = append(newKeys, key)
			usedKeys[key] = true
		}
	}

	// Then add any new models not in saved order
	for _, key := range r.modelKeys {
		if !usedKeys[key] {
			newKeys = append(newKeys, key)
		}
	}

	r.modelKeys = newKeys
}

// SaveModelOrder saves the current model order to database
func (r *Registry) SaveModelOrder(order []string) error {
	ctx := context.Background()

	// Convert order to JSON array string
	var orderNames []string
	for _, key := range order {
		if config, ok := r.models[strings.ToLower(key)]; ok {
			orderNames = append(orderNames, config.Name)
		}
	}

	// Build JSON manually
	jsonValue := "[\"" + strings.Join(orderNames, "\",\"") + "\"]"

	// Check if setting exists
	existing, err := r.client.Setting.Query().
		Where(setting.KeyEQ("admin_model_order")).
		Only(ctx)

	if err != nil {
		// Create new setting
		_, err = r.client.Setting.Create().
			SetKey("admin_model_order").
			SetValue(jsonValue).
			Save(ctx)
	} else {
		// Update existing setting
		err = r.client.Setting.UpdateOne(existing).
			SetValue(jsonValue).
			Exec(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to save model order: %w", err)
	}

	// Update in-memory order
	r.modelKeys = make([]string, len(order))
	for i, name := range order {
		r.modelKeys[i] = strings.ToLower(name)
	}

	return nil
}
