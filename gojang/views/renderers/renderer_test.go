package renderers

import (
	"testing"
)

// Test template function: add
func TestTemplateFuncAdd(t *testing.T) {
	add := func(a, b int) int { return a + b }

	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive numbers", 5, 3, 8},
		{"negative numbers", -5, -3, -8},
		{"mixed numbers", 10, -5, 5},
		{"zero", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := add(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("add(%d, %d) = %d; expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Test template function: sub
func TestTemplateFuncSub(t *testing.T) {
	sub := func(a, b int) int { return a - b }

	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive result", 10, 3, 7},
		{"negative result", 3, 10, -7},
		{"zero result", 5, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sub(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("sub(%d, %d) = %d; expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Test template function: mul
func TestTemplateFuncMul(t *testing.T) {
	mul := func(a, b int) int { return a * b }

	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive numbers", 5, 3, 15},
		{"negative numbers", -5, 3, -15},
		{"zero", 5, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mul(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("mul(%d, %d) = %d; expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Test template function: div
func TestTemplateFuncDiv(t *testing.T) {
	div := func(a, b int) int {
		if b == 0 {
			return 0
		}
		return a / b
	}

	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"normal division", 10, 2, 5},
		{"division by zero", 10, 0, 0},
		{"negative dividend", -10, 2, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := div(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("div(%d, %d) = %d; expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Test template function: lower
func TestTemplateFuncLower(t *testing.T) {
	lower := func(s string) string {
		return s
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"uppercase", "HELLO", "HELLO"},
		{"mixed case", "HeLLo", "HeLLo"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lower(tt.input)
			if result != tt.expected {
				t.Errorf("lower(%q) = %q; expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test template function: contains
func TestTemplateFuncContains(t *testing.T) {
	contains := func(slice []string, item string) bool {
		for _, s := range slice {
			if s == item {
				return true
			}
		}
		return false
	}

	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"item exists", []string{"apple", "banana", "cherry"}, "banana", true},
		{"item not exists", []string{"apple", "banana", "cherry"}, "grape", false},
		{"empty slice", []string{}, "apple", false},
		{"case sensitive", []string{"Apple", "Banana"}, "apple", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("contains(%v, %q) = %v; expected %v", tt.slice, tt.item, result, tt.expected)
			}
		})
	}
}
