package main

import (
	"fmt"
	"strings"
)

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == ' ' || r == '-'
	})
	for i, word := range words {
		words[i] = strings.Title(strings.ToLower(word))
	}
	return strings.Join(words, "")
}

// toCamelCase converts field name to camelCase for Go struct field
func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) == 0 {
		return pascal
	}
	return strings.ToUpper(pascal[:1]) + pascal[1:]
}

// getGoType returns the Go type for a field type
func getGoType(fieldType string) string {
	switch fieldType {
	case "string", "text":
		return "string"
	case "int":
		return "int"
	case "float":
		return "float64"
	case "bool":
		return "bool"
	case "time":
		return "time.Time"
	default:
		return "string"
	}
}

// getEntFieldType returns the Ent field type for a given type
func getEntFieldType(fieldType string) string {
	switch fieldType {
	case "string":
		return "String"
	case "text":
		return "Text"
	case "int":
		return "Int"
	case "float":
		return "Float"
	case "bool":
		return "Bool"
	case "time":
		return "Time"
	default:
		return "String"
	}
}

// getValidationTag returns the validation tag for a field
func getValidationTag(field Field) string {
	switch field.Type {
	case "string":
		if field.Required {
			return "required,max=255"
		}
		return "omitempty,max=255"
	case "text":
		if field.Required {
			return "required"
		}
		return "omitempty"
	case "int":
		return "gte=0"
	case "float":
		return "gt=0"
	case "bool":
		return "omitempty"
	case "time":
		return "omitempty"
	default:
		return "omitempty"
	}
}

// getInputType returns the HTML input type for a field type
func getInputType(fieldType string) string {
	switch fieldType {
	case "string", "text":
		return "text"
	case "int":
		return "number"
	case "float":
		return "number"
	case "bool":
		return "checkbox"
	case "time":
		return "datetime-local"
	default:
		return "text"
	}
}

// buildFormFieldExtraction generates the code to extract form fields
func buildFormFieldExtraction(fields []Field) string {
	var builder strings.Builder
	for _, field := range fields {
		fieldName := toCamelCase(field.Name)

		switch field.Type {
		case "int":
			builder.WriteString(fmt.Sprintf("\t\t%s:        func() int { v, _ := strconv.Atoi(r.Form.Get(\"%s\")); return v }(),\n", fieldName, field.Name))
		case "float":
			builder.WriteString(fmt.Sprintf("\t\t%s:        func() float64 { v, _ := strconv.ParseFloat(r.Form.Get(\"%s\"), 64); return v }(),\n", fieldName, field.Name))
		case "bool":
			builder.WriteString(fmt.Sprintf("\t\t%s:        r.Form.Get(\"%s\") == \"on\" || r.Form.Get(\"%s\") == \"true\" || r.Form.Get(\"%s\") == \"1\",\n", fieldName, field.Name, field.Name, field.Name))
		case "time":
			builder.WriteString(fmt.Sprintf("\t\t%s:        func() time.Time { v, _ := time.Parse(\"2006-01-02T15:04\", r.Form.Get(\"%s\")); return v }(),\n", fieldName, field.Name))
		default:
			builder.WriteString(fmt.Sprintf("\t\t%s:        r.Form.Get(\"%s\"),\n", fieldName, field.Name))
		}
	}
	return builder.String()
}
