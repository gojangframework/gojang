package admin

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gojangframework/gojang/gojang/models"
)

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
