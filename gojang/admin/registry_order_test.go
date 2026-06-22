package admin

import (
	"strings"
	"testing"

	"github.com/gojangframework/gojang/gojang/models"
)

func TestSaveModelOrderWithoutDriverReturnsError(t *testing.T) {
	registry := NewRegistry(models.NewClient())

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("SaveModelOrder should return an error instead of panicking, got panic %v", recovered)
		}
	}()

	err := registry.SaveModelOrder([]string{"User"})
	if err == nil {
		t.Fatal("expected SaveModelOrder to reject a client without a database driver")
	}
	if !strings.Contains(err.Error(), "settings client is not available") {
		t.Fatalf("expected settings client error, got %v", err)
	}
}

func TestNormalizedModelOrderKeepsMissingResourcesVisible(t *testing.T) {
	registry := &Registry{
		models: map[string]*ModelConfig{
			"user":    {Name: "User"},
			"post":    {Name: "Post"},
			"setting": {Name: "Setting"},
		},
		modelKeys: []string{"user", "post", "setting"},
	}

	got := registry.normalizedModelOrder([]string{"Setting", "missing", "User", "Setting"})
	want := []string{"setting", "user", "post"}

	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
