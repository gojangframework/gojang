package views

import (
	"io/fs"

	appassets "github.com/gojangframework/gojang/app"
)

// TemplateFiles points to the embedded developer-owned app tree.
// Any immediate app child directory named "templates" can provide public templates.
var TemplateFiles fs.FS = appassets.FS

// StaticFiles points to the developer-owned app static directory.
var StaticFiles fs.FS = mustSub(appassets.FS, "views")

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
