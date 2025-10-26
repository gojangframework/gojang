package admin

import (
	"context"

	"github.com/google/uuid"
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
	QueryAll          func(ctx context.Context) ([]interface{}, error)
	QueryAllPaginated func(ctx context.Context, limit, offset int) ([]interface{}, error)
	CountAll          func(ctx context.Context) (int, error)
	QueryByID         func(ctx context.Context, id uuid.UUID) (interface{}, error)
	CreateFunc        func(ctx context.Context, data map[string]interface{}) (interface{}, error)
	UpdateFunc        func(ctx context.Context, id uuid.UUID, data map[string]interface{}) error
	DeleteFunc        func(ctx context.Context, id uuid.UUID) error
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
