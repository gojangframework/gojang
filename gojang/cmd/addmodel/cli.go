package main

import (
	"fmt"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

// colorize adds ANSI color codes to text
func colorize(color, text string) string {
	return color + text + colorReset
}

// showExamples displays usage examples
func showExamples() {
	fmt.Println(colorize(colorCyan, "\n🚀 Gojang Add Model - Usage Examples"))
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println(colorize(colorYellow, "\n1. Interactive Mode (Default):"))
	fmt.Println("   go run ./gojang/cmd/addmodel")
	fmt.Println("   # You'll be prompted for model name, icon, and fields")

	fmt.Println(colorize(colorYellow, "\n2. Non-Interactive Mode:"))
	fmt.Println("   go run ./gojang/cmd/addmodel \\")
	fmt.Println("     --model Product \\")
	fmt.Println("     --icon '📦' \\")
	fmt.Println("     --fields 'name:string:required,price:float:required,stock:int'")

	fmt.Println(colorize(colorYellow, "\n3. Preview with Dry-Run:"))
	fmt.Println("   go run ./gojang/cmd/addmodel \\")
	fmt.Println("     --model Product \\")
	fmt.Println("     --fields 'name:string:required,price:float' \\")
	fmt.Println("     --dry-run")

	fmt.Println(colorize(colorYellow, "\n4. Without Timestamps:"))
	fmt.Println("   go run ./gojang/cmd/addmodel \\")
	fmt.Println("     --model Tag \\")
	fmt.Println("     --fields 'name:string:required' \\")
	fmt.Println("     --timestamps=false")

	fmt.Println(colorize(colorYellow, "\n5. Complex Example:"))
	fmt.Println("   go run ./gojang/cmd/addmodel \\")
	fmt.Println("     --model Article \\")
	fmt.Println("     --icon '📰' \\")
	fmt.Println("     --fields 'title:string:required,content:text:required,published:bool,views:int'")

	fmt.Println(colorize(colorGreen, "\n📝 Field Format:"))
	fmt.Println("   name:type[:required]")
	fmt.Println("   - name: lowercase, snake_case (e.g., 'user_name', 'created_by')")
	fmt.Println("   - type: string, text, int, float, bool, time")
	fmt.Println("   - required: optional suffix to make field required")

	fmt.Println(colorize(colorRed, "\n⚠️  Restrictions:"))
	fmt.Println("   - Cannot use Go reserved keywords (for, func, if, return, etc.)")
	fmt.Println("   - Cannot use Go built-in types (String, Int, Int16, Error, etc.)")
	fmt.Println("   - Cannot use Ent predeclared identifiers (Client, Mutation, Config, Query, Tx, Value, Hook, Policy, etc.)")
	fmt.Println("   - Field names must start with lowercase letter")
	fmt.Println("   - Model names must start with uppercase letter (PascalCase)")

	fmt.Println(colorize(colorBlue, "\n📚 For more information:"))
	fmt.Println("   See: gojang/cmd/addmodel/README.md")
	fmt.Println()
}
