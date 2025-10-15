package main

import (
	"fmt"
	"regexp"
	"strings"
)

// isValidModelName checks if the model name is a valid Go identifier and not a reserved name
func isValidModelName(name string) bool {
	matched, _ := regexp.MatchString(`^[A-Z][A-Za-z0-9]*$`, name)
	if !matched {
		return false
	}
	// Check if it's a reserved name
	return !isReservedKeyword(name)
}

// isReservedKeyword checks if the name is a Go reserved keyword, built-in type, or Ent predeclared identifier
func isReservedKeyword(name string) bool {
	// Go reserved keywords (both lowercase and capitalized)
	keywords := map[string]bool{
		"break": true, "Break": true,
		"case": true, "Case": true,
		"chan": true, "Chan": true,
		"const": true, "Const": true,
		"continue": true, "Continue": true,
		"default": true, "Default": true,
		"defer": true, "Defer": true,
		"else": true, "Else": true,
		"fallthrough": true, "Fallthrough": true,
		"for": true, "For": true,
		"func": true, "Func": true,
		"go": true, "Go": true,
		"goto": true, "Goto": true,
		"if": true, "If": true,
		"import": true, "Import": true,
		"interface": true, "Interface": true,
		"map": true, "Map": true,
		"package": true, "Package": true,
		"range": true, "Range": true,
		"return": true, "Return": true,
		"select": true, "Select": true,
		"struct": true, "Struct": true,
		"switch": true, "Switch": true,
		"type": true, "Type": true,
		"var": true, "Var": true,
	}

	// Go built-in types (both lowercase for fields and uppercase for models)
	builtinTypes := map[string]bool{
		// Numeric types
		"int": true, "Int": true,
		"int8": true, "Int8": true,
		"int16": true, "Int16": true,
		"int32": true, "Int32": true,
		"int64": true, "Int64": true,
		"uint": true, "Uint": true,
		"uint8": true, "Uint8": true,
		"uint16": true, "Uint16": true,
		"uint32": true, "Uint32": true,
		"uint64": true, "Uint64": true,
		"float32": true, "Float32": true,
		"float64": true, "Float64": true,
		"complex64": true, "Complex64": true,
		"complex128": true, "Complex128": true,
		"byte": true, "Byte": true,
		"rune": true, "Rune": true,
		// String & boolean
		"string": true, "String": true,
		"bool": true, "Bool": true,
		// Other built-in types
		"error": true, "Error": true,
		"any": true, "Any": true,
	}

	// Ent predeclared identifiers (case-insensitive for model names)
	entIdentifiers := map[string]bool{
		"client": true, "Client": true,
		"mutation": true, "Mutation": true,
		"config": true, "Config": true,
		"query": true, "Query": true,
		"tx": true, "Tx": true,
		"value": true, "Value": true,
		"hook": true, "Hook": true,
		"policy": true, "Policy": true,
		"orderfunc": true, "OrderFunc": true,
		"predicate": true, "Predicate": true,
	}

	return keywords[name] || builtinTypes[name] || entIdentifiers[name]
}

// parseField parses a field input string
func parseField(input string) (Field, error) {
	parts := strings.Split(input, ":")
	if len(parts) != 2 {
		return Field{}, fmt.Errorf("invalid format, use 'name:type'")
	}

	name := strings.TrimSpace(parts[0])
	fieldType := strings.TrimSpace(strings.ToLower(parts[1]))

	// Validate field name
	if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(name) {
		return Field{}, fmt.Errorf("field name must start with lowercase letter and contain only alphanumeric and underscore")
	}

	// Check for Go reserved keywords
	if isReservedKeyword(name) {
		return Field{}, fmt.Errorf("field name '%s' is a Go reserved keyword or built-in type", name)
	}

	// Validate field type
	validTypes := map[string]bool{
		"string": true, "text": true, "int": true,
		"float": true, "bool": true, "time": true,
	}
	if !validTypes[fieldType] {
		return Field{}, fmt.Errorf("unsupported type '%s'", fieldType)
	}

	return Field{
		Name:     name,
		Type:     fieldType,
		Required: false,
	}, nil
}
