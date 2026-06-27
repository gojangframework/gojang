package admin

import (
	"fmt"
	"html/template"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func workspaceTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"workspaceFields":         workspaceFields,
		"drawerFields":            drawerFields,
		"cellValue":               cellValue,
		"inputValue":              inputValue,
		"boolValue":               boolValue,
		"boolCreateValue":         boolCreateValue,
		"canInlineEdit":           canInlineEdit,
		"canEditRecordField":      canEditRecordField,
		"fieldReadonlyForRecord":  fieldReadonlyForRecord,
		"inputType":               inputType,
		"gridQuery":               gridQuery,
		"gridQueryWithoutFilter":  gridQueryWithoutFilter,
		"gridQueryWithoutRelated": gridQueryWithoutRelated,
		"fieldSelected":           fieldSelected,
		"relationURL":             relationURL,
		"relationSummary":         relationSummary,
		"dict":                    dict,
	}
}

func dict(values ...interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			continue
		}
		result[key] = values[i+1]
	}
	return result
}

func workspaceFields(config *ModelConfig) []FieldConfig {
	if config == nil {
		return nil
	}
	byName := map[string]FieldConfig{}
	for _, field := range config.Fields {
		byName[field.Name] = field
	}
	fields := make([]FieldConfig, 0, len(config.ListFields))
	for _, name := range config.ListFields {
		if field, ok := byName[name]; ok && field.Visible && !field.Hidden {
			fields = append(fields, field)
		}
	}
	if len(fields) > 0 {
		return fields
	}
	for _, field := range config.Fields {
		if field.Visible && !field.Hidden {
			fields = append(fields, field)
		}
	}
	return fields
}

func fieldSelected(grid *resourcePageData, fieldName string) bool {
	return grid != nil && grid.SelectedFields[fieldName]
}

func gridQuery(grid *resourcePageData, page int) template.URL {
	return template.URL(gridQueryValues(grid, page).Encode())
}

func gridQueryWithoutFilter(grid *resourcePageData, page int) template.URL {
	values := gridQueryValues(grid, page)
	values.Del("filter_field")
	values.Del("filter_op")
	values.Del("filter_value")
	return template.URL(values.Encode())
}

func gridQueryWithoutRelated(grid *resourcePageData, page int) template.URL {
	values := gridQueryValues(grid, page)
	values.Del("related_from")
	values.Del("related_field")
	values.Del("related_id")
	return template.URL(values.Encode())
}

func gridQueryValues(grid *resourcePageData, page int) url.Values {
	values := url.Values{}
	values.Set("view", workspaceViewGrid)
	if page < 1 {
		page = 1
	}
	values.Set("page", strconv.Itoa(page))
	if grid == nil {
		values.Set("per_page", "50")
		return values
	}
	values.Set("per_page", strconv.Itoa(grid.PerPage))
	for _, field := range grid.Fields {
		values.Add("fields", field.Name)
	}
	if grid.Filter.Valid {
		values.Set("filter_field", grid.Filter.Field.Name)
		values.Set("filter_op", grid.Filter.Op)
		values.Set("filter_value", grid.Filter.Value)
	}
	if grid.Sort.Valid {
		values.Set("sort_field", grid.Sort.Field.Name)
		values.Set("sort_dir", grid.Sort.Dir)
	}
	if grid.Related.Valid {
		values.Set("related_from", grid.Related.SourceResource)
		values.Set("related_field", grid.Related.SourceField.Name)
		values.Set("related_id", grid.Related.SourceID.String())
	}
	return values
}

func drawerFields(config *ModelConfig) []FieldConfig {
	if config == nil {
		return nil
	}
	fields := make([]FieldConfig, 0, len(config.Fields))
	for _, field := range config.Fields {
		if !field.Hidden {
			fields = append(fields, field)
		}
	}
	return fields
}

func cellValue(record interface{}, field FieldConfig) string {
	if field.Relation {
		return relationSummary(record, field)
	}
	value := ExtractFieldValue(record, field.Name)
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func relationURL(config *ModelConfig, record interface{}, field FieldConfig) template.URL {
	if config == nil || record == nil || !field.Relation || field.RelationTarget == "" {
		return ""
	}
	id := getIDValue(record)
	if id == "" {
		return ""
	}
	values := url.Values{}
	values.Set("view", workspaceViewGrid)
	values.Set("related_from", config.Name)
	values.Set("related_field", field.Name)
	values.Set("related_id", id)
	return template.URL("/admin/t/" + strings.ToLower(field.RelationTarget) + "?" + values.Encode())
}

func relationSummary(record interface{}, field FieldConfig) string {
	if !field.Relation {
		return ""
	}
	value := rawFieldValue(record, field.Name)
	if value == nil {
		return "Open related " + field.RelationTarget
	}
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "No related " + field.RelationTarget
		}
		return cellLabel(value)
	}
	if v.Kind() == reflect.Slice {
		count := v.Len()
		if count == 0 {
			return "No related " + pluralize(field.RelationTarget)
		}
		if count == 1 {
			return "1 related " + field.RelationTarget
		}
		return fmt.Sprintf("%d related %s", count, pluralize(field.RelationTarget))
	}
	return fmt.Sprint(value)
}

func cellLabel(record interface{}) string {
	if record == nil {
		return ""
	}
	for _, name := range []string{"Email", "Name", "Title", "Subject", "Key", "ID"} {
		if value := rawFieldValue(record, name); value != nil {
			return fmt.Sprint(formatFieldValue(value))
		}
	}
	return fmt.Sprint(record)
}

func inputValue(record interface{}, field FieldConfig) string {
	value := rawFieldValue(record, field.Name)
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case time.Time:
		if v.IsZero() {
			return ""
		}
		if field.Type == FieldTypeTime {
			return v.Format("2006-01-02T15:04")
		}
		return v.Format(time.RFC3339)
	case *time.Time:
		if v == nil || v.IsZero() {
			return ""
		}
		return v.Format("2006-01-02T15:04")
	default:
		return fmt.Sprint(v)
	}
}

func boolValue(record interface{}, field FieldConfig) bool {
	value := rawFieldValue(record, field.Name)
	if b, ok := value.(bool); ok {
		return b
	}
	return false
}

func boolCreateValue(field FieldConfig) bool {
	if value, ok := field.Default.(bool); ok {
		return value
	}
	return false
}

func canEditRecordField(config *ModelConfig, record interface{}, field FieldConfig) bool {
	return canInlineEdit(field) && !isProtectedRecordField(config, record, field.Name)
}

func fieldReadonlyForRecord(config *ModelConfig, record interface{}, field FieldConfig) bool {
	return field.Readonly || !field.Editable || isProtectedRecordField(config, record, field.Name)
}

func isProtectedRecordField(config *ModelConfig, record interface{}, fieldName string) bool {
	if config == nil || config.Name != "AdminSetting" {
		return false
	}
	if fieldName == "Key" {
		return record != nil
	}
	return fieldName == "Value" && isAdminSettingRecord(record)
}

func isAdminSettingRecord(record interface{}) bool {
	key, _ := rawFieldValue(record, "Key").(string)
	return strings.HasPrefix(key, "admin.")
}

func inputType(field FieldConfig) string {
	switch field.Type {
	case FieldTypeEmail:
		return "email"
	case FieldTypePassword:
		return "password"
	case FieldTypeInt, FieldTypeFloat:
		return "number"
	case FieldTypeTime:
		return "datetime-local"
	default:
		return "text"
	}
}

func rawFieldValue(obj interface{}, fieldName string) interface{} {
	if obj == nil {
		return nil
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	field := v.FieldByName(fieldName)
	if field.IsValid() && field.CanInterface() {
		return field.Interface()
	}
	edges := v.FieldByName("Edges")
	if edges.IsValid() {
		edgeField := edges.FieldByName(fieldName)
		if edgeField.IsValid() && edgeField.CanInterface() {
			return edgeField.Interface()
		}
	}
	return nil
}
