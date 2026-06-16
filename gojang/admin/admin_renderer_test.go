package admin

import "testing"

func TestNewAdminRendererLoadsEmbeddedTemplates(t *testing.T) {
	renderer, err := NewAdminRenderer(false)
	if err != nil {
		t.Fatalf("NewAdminRenderer(false) returned error: %v", err)
	}

	for _, name := range []string{
		"model_index.html",
		"model_list.partial.html",
	} {
		if renderer.templates[name] == nil {
			t.Fatalf("expected embedded admin template %q to be loaded", name)
		}
	}

	modelIndex := renderer.templates["model_index.html"]
	if modelIndex.Lookup("model_list.partial.html") == nil {
		t.Fatal("expected model_index.html to include model_list.partial.html")
	}
}
