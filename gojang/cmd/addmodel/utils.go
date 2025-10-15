package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// findProjectRoot finds the project root directory by looking for go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (go.mod not found)")
		}
		dir = parent
	}
}

// writeFile wraps os.WriteFile with dry-run support
func writeFile(path string, data []byte, perm os.FileMode) error {
	if dryRun {
		fmt.Printf("  [DRY-RUN] Would write to: %s\n", path)
		return nil
	}
	return os.WriteFile(path, data, perm)
}

// mkdir wraps os.MkdirAll with dry-run support
func mkdir(path string, perm os.FileMode) error {
	if dryRun {
		fmt.Printf("  [DRY-RUN] Would create directory: %s\n", path)
		return nil
	}
	return os.MkdirAll(path, perm)
}
