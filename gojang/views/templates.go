package views

import "embed"

//go:embed templates
var TemplateFiles embed.FS

//go:embed static
var StaticFiles embed.FS
