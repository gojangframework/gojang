package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// createTemplates creates the template directory and files
func createTemplates(dir, modelName string, fields []Field) error {
	// Create directory
	if err := mkdir(dir, 0755); err != nil {
		return err
	}

	modelLower := strings.ToLower(modelName)
	modelPlural := modelLower + "s"
	modelTitle := toCamelCase(modelName)

	// Create index.html
	indexPath := filepath.Join(dir, "index.html")
	if err := createIndexTemplate(indexPath, modelName, modelTitle, modelPlural, fields); err != nil {
		return err
	}

	// Create new.partial.html
	newPath := filepath.Join(dir, "new.partial.html")
	if err := createFormTemplate(newPath, modelName, modelTitle, modelPlural, fields, "new"); err != nil {
		return err
	}

	// Create edit.partial.html
	editPath := filepath.Join(dir, "edit.partial.html")
	if err := createFormTemplate(editPath, modelName, modelTitle, modelPlural, fields, "edit"); err != nil {
		return err
	}

	return nil
}

// createIndexTemplate creates the index template
func createIndexTemplate(path, modelName, modelTitle, modelPlural string, fields []Field) error {
	// Build table headers
	var headers strings.Builder
	for i, field := range fields {
		if i < 4 { // Show first 4 fields
			headers.WriteString(fmt.Sprintf("                <th>%s</th>\n", toCamelCase(field.Name)))
		}
	}

	// Build table cells
	var cells strings.Builder
	for i, field := range fields {
		if i < 4 {
			fieldName := toCamelCase(field.Name)
			if field.Type == "float" {
				cells.WriteString(fmt.Sprintf("                <td>${{printf \"%%.2f\" .%s}}</td>\n", fieldName))
			} else if field.Type == "bool" {
				cells.WriteString(fmt.Sprintf("                <td>{{if .%s}}Yes{{else}}No{{end}}</td>\n", fieldName))
			} else {
				cells.WriteString(fmt.Sprintf("                <td>{{.%s}}</td>\n", fieldName))
			}
		}
	}

	content := fmt.Sprintf(`{{define "title"}}%s{{end}}

{{define "content"}}
<div class="container" style="padding: 2rem 2rem;">
	<h1>%s</h1>
	<h3>This is a sample page for demonstration only, not fully implemented.</h3>

    {{if .Data.%s}}
    <div class="table-container">
        <table class="table">
            <thead>
                <tr>
                    <th style="width: 80px;">#</th>
%s                </tr>
            </thead>
            <tbody>
                {{range .Data.%s}}
                <tr>
                    <td style="font-weight: 600; color: #64748b;">{{.ID}}</td>
%s                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
    {{else}}
    <div style="background: white; padding: 3rem; text-align: center; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1);">
        <p style="font-size: 1.25rem; color: #64748b; margin-bottom: 1rem;">📭 No %s found</p>
    </div>
    {{end}}
</div>
{{end}}
`, modelTitle, modelTitle, toCamelCase(modelName), headers.String(), toCamelCase(modelName), cells.String(), modelPlural)

	return writeFile(path, []byte(content), 0644)
}

// createFormTemplate creates new or edit form template
func createFormTemplate(path, modelName, modelTitle, modelPlural string, fields []Field, formType string) error {
	isEdit := formType == "edit"
	title := "New " + modelTitle
	action := "/" + modelPlural
	htmxAttr := `hx-post="` + action + `"`
	buttonText := "Create " + modelTitle

	if isEdit {
		title = "Edit " + modelTitle
		action = "/" + modelPlural + "/{{.Data." + toCamelCase(modelName) + ".ID}}"
		htmxAttr = `hx-put="` + "/" + modelPlural + "/{{.Data." + toCamelCase(modelName) + `.ID}}"`
		buttonText = "Update " + modelTitle
	}

	// Build form fields
	var formFields strings.Builder
	for _, field := range fields {
		fieldName := field.Name
		fieldTitle := toCamelCase(field.Name)
		inputType := getInputType(field.Type)

		value := ""
		if isEdit {
			if field.Type == "float" {
				value = fmt.Sprintf(`value="{{if .Data.Form}}{{.Data.Form.%s}}{{else}}{{.Data.%s.%s}}{{end}}"`, fieldTitle, toCamelCase(modelName), fieldTitle)
			} else if field.Type == "bool" {
				formFields.WriteString(fmt.Sprintf(`
    <div class="form-group">
        <label>
            <input type="checkbox" 
                   id="%s" 
                   name="%s"
                   {{if .Data.Form}}{{if .Data.Form.%s}}checked{{end}}{{else}}{{if .Data.%s.%s}}checked{{end}}{{end}}>
            %s
        </label>
    </div>
`, fieldName, fieldName, fieldTitle, toCamelCase(modelName), fieldTitle, fieldTitle))
				continue
			} else {
				value = fmt.Sprintf(`value="{{if .Data.Form}}{{.Data.Form.%s}}{{else}}{{.Data.%s.%s}}{{end}}"`, fieldTitle, toCamelCase(modelName), fieldTitle)
			}
		} else {
			if field.Type == "bool" {
				formFields.WriteString(fmt.Sprintf(`
    <div class="form-group">
        <label>
            <input type="checkbox" 
                   id="%s" 
                   name="%s"
                   {{if .Data.Form}}{{if .Data.Form.%s}}checked{{end}}{{end}}>
            %s
        </label>
    </div>
`, fieldName, fieldName, fieldTitle, fieldTitle))
				continue
			} else {
				value = fmt.Sprintf(`value="{{if .Data.Form}}{{.Data.Form.%s}}{{end}}"`, fieldTitle)
			}
		}

		if field.Type == "text" {
			formFields.WriteString(fmt.Sprintf(`
    <div class="form-group">
        <label for="%s">%s</label>
        <textarea id="%s" 
                  name="%s" 
                  rows="3"
                  class="form-control">{{if .Data.Form}}{{.Data.Form.%s}}{{else}}{{if .Data.%s}}{{.Data.%s.%s}}{{end}}{{end}}</textarea>
    </div>
`, fieldName, fieldTitle, fieldName, fieldName, fieldTitle, toCamelCase(modelName), toCamelCase(modelName), fieldTitle))
		} else if field.Type != "bool" {
			required := ""
			if field.Required {
				required = "\n               required"
			}
			step := ""
			if field.Type == "float" {
				step = "\n               step=\"0.01\""
			}
			formFields.WriteString(fmt.Sprintf(`
    <div class="form-group">
        <label for="%s">%s</label>
        <input type="%s" 
               id="%s" 
               name="%s" %s%s
               %s
               class="form-control">
    </div>
`, fieldName, fieldTitle, inputType, fieldName, fieldName, step, required, value))
		}
	}

	content := fmt.Sprintf(`{{define "title"}}%s{{end}}

{{define "content"}}
<div class="modal-header">
    <h2>%s</h2>
</div>

<form method="POST" action="%s" %s hx-swap="none">
    {{if .Data.Errors}}
    <div class="alert alert-danger">
        {{range .Data.Errors}}
        <p>{{.}}</p>
        {{end}}
    </div>
    {{end}}
%s
    <div class="form-actions">
        <button type="submit" class="btn btn-primary">%s</button>
        <button type="button" onclick="closeModal()" class="btn btn-secondary">Cancel</button>
    </div>
</form>
{{end}}
`, title, title, action, htmxAttr, formFields.String(), buttonText)

	return writeFile(path, []byte(content), 0644)
}
