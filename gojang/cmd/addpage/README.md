# Add Page Command

A command-line tool to automate the creation of static pages in Gojang.

## Overview

The `addpage` command automates the three-step process of adding a static page:
1. Creates the HTML template file
2. Adds the handler to `gojang/http/handlers/pages.go`
3. Registers the route in `gojang/http/routes/pages.go`

## Usage

Run the command from the project root:

```bash
go run ./gojang/cmd/addpage
```

Or build and run it:

```bash
go build -o addpage ./gojang/cmd/addpage
./addpage
```

## Interactive Prompts

The command will prompt you for:

1. **Page name**: A descriptive name (e.g., "About", "Contact", "Terms")
   - Must contain only letters and spaces
   - Used to generate the handler function name (e.g., "About" → `About()`)

2. **Page title**: The title displayed on the page (e.g., "About Us", "Contact Us")
   - Optional: defaults to the page name if not provided
   - Used in the template's `{{define "title"}}` block

3. **Route path**: The URL path for the page (e.g., "/about", "/contact")
   - Optional: defaults to lowercase page name with hyphens (e.g., "About Us" → "/about-us")
   - Must start with "/"

4. **Require authentication**: Whether the page requires login
   - Enter "y" or "yes" for protected pages
   - Enter "n" or "no" (or just press Enter) for public pages

## Example Usage

### Creating a Public Page

```
🚀 Gojang Static Page Generator
==================================

Page name (e.g., 'About', 'Contact', 'Terms'): About
Page title (e.g., 'About Us', 'Contact Us'): About Us
Route path (default: /about): 
Require authentication? (y/N): n

📝 Creating template file...
✅ Created: /home/user/project/gojang/views/templates/about.html

🔧 Adding handler to pages.go...
✅ Added handler: About

🔗 Adding route to pages.go...
✅ Added route: /about -> About

✨ Static page created successfully!

Next steps:
1. Restart your server: go run ./gojang/cmd/web
2. Visit: http://localhost:8080/about
3. Edit the template to customize your page
```

### Creating a Protected Page

```
🚀 Gojang Static Page Generator
==================================

Page name (e.g., 'About', 'Contact', 'Terms'): Settings
Page title (e.g., 'About Us', 'Contact Us'): Account Settings
Route path (default: /settings): 
Require authentication? (y/N): y

📝 Creating template file...
✅ Created: /home/user/project/gojang/views/templates/settings.html

🔧 Adding handler to pages.go...
✅ Added handler: Settings

🔗 Adding route to pages.go...
✅ Added route: /settings -> Settings

✨ Static page created successfully!

Next steps:
1. Restart your server: go run ./gojang/cmd/web
2. Visit: http://localhost:8080/settings
3. Edit the template to customize your page
```

## What Gets Created

### 1. Template File

Location: `gojang/views/templates/{page-name}.html`

The template includes:
- `{{define "title"}}` block with the page title
- `{{define "content"}}` block with placeholder content
- Basic card layout with helpful tips

### 2. Handler Function

Added to: `gojang/http/handlers/pages.go`

The handler includes:
- Proper function name (e.g., `About`, `Contact`, `Settings`)
- Template rendering with title and data
- Documentation comment

### 3. Route Registration

Added to: `gojang/http/routes/pages.go`

- Public pages: Added after the home route
- Protected pages: Added in the authenticated group after the dashboard route

## Customizing Generated Pages

After running the command, you can customize the generated template at:
`gojang/views/templates/{page-name}.html`

The handler and route are ready to use, but you can modify them in:
- `gojang/http/handlers/pages.go` - Add custom data or logic
- `gojang/http/routes/pages.go` - Change route patterns or add middleware

## Error Handling

The command will fail with helpful error messages if:
- Page name contains invalid characters (only letters and spaces allowed)
- Route path doesn't start with "/"
- Template file already exists
- Handler function already exists
- Route already exists
- Project root (go.mod) cannot be found

## Tips

- Use descriptive page names (e.g., "Terms of Service" instead of just "Terms")
- Keep route paths lowercase and use hyphens (e.g., "/terms-of-service")
- For pages that need custom data, edit the handler after generation
- For complex authentication logic, you can add middleware in routes.go

## See Also

- [Creating Static Pages Documentation](../../../docs/creating-static-pages.md)
- [Handler Documentation](../../http/handlers/README.md)
- [Routing Documentation](../../http/routes/README.md)
