package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Field represents a model field
type Field struct {
	Name     string
	Type     string
	Required bool
}

var dryRun bool

func main() {
	// Command-line flags
	modelNameFlag := flag.String("model", "", "Model name (e.g., 'Product', 'Category', 'Order')")
	modelIconFlag := flag.String("icon", "📄", "Model icon (e.g., '📦', '🏷️', '📋')")
	fieldsFlag := flag.String("fields", "", "Comma-separated fields (e.g., 'name:string:required,price:float,stock:int')")
	dryRunFlag := flag.Bool("dry-run", false, "Preview changes without writing files")
	timestampsFlag := flag.Bool("timestamps", true, "Add created_at and updated_at fields (default: true)")
	helpExamples := flag.Bool("examples", false, "Show usage examples and exit")
	flag.Parse()

	// Show examples if requested
	if *helpExamples {
		showExamples()
		return
	}

	fmt.Println("🚀 Gojang Data Model Generator")
	fmt.Println("================================")
	fmt.Println()

	var modelName, modelIcon string
	var fields []Field

	// Check if running in non-interactive mode
	if *modelNameFlag != "" {
		modelName = *modelNameFlag
		modelIcon = *modelIconFlag

		// Parse fields from flag
		if *fieldsFlag == "" {
			log.Fatal("❌ Fields are required when using --model flag")
		}

		fields = parseFieldsFromString(*fieldsFlag)
	} else {
		// Interactive mode
		modelName, modelIcon, fields = runInteractiveMode()
	}

	// Validate model name
	if !isValidModelName(modelName) {
		log.Fatal("❌ Model name must start with uppercase letter, contain only alphanumeric characters, and not be a Go keyword, built-in type, or Ent predeclared identifier (String, Int, Error, Client, Mutation, Config, Query, Tx, Value, Hook, Policy, etc.)")
	}

	if len(fields) == 0 {
		log.Fatal("❌ At least one field is required")
	}

	// Print summary
	printSummary(modelName, modelIcon, fields, *dryRunFlag)

	// Skip confirmation in non-interactive mode
	if *modelNameFlag == "" && !confirmCreation() {
		fmt.Println("❌ Cancelled")
		return
	}

	// Find project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("❌ Failed to find project root: %v", err)
	}

	// Set global dry-run flag
	dryRun = *dryRunFlag

	// Execute model creation steps
	executeModelCreation(projectRoot, modelName, modelIcon, fields, *timestampsFlag, *dryRunFlag)
}

// parseFieldsFromString parses the fields flag string into Field structs
func parseFieldsFromString(fieldsStr string) []Field {
	var fields []Field
	fieldSpecs := strings.Split(fieldsStr, ",")

	for _, spec := range fieldSpecs {
		parts := strings.Split(strings.TrimSpace(spec), ":")
		if len(parts) < 2 {
			log.Fatalf("❌ Invalid field format: %s (expected name:type or name:type:required)", spec)
		}

		field := Field{
			Name:     parts[0],
			Type:     parts[1],
			Required: len(parts) > 2 && parts[2] == "required",
		}

		// Validate field
		if err := validateField(field); err != nil {
			log.Fatalf("❌ %v", err)
		}

		fields = append(fields, field)
	}

	return fields
}

// validateField validates a single field
func validateField(field Field) error {
	// Validate field name format
	if !isValidFieldName(field.Name) {
		return fmt.Errorf("invalid field name: %s (must start with lowercase letter and contain only alphanumeric and underscore)", field.Name)
	}

	// Check for reserved keywords
	if isReservedKeyword(field.Name) {
		return fmt.Errorf("field name '%s' is a Go reserved keyword, built-in type, or Ent identifier", field.Name)
	}

	// Validate field type
	validTypes := map[string]bool{
		"string": true, "text": true, "int": true,
		"float": true, "bool": true, "time": true,
	}
	if !validTypes[field.Type] {
		return fmt.Errorf("invalid field type: %s (supported: string, text, int, float, bool, time)", field.Type)
	}

	return nil
}

// isValidFieldName checks if field name follows conventions
func isValidFieldName(name string) bool {
	// Field names must start with lowercase and contain only alphanumeric and underscore
	for i, r := range name {
		if i == 0 {
			if r < 'a' || r > 'z' {
				return false
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
				return false
			}
		}
	}
	return len(name) > 0
}

// runInteractiveMode runs the interactive prompts for model creation
func runInteractiveMode() (string, string, []Field) {
	reader := bufio.NewReader(os.Stdin)

	// Get model name
	fmt.Print("Model name (e.g., 'Product', 'Category', 'Order'): ")
	modelName, _ := reader.ReadString('\n')
	modelName = strings.TrimSpace(modelName)

	if modelName == "" {
		log.Fatal("❌ Model name is required")
	}

	// Get model icon (optional)
	fmt.Print("Model icon (optional, e.g., '📦', '🏷️', '📋') [default: 📄]: ")
	modelIcon, _ := reader.ReadString('\n')
	modelIcon = strings.TrimSpace(modelIcon)
	if modelIcon == "" {
		modelIcon = "📄" // Default icon
	}

	// Get model fields
	fmt.Println()
	fmt.Println("Enter fields for the model (press Enter without input to finish):")
	fmt.Println("Format: name:type (e.g., 'name:string', 'price:float', 'stock:int', 'active:bool')")
	fmt.Println("Supported types: string, text, int, float, bool, time")

	var fields []Field
	for {
		fmt.Printf("Field %d: ", len(fields)+1)
		fieldInput, _ := reader.ReadString('\n')
		fieldInput = strings.TrimSpace(fieldInput)

		if fieldInput == "" {
			break
		}

		field, err := parseField(fieldInput)
		if err != nil {
			fmt.Printf("⚠️  %v. Try again.\n", err)
			continue
		}

		// Ask if field is required
		fmt.Printf("   Is '%s' required? (Y/n): ", field.Name)
		required, _ := reader.ReadString('\n')
		required = strings.TrimSpace(strings.ToLower(required))
		field.Required = required != "n" && required != "no"

		fields = append(fields, field)
		fmt.Printf("✅ Added: %s (%s)\n", field.Name, field.Type)
	}

	if len(fields) == 0 {
		log.Fatal("❌ At least one field is required")
	}

	return modelName, modelIcon, fields
}

// printSummary prints the model creation summary
func printSummary(modelName, modelIcon string, fields []Field, isDryRun bool) {
	fmt.Println()
	if isDryRun {
		fmt.Println("🔍 DRY RUN MODE - No files will be created")
		fmt.Println()
	}
	fmt.Println("Model summary:")
	fmt.Printf("  Name: %s\n", modelName)
	fmt.Printf("  Icon: %s\n", modelIcon)
	fmt.Println("  Fields:")
	for _, field := range fields {
		req := ""
		if field.Required {
			req = " (required)"
		}
		fmt.Printf("    - %s: %s%s\n", field.Name, field.Type, req)
	}
	fmt.Println()
}

// confirmCreation asks for user confirmation
func confirmCreation() bool {
	fmt.Print("Continue? (Y/n): ")
	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	return confirm != "n" && confirm != "no"
}

// executeModelCreation runs all the steps to create a model
func executeModelCreation(projectRoot, modelName, modelIcon string, fields []Field, includeTimestamps, isDryRun bool) {
	// Step 1: Create Ent schema
	fmt.Println()
	fmt.Println("📝 Step 1: Creating Ent schema...")
	schemaPath := filepath.Join(projectRoot, "gojang", "models", "schema", strings.ToLower(modelName)+".go")
	if err := createSchema(schemaPath, modelName, fields, includeTimestamps); err != nil {
		log.Fatalf("❌ Failed to create schema: %v", err)
	}
	fmt.Printf("✅ Created: %s\n", schemaPath)

	// Step 2: Generate Ent code
	fmt.Println()
	fmt.Println("⚙️  Step 2: Generating Ent code...")
	modelsDir := filepath.Join(projectRoot, "gojang", "models")
	if err := generateEntCode(modelsDir); err != nil {
		log.Fatalf("❌ Failed to generate Ent code: %v", err)
	}
	fmt.Println("✅ Ent code generated")

	// Step 3: Create form struct
	fmt.Println()
	fmt.Println("📝 Step 3: Adding form validation struct...")
	formsPath := filepath.Join(projectRoot, "gojang", "views", "forms", "forms.go")
	if err := addFormStruct(formsPath, modelName, fields); err != nil {
		log.Fatalf("❌ Failed to add form struct: %v", err)
	}
	fmt.Println("✅ Form validation struct added")

	// Step 4: Create handler
	fmt.Println()
	fmt.Println("📝 Step 4: Creating handler...")
	handlerPath := filepath.Join(projectRoot, "gojang", "http", "handlers", strings.ToLower(modelName)+"s.go")
	if err := createHandler(handlerPath, modelName, fields); err != nil {
		log.Fatalf("❌ Failed to create handler: %v", err)
	}
	fmt.Printf("✅ Created: %s\n", handlerPath)

	// Step 5: Create routes
	fmt.Println()
	fmt.Println("📝 Step 5: Creating routes...")
	routesPath := filepath.Join(projectRoot, "gojang", "http", "routes", strings.ToLower(modelName)+"s.go")
	if err := createRoutes(routesPath, modelName); err != nil {
		log.Fatalf("❌ Failed to create routes: %v", err)
	}
	fmt.Printf("✅ Created: %s\n", routesPath)

	// Step 6: Update main.go
	fmt.Println()
	fmt.Println("📝 Step 6: Registering routes in main.go...")
	mainPath := filepath.Join(projectRoot, "gojang", "cmd", "web", "main.go")
	if err := updateMainGo(mainPath, modelName); err != nil {
		log.Fatalf("❌ Failed to update main.go: %v", err)
	}
	fmt.Println("✅ Routes registered")

	// Step 7: Create templates
	fmt.Println()
	fmt.Println("📝 Step 7: Creating templates...")
	templateDir := filepath.Join(projectRoot, "gojang", "views", "templates", strings.ToLower(modelName)+"s")
	if err := createTemplates(templateDir, modelName, fields); err != nil {
		log.Fatalf("❌ Failed to create templates: %v", err)
	}
	fmt.Printf("✅ Created templates in: %s\n", templateDir)

	// Step 8: Register with admin panel
	fmt.Println()
	fmt.Println("📝 Step 8: Registering with admin panel...")
	adminModelsPath := filepath.Join(projectRoot, "gojang", "admin", "models.go")
	if err := registerWithAdmin(adminModelsPath, modelName, modelIcon, fields); err != nil {
		log.Fatalf("❌ Failed to register with admin: %v", err)
	}
	fmt.Println("✅ Registered with admin panel")

	// Success message
	printSuccessMessage(modelName, isDryRun, schemaPath, modelsDir, formsPath, handlerPath, routesPath, mainPath, templateDir, adminModelsPath)
}

// printSuccessMessage prints the final success message
func printSuccessMessage(modelName string, isDryRun bool, paths ...string) {
	fmt.Println()
	if isDryRun {
		fmt.Println("✨ Dry run completed successfully!")
		fmt.Println()
		fmt.Println("Files that would be created/modified:")
		for _, path := range paths {
			if strings.Contains(path, "models") && !strings.Contains(path, "schema") {
				fmt.Printf("  - %s (generated)\n", path)
			} else if strings.Contains(path, "forms") || strings.Contains(path, "main.go") || strings.Contains(path, "admin") {
				fmt.Printf("  - %s (modified)\n", path)
			} else if strings.Contains(path, "templates") {
				fmt.Printf("  - %s (directory with templates)\n", path)
			} else {
				fmt.Printf("  - %s\n", path)
			}
		}
		fmt.Println()
		fmt.Println("Run without --dry-run flag to create the model")
	} else {
		fmt.Println("✨ Model created successfully!")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("1. Review the generated files and customize as needed")
		fmt.Println("2. Restart your server: go run ./gojang/cmd/web")
		fmt.Printf("3. Visit: http://localhost:8080/%ss\n", strings.ToLower(modelName))
		fmt.Printf("4. Admin panel: http://localhost:8080/admin/%ss\n", strings.ToLower(modelName))
	}
}
