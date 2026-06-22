package assets

import "embed"

// FS contains developer-owned public templates, static assets, and translations.
// Feature packages can add templates under app/<feature>/templates.
//
//go:embed */templates views/static views/i18n/*.json gojang/models/migrations
var FS embed.FS
