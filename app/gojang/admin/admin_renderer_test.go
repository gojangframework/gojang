package admin

import "testing"

func TestNewAdminRendererLoadsEmbeddedTemplates(t *testing.T) {
	renderer, err := NewAdminRenderer(false)
	if err != nil {
		t.Fatalf("NewAdminRenderer(false) returned error: %v", err)
	}

	for _, name := range []string{
		"workspace.html",
		"workspace_overview.partial.html",
		"workspace_grid.partial.html",
		"grid_cell.partial.html",
		"record_drawer.partial.html",
		"admin_main.html",
	} {
		if renderer.templates[name] == nil {
			t.Fatalf("expected embedded admin template %q to be loaded", name)
		}
	}

	workspace := renderer.templates["workspace.html"]
	if workspace.Lookup("workspace_overview.partial.html") == nil {
		t.Fatal("expected workspace.html to include workspace_overview.partial.html")
	}
	if workspace.Lookup("workspace_grid.partial.html") == nil {
		t.Fatal("expected workspace.html to include workspace_grid.partial.html")
	}
	if workspace.Lookup("grid_cell.partial.html") == nil {
		t.Fatal("expected workspace.html to include grid_cell.partial.html")
	}
}
