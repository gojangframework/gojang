# Add Model Command

A command-line tool to automate the creation of data models in Gojang.

## Overview

The `addmodel` command automates the complete process of adding a new data model to your Gojang application:

1. Creates the Ent schema file
2. Generates Ent code
3. Adds form validation struct
4. Creates handler with CRUD operations
5. Creates routes file
6. Registers routes in `main.go`
7. Creates HTML templates
8. Registers model with admin panel

## Features

✨ **Interactive & Non-Interactive Modes**: Use prompts or command-line flags  
🔍 **Dry-Run Support**: Preview changes before committing  
🛡️ **Input Validation**: Prevents Go reserved keywords and invalid names  
⏱️ **Timestamp Control**: Optional automatic created_at fields  
🎨 **Colored Output**: Easy-to-read console messages  
📚 **Built-in Examples**: `--examples` flag shows usage patterns  

## Quick Start

### See Examples

To see detailed usage examples with colors:

```bash
go run ./gojang/cmd/addmodel --examples
```

## Usage

### Interactive Mode

Run the command from the project root:

```bash
go run ./gojang/cmd/addmodel
```

Or build and run it:

```bash
go build -o addmodel ./gojang/cmd/addmodel
./addmodel
```

### Non-Interactive Mode (Command-Line Flags)

You can also use command-line flags for automation and scripting:

```bash
go run ./gojang/cmd/addmodel \
  --model Product \
  --icon "📦" \
  --fields "name:string:required,description:text,price:float:required,stock:int"
```

**Available flags:**
- `--model`: Model name (required in non-interactive mode)
- `--icon`: Model icon (default: "📄")
- `--fields`: Comma-separated fields in format `name:type` or `name:type:required`
- `--dry-run`: Preview changes without writing files
- `--timestamps`: Add created_at timestamp field (default: true, use `--timestamps=false` to disable)
- `--examples`: Show detailed usage examples and exit
- `-h`, `--help`: Show available flags

**Field format:** `name:type[:required]`
- `name`: Field name (lowercase, snake_case)
- `type`: Field type (string, text, int, float, bool, time)
- `required`: Optional, include to make field required

**Field name restrictions:**
- Cannot use Go reserved keywords (e.g., `for`, `func`, `if`, `return`, `type`, `var`, `const`, etc.)
- Must start with a lowercase letter
- Can only contain lowercase letters, numbers, and underscores

**Examples:**

Preview changes without creating files:
```bash
./addmodel --model Product --fields "name:string:required,price:float" --dry-run
```

Create a complete model non-interactively:
```bash
./addmodel \
  --model Article \
  --icon "📰" \
  --fields "title:string:required,content:text:required,published:bool,published_at:time"
```

Create a model without automatic timestamps:
```bash
./addmodel \
  --model SimpleTag \
  --icon "🏷️" \
  --fields "name:string:required" \
  --timestamps=false
```

## Interactive Prompts

The command will prompt you for:

### 1. Model Name

A singular, PascalCase name for your model (e.g., "Product", "Category", "Order")

- Must start with an uppercase letter
- Must contain only alphanumeric characters
- Used to generate all files and code

### 2. Model Icon

An emoji icon to represent your model in the admin panel (e.g., "📦", "🏷️", "📋")

- Optional: defaults to "📄" if not provided
- Makes the admin panel more visually appealing

### 3. Model Fields

Define the fields for your model using the format: `name:type`

**Supported types:**
- `string` - Short text (max 255 chars)
- `text` - Long text
- `int` - Integer number
- `float` - Decimal number
- `bool` - Boolean (true/false)
- `time` - Timestamp

**Field naming:**
- Must start with a lowercase letter
- Can contain lowercase letters, numbers, and underscores
- Use snake_case (e.g., `product_name`, `unit_price`)

**Examples:**
```
name:string
description:text
price:float
stock:int
is_active:bool
published_at:time
```

For `string` and `text` fields, you'll be asked if the field is required.

Press Enter without input to finish adding fields.

## Example Usage

### Creating a Product Model

```
$ go run ./gojang/cmd/addmodel

🚀 Gojang Data Model Generator
================================

Model name (e.g., 'Product', 'Category', 'Order'): Product
Model icon (e.g., '📦', '🏷️', '📋'): 📦

Enter fields for the model (press Enter without input to finish):
Format: name:type (e.g., 'name:string', 'price:float', 'stock:int', 'active:bool')
Supported types: string, text, int, float, bool, time
Field 1: name:string
   Is 'name' required? (Y/n): y
✅ Added: name (string)
Field 2: description:text
   Is 'description' required? (Y/n): n
✅ Added: description (text)
Field 3: price:float
✅ Added: price (float)
Field 4: stock:int
✅ Added: stock (int)
Field 5: 

Model summary:
  Name: Product
  Icon: 📦
  Fields:
    - name: string (required)
    - description: text
    - price: float
    - stock: int

Continue? (Y/n): y

📝 Step 1: Creating Ent schema...
✅ Created: /path/to/gojang/models/schema/product.go

⚙️  Step 2: Generating Ent code...
✅ Ent code generated

📝 Step 3: Adding form validation struct...
✅ Form validation struct added

📝 Step 4: Creating handler...
✅ Created: /path/to/gojang/http/handlers/products.go

📝 Step 5: Creating routes...
✅ Created: /path/to/gojang/http/routes/products.go

📝 Step 6: Registering routes in main.go...
✅ Routes registered

📝 Step 7: Creating templates...
✅ Created templates in: /path/to/gojang/views/templates/products

📝 Step 8: Registering with admin panel...
✅ Registered with admin panel

✨ Model created successfully!

Next steps:
1. Review the generated files and customize as needed
2. Restart your server: go run ./gojang/cmd/web
3. Visit: http://localhost:8080/products
4. Admin panel: http://localhost:8080/admin/products
```

## What Gets Created

### 1. Ent Schema

**Location:** `gojang/models/schema/{model}.go`

The Ent schema defines your database model with:
- Field definitions with appropriate types
- Validation rules (NotEmpty, Positive, etc.)
- Default values
- Automatic `created_at` timestamp

### 2. Generated Ent Code

The tool automatically runs `go generate ./...` in the models directory, creating:
- Model struct
- Create/Update/Query/Delete builders
- Type-safe field accessors

### 3. Form Validation Struct

**Added to:** `gojang/views/forms/forms.go`

A validation struct with:
- Go struct tags for form decoding
- Validation tags for input validation
- Proper Go types for all fields

### 4. Handler

**Location:** `gojang/http/handlers/{model}s.go`

Complete CRUD handler with:
- `Index()` - List all records
- `New()` - Show create form
- `Create()` - Save new record
- `Edit()` - Show edit form
- `Update()` - Save changes
- `Delete()` - Remove record

### 5. Routes

**Location:** `gojang/http/routes/{model}s.go`

Route definitions with:
- Public route for listing (GET /)
- Protected routes for mutations (require authentication)
- Proper HTTP methods (GET, POST, PUT, DELETE)

### 6. Main.go Registration

**Modified:** `gojang/cmd/web/main.go`

Adds:
- Handler initialization
- Route mounting

### 7. Templates

**Location:** `gojang/views/templates/{model}s/`

Three HTML templates:
- `index.html` - List view with table
- `new.partial.html` - Create form (modal)
- `edit.partial.html` - Update form (modal)

All templates include:
- HTMX integration for dynamic behavior
- Form validation error display
- Responsive layout

### 8. Admin Panel Registration

**Modified:** `gojang/admin/models.go`

Registers your model with:
- Model icon
- Plural name
- List fields to display
- Readonly fields

The admin panel automatically handles all CRUD operations using reflection.

## Customizing Generated Code

After generation, you can customize:

### Adding Relationships

Edit the schema file to add edges (relationships):

```go
func (Product) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("category", Category.Type).
            Ref("products").
            Unique(),
    }
}
```

Then run `go generate ./...` again.

### Adding Validation

Edit the form struct in `forms.go`:

```go
type ProductForm struct {
    Name  string  `form:"name" validate:"required,max=100"`
    Price float64 `form:"price" validate:"required,gt=0,lte=999999"`
}
```

### Customizing Templates

Edit the HTML templates to:
- Add custom styling
- Change field layouts
- Add additional features

### Adding Business Logic

Edit the handler to:
- Add authorization checks
- Transform data before saving
- Send notifications
- Trigger other actions

## Error Handling

The tool checks for existing files and will not overwrite:
- ✅ Schema files
- ✅ Handler files
- ✅ Routes files
- ✅ Template directories

If a file already exists, you'll get a clear error message.

## Troubleshooting

**"could not find project root"**
- Make sure you run the command from the project directory
- The tool looks for `go.mod` to find the root

**"Ent code generation failed"**
- Check that Ent is properly installed
- Try running `go generate ./...` manually in `gojang/models`

**"handler already exists"**
- The model was already created
- Delete the existing files if you want to regenerate

**Templates not rendering**
- Restart the server after generating the model
- Check that template files exist in the correct directory

## Tips

1. **Plan your fields** - Think about what data you need before starting
2. **Start simple** - You can always add more fields later
3. **Use meaningful names** - Choose clear, descriptive names for fields
4. **Review generated code** - Always review and customize as needed
5. **Test immediately** - Run the server and test CRUD operations right away

## See Also

- [Creating Data Models Documentation](../../../docs/creating-data-models.md)
- [Quick Start Guide](../../../docs/quick-start-data-model.md)
- [Ent Documentation](https://entgo.io/docs/getting-started)
- [Admin Panel Documentation](../../admin/README.md)
