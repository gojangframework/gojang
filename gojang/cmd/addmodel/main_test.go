package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsValidModelName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		// Valid names
		{"User", true},
		{"Product", true},
		{"OrderItem", true},
		{"MyModel123", true},

		// Invalid: lowercase
		{"user", false},
		{"myModel", false},

		// Invalid: Go keywords
		{"Type", false},
		{"Interface", false},
		{"Package", false},

		// Invalid: Go built-in types
		{"String", false},
		{"Int", false},
		{"Int16", false},
		{"Int32", false},
		{"Int64", false},
		{"Uint", false},
		{"Uint8", false},
		{"Uint16", false},
		{"Uint32", false},
		{"Uint64", false},
		{"Float32", false},
		{"Float64", false},
		{"Bool", false},
		{"Byte", false},
		{"Rune", false},
		{"Error", false},
		{"Any", false},

		// Invalid: Ent predeclared identifiers
		{"Client", false},
		{"Mutation", false},
		{"Config", false},
		{"Query", false},
		{"Tx", false},
		{"Value", false},
		{"Hook", false},
		{"Policy", false},
		{"OrderFunc", false},
		{"Predicate", false},

		// Invalid: special characters
		{"User-Name", false},
		{"User_Name", false},
		{"User Name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidModelName(tt.name)
			if got != tt.valid {
				t.Errorf("isValidModelName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestParseField(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantField Field
		wantErr   bool
	}{
		{
			name:      "valid string field",
			input:     "name:string",
			wantField: Field{Name: "name", Type: "string", Required: false},
			wantErr:   false,
		},
		{
			name:      "valid int field",
			input:     "count:int",
			wantField: Field{Name: "count", Type: "int", Required: false},
			wantErr:   false,
		},
		{
			name:      "valid float field",
			input:     "price:float",
			wantField: Field{Name: "price", Type: "float", Required: false},
			wantErr:   false,
		},
		{
			name:      "valid bool field",
			input:     "active:bool",
			wantField: Field{Name: "active", Type: "bool", Required: false},
			wantErr:   false,
		},
		{
			name:      "valid text field",
			input:     "description:text",
			wantField: Field{Name: "description", Type: "text", Required: false},
			wantErr:   false,
		},
		{
			name:      "valid time field",
			input:     "created:time",
			wantField: Field{Name: "created", Type: "time", Required: false},
			wantErr:   false,
		},
		{
			name:      "field with underscore",
			input:     "unit_price:float",
			wantField: Field{Name: "unit_price", Type: "float", Required: false},
			wantErr:   false,
		},
		{
			name:    "invalid format - no colon",
			input:   "namestring",
			wantErr: true,
		},
		{
			name:    "invalid format - multiple colons",
			input:   "name:string:extra",
			wantErr: true,
		},
		{
			name:    "invalid field name - uppercase",
			input:   "Name:string",
			wantErr: true,
		},
		{
			name:    "invalid field name - starts with number",
			input:   "1name:string",
			wantErr: true,
		},
		{
			name:    "invalid type",
			input:   "name:invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseField(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseField(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Name != tt.wantField.Name || got.Type != tt.wantField.Type {
					t.Errorf("parseField(%q) = %+v, want %+v", tt.input, got, tt.wantField)
				}
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "product", "Product"},
		{"snake_case", "product_name", "ProductName"},
		{"kebab-case", "product-name", "ProductName"},
		{"space separated", "product name", "ProductName"},
		{"already PascalCase", "ProductName", "Productname"},
		{"mixed", "product_name-test", "ProductNameTest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toPascalCase(tt.input)
			if got != tt.want {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "name", "Name"},
		{"snake_case", "product_name", "ProductName"},
		{"kebab-case", "product-name", "ProductName"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toCamelCase(tt.input)
			if got != tt.want {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetEntFieldType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"string", "String"},
		{"text", "Text"},
		{"int", "Int"},
		{"float", "Float"},
		{"bool", "Bool"},
		{"time", "Time"},
		{"unknown", "String"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := getEntFieldType(tt.input)
			if got != tt.want {
				t.Errorf("getEntFieldType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetGoType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"string", "string"},
		{"text", "string"},
		{"int", "int"},
		{"float", "float64"},
		{"bool", "bool"},
		{"time", "time.Time"},
		{"unknown", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := getGoType(tt.input)
			if got != tt.want {
				t.Errorf("getGoType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetValidationTag(t *testing.T) {
	tests := []struct {
		name  string
		field Field
		want  string
	}{
		{
			name:  "required string",
			field: Field{Name: "name", Type: "string", Required: true},
			want:  "required,max=255",
		},
		{
			name:  "optional string",
			field: Field{Name: "name", Type: "string", Required: false},
			want:  "omitempty,max=255",
		},
		{
			name:  "required text",
			field: Field{Name: "desc", Type: "text", Required: true},
			want:  "required",
		},
		{
			name:  "optional text",
			field: Field{Name: "desc", Type: "text", Required: false},
			want:  "omitempty",
		},
		{
			name:  "int field",
			field: Field{Name: "count", Type: "int", Required: false},
			want:  "gte=0",
		},
		{
			name:  "float field",
			field: Field{Name: "price", Type: "float", Required: false},
			want:  "gt=0",
		},
		{
			name:  "bool field",
			field: Field{Name: "active", Type: "bool", Required: false},
			want:  "omitempty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getValidationTag(tt.field)
			if got != tt.want {
				t.Errorf("getValidationTag(%+v) = %q, want %q", tt.field, got, tt.want)
			}
		})
	}
}

func TestGetInputType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"string", "text"},
		{"text", "text"},
		{"int", "number"},
		{"float", "number"},
		{"bool", "checkbox"},
		{"time", "datetime-local"},
		{"unknown", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := getInputType(tt.input)
			if got != tt.want {
				t.Errorf("getInputType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCreateSchema(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "product.go")

	fields := []Field{
		{Name: "name", Type: "string", Required: true},
		{Name: "price", Type: "float", Required: false},
		{Name: "stock", Type: "int", Required: false},
	}

	err := createSchema(schemaPath, "Product", fields, true)
	if err != nil {
		t.Fatalf("createSchema failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read schema file: %v", err)
	}

	contentStr := string(content)
	expectedStrings := []string{
		"package schema",
		"type Product struct",
		"func (Product) Fields()",
		`field.UUID("id", uuid.UUID{})`,
		`Default(uuid.New)`,
		`field.String("name")`,
		`field.Float("price")`,
		`field.Int("stock")`,
		`field.Time("created_at")`,
		"NotEmpty()",
		"Positive()",
		"Default(0)",
		`"github.com/google/uuid"`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Schema content missing expected string: %q", expected)
		}
	}
}

func TestCreateSchema_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "product.go")

	// Create the file first
	os.WriteFile(schemaPath, []byte("existing content"), 0644)

	fields := []Field{{Name: "name", Type: "string", Required: true}}
	err := createSchema(schemaPath, "Product", fields, true)

	if err == nil {
		t.Error("Expected error for existing file, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' error, got: %v", err)
	}
}

func TestAddFormStruct(t *testing.T) {
	tmpDir := t.TempDir()
	formsPath := filepath.Join(tmpDir, "forms.go")

	// Create initial forms.go
	initialContent := `package forms

// Validate validates a form struct
func Validate(form interface{}) map[string]string {
	return nil
}
`
	os.WriteFile(formsPath, []byte(initialContent), 0644)

	fields := []Field{
		{Name: "name", Type: "string", Required: true},
		{Name: "price", Type: "float", Required: false},
	}

	err := addFormStruct(formsPath, "Product", fields)
	if err != nil {
		t.Fatalf("addFormStruct failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(formsPath)
	if err != nil {
		t.Fatalf("Failed to read forms file: %v", err)
	}

	contentStr := string(content)
	expectedStrings := []string{
		"type ProductForm struct",
		"Name string",
		"Price float64",
		`form:"name"`,
		`form:"price"`,
		`validate:"required,max=255"`,
		`validate:"gt=0"`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Forms content missing expected string: %q", expected)
		}
	}
}

func TestCreateHandler(t *testing.T) {
	tmpDir := t.TempDir()
	handlerPath := filepath.Join(tmpDir, "products.go")

	fields := []Field{
		{Name: "name", Type: "string", Required: true},
		{Name: "price", Type: "float", Required: false},
	}

	err := createHandler(handlerPath, "Product", fields)
	if err != nil {
		t.Fatalf("createHandler failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(handlerPath)
	if err != nil {
		t.Fatalf("Failed to read handler file: %v", err)
	}

	contentStr := string(content)
	expectedStrings := []string{
		"package handlers",
		"type ProductHandler struct",
		"func NewProductHandler",
		"func (h *ProductHandler) Index",
		"func (h *ProductHandler) New",
		"func (h *ProductHandler) Create",
		"func (h *ProductHandler) Edit",
		"func (h *ProductHandler) Update",
		"func (h *ProductHandler) Delete",
		"SetName(form.Name)",
		"SetPrice(form.Price)",
		`"github.com/google/uuid"`,
		"uuid.Parse(idStr)",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Handler content missing expected string: %q", expected)
		}
	}
}

func TestCreateRoutes(t *testing.T) {
	tmpDir := t.TempDir()
	routesPath := filepath.Join(tmpDir, "products.go")

	err := createRoutes(routesPath, "Product")
	if err != nil {
		t.Fatalf("createRoutes failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(routesPath)
	if err != nil {
		t.Fatalf("Failed to read routes file: %v", err)
	}

	contentStr := string(content)
	expectedStrings := []string{
		"package routes",
		"func ProductRoutes",
		"*handlers.ProductHandler",
		`r.Get("/", handler.Index)`,
		`auth.Get("/new", handler.New)`,
		`auth.Post("/", handler.Create)`,
		`auth.Get("/{id}/edit", handler.Edit)`,
		`auth.Put("/{id}", handler.Update)`,
		`auth.Delete("/{id}", handler.Delete)`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Routes content missing expected string: %q", expected)
		}
	}
}

func TestUpdateMainGo(t *testing.T) {
	tmpDir := t.TempDir()
	mainPath := filepath.Join(tmpDir, "main.go")

	// Create initial main.go with minimal structure
	initialContent := `package main

func main() {
	postHandler := handlers.NewPostHandler(client, publicRenderer)

	r.Mount("/posts", routes.PostRoutes(postHandler, sessionManager, client))
	r.Mount("/admin", admin.AdminRoutes(adminHandler, sessionManager, client))
}
`
	os.WriteFile(mainPath, []byte(initialContent), 0644)

	err := updateMainGo(mainPath, "Product")
	if err != nil {
		t.Fatalf("updateMainGo failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("Failed to read main file: %v", err)
	}

	contentStr := string(content)
	expectedStrings := []string{
		"productHandler := handlers.NewProductHandler(client, publicRenderer)",
		`r.Mount("/products", routes.ProductRoutes(productHandler, sessionManager, client))`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Main.go content missing expected string: %q", expected)
		}
	}
}

func TestCreateIndexTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "index.html")

	fields := []Field{
		{Name: "name", Type: "string", Required: true},
		{Name: "price", Type: "float", Required: false},
	}

	err := createIndexTemplate(indexPath, "Product", "Product", "products", fields)
	if err != nil {
		t.Fatalf("createIndexTemplate failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read template file: %v", err)
	}

	contentStr := string(content)
	expectedStrings := []string{
		`{{define "title"}}Product{{end}}`,
		`{{define "content"}}`,
		`<h1>Product</h1>`,
		`<th>Name</th>`,
		`<th>Price</th>`,
		`{{.Name}}`,
		`{{printf "%.2f" .Price}}`,
	}

	// Verify that buttons are NOT included
	unwantedStrings := []string{
		`href="/products/new"`,
		`{{if .User}}`,
		`{{if $.User}}`,
		`btn btn-primary`,
		`btn btn-sm btn-primary`,
		`btn btn-sm btn-danger`,
		`Edit`,
		`Delete`,
		`Actions`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Index template missing expected string: %q", expected)
		}
	}

	for _, unwanted := range unwantedStrings {
		if strings.Contains(contentStr, unwanted) {
			t.Errorf("Index template should not contain: %q", unwanted)
		}
	}
}

func TestCreateFormTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	newPath := filepath.Join(tmpDir, "new.partial.html")

	fields := []Field{
		{Name: "name", Type: "string", Required: true},
		{Name: "description", Type: "text", Required: false},
		{Name: "price", Type: "float", Required: false},
		{Name: "active", Type: "bool", Required: false},
	}

	err := createFormTemplate(newPath, "Product", "Product", "products", fields, "new")
	if err != nil {
		t.Fatalf("createFormTemplate failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("Failed to read template file: %v", err)
	}

	contentStr := string(content)
	expectedStrings := []string{
		`{{define "title"}}New Product{{end}}`,
		`{{define "content"}}`,
		`<h2>New Product</h2>`,
		`action="/products"`,
		`hx-post="/products"`,
		`<label for="name">Name</label>`,
		`<input type="text"`,
		`name="name"`,
		`required`,
		`<label for="description">Description</label>`,
		`<textarea`,
		`<label for="price">Price</label>`,
		`type="number"`,
		`step="0.01"`,
		`<input type="checkbox"`,
		`name="active"`,
		`Create Product`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Form template missing expected string: %q", expected)
		}
	}
}

func TestRegisterWithAdmin(t *testing.T) {
	tmpDir := t.TempDir()
	adminPath := filepath.Join(tmpDir, "models.go")

	// Create initial admin models.go
	initialContent := `package admin

func RegisterModels(registry *Registry) {
	// Existing models
}
`
	os.WriteFile(adminPath, []byte(initialContent), 0644)

	fields := []Field{
		{Name: "name", Type: "string", Required: true},
		{Name: "price", Type: "float", Required: false},
	}

	err := registerWithAdmin(adminPath, "Product", "📦", fields)
	if err != nil {
		t.Fatalf("registerWithAdmin failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(adminPath)
	if err != nil {
		t.Fatalf("Failed to read admin file: %v", err)
	}

	contentStr := string(content)
	expectedStrings := []string{
		"registry.RegisterModel(ModelRegistration{",
		"ModelType:      &models.Product{}",
		`Icon:           "📦"`,
		`NamePlural:     "Products"`,
		`ListFields:     []string{"ID", "Name", "Price"}`,
		`ReadonlyFields: []string{"ID", "CreatedAt"}`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Admin registration missing expected string: %q", expected)
		}
	}
}

func TestDryRunMode(t *testing.T) {
	// Save original dryRun state
	originalDryRun := dryRun
	defer func() { dryRun = originalDryRun }()

	// Enable dry-run mode
	dryRun = true

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.go")

	// Try to write a file in dry-run mode
	err := writeFile(testPath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("writeFile in dry-run mode failed: %v", err)
	}

	// File should not exist
	if _, err := os.Stat(testPath); err == nil {
		t.Error("File should not exist in dry-run mode")
	}

	// Disable dry-run mode
	dryRun = false

	// Write file normally
	err = writeFile(testPath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("writeFile failed: %v", err)
	}

	// File should exist
	if _, err := os.Stat(testPath); err != nil {
		t.Errorf("File should exist after normal write: %v", err)
	}
}

func TestMkdirDryRunMode(t *testing.T) {
	// Save original dryRun state
	originalDryRun := dryRun
	defer func() { dryRun = originalDryRun }()

	// Enable dry-run mode
	dryRun = true

	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "testdir")

	// Try to create directory in dry-run mode
	err := mkdir(testDir, 0755)
	if err != nil {
		t.Fatalf("mkdir in dry-run mode failed: %v", err)
	}

	// Directory should not exist
	if _, err := os.Stat(testDir); err == nil {
		t.Error("Directory should not exist in dry-run mode")
	}

	// Disable dry-run mode
	dryRun = false

	// Create directory normally
	err = mkdir(testDir, 0755)
	if err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	// Directory should exist
	if _, err := os.Stat(testDir); err != nil {
		t.Errorf("Directory should exist after normal mkdir: %v", err)
	}
}

func TestIsReservedKeyword(t *testing.T) {
	tests := []struct {
		name     string
		keyword  string
		expected bool
	}{
		// Go keywords
		{"go keyword", "for", true},
		{"go keyword", "func", true},
		{"go keyword", "if", true},
		{"go keyword", "return", true},
		{"go keyword", "type", true},

		// Ent predeclared identifiers (uppercase)
		{"ent identifier", "Client", true},
		{"ent identifier", "Mutation", true},
		{"ent identifier", "Config", true},
		{"ent identifier", "Query", true},
		{"ent identifier", "Tx", true},
		{"ent identifier", "Value", true},
		{"ent identifier", "Hook", true},
		{"ent identifier", "Policy", true},
		{"ent identifier", "OrderFunc", true},
		{"ent identifier", "Predicate", true},

		// Ent predeclared identifiers (lowercase)
		{"ent identifier lowercase", "client", true},
		{"ent identifier lowercase", "mutation", true},
		{"ent identifier lowercase", "config", true},
		{"ent identifier lowercase", "query", true},
		{"ent identifier lowercase", "tx", true},

		// Valid names
		{"valid name", "name", false},
		{"valid name", "user_id", false},
		{"valid name", "price", false},
		{"valid name", "Product", false},
		{"valid name", "Customer", false},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_"+tt.keyword, func(t *testing.T) {
			result := isReservedKeyword(tt.keyword)
			if result != tt.expected {
				t.Errorf("isReservedKeyword(%q) = %v, want %v", tt.keyword, result, tt.expected)
			}
		})
	}
}

func TestParseField_ReservedKeywords(t *testing.T) {
	reservedKeywords := []string{
		// Go keywords
		"for", "func", "if", "return", "type", "var", "const",
		// Ent predeclared identifiers (lowercase for field names)
		"client", "mutation", "config", "query", "tx",
	}

	for _, keyword := range reservedKeywords {
		_, err := parseField(keyword + ":string")
		if err == nil {
			t.Errorf("parseField should reject reserved keyword: %s", keyword)
		}
		if !strings.Contains(err.Error(), "reserved keyword") {
			t.Errorf("parseField error should mention reserved keyword, got: %v", err)
		}
	}
}
