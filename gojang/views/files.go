package views

import (
	"io/fs"
	"os"
	"path/filepath"
)

// TemplateFiles points to the developer-owned app tree.
// Any directory named "templates" under app can provide public templates.
var TemplateFiles fs.FS = appFS()

// StaticFiles points to the developer-owned app static directory.
var StaticFiles fs.FS = appViewsFS()

func appViewsFS() fs.FS {
	if root, err := findProjectRoot(); err == nil {
		return os.DirFS(filepath.Join(root, "app", "views"))
	}
	return os.DirFS(filepath.Join("app", "views"))
}

func appFS() fs.FS {
	if root, err := findProjectRoot(); err == nil {
		return os.DirFS(filepath.Join(root, "app"))
	}
	return os.DirFS("app")
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "app", "views")); err == nil {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
