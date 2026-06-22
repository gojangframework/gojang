package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/gojangframework/gojang/app/gojang/models"
	"github.com/google/uuid"
)

func TestNewRegistryDiscoversEntResources(t *testing.T) {
	registry := NewRegistry(models.NewClient())

	for _, name := range []string{"AdminSetting", "Post", "User"} {
		config, err := registry.Get(name)
		if err != nil {
			t.Fatalf("expected resource %s to be discovered: %v", name, err)
		}
		if config.QueryAll == nil || config.CreateFunc == nil || config.UpdateFunc == nil || config.DeleteFunc == nil {
			t.Fatalf("expected resource %s to have CRUD functions", name)
		}
	}
}

func TestAdminSettingIsInternalAndHiddenFromResourceList(t *testing.T) {
	registry := NewRegistry(models.NewClient())

	config, err := registry.Get("AdminSetting")
	if err != nil {
		t.Fatal(err)
	}
	if !config.Internal {
		t.Fatal("expected AdminSetting to be marked internal")
	}
	for _, resource := range registry.List() {
		if resource.Name == "AdminSetting" {
			t.Fatal("expected AdminSetting to be hidden from public admin resources")
		}
	}
}

func TestDiscoveredUserProtectsSensitiveAndSystemFields(t *testing.T) {
	registry := NewRegistry(models.NewClient())
	config, err := registry.Get("User")
	if err != nil {
		t.Fatal(err)
	}

	passwordHash, ok := findField(config, "PasswordHash")
	if !ok {
		t.Fatal("expected PasswordHash field to be discovered")
	}
	if !passwordHash.Hidden || passwordHash.Visible || passwordHash.Editable {
		t.Fatalf("expected PasswordHash to be hidden and non-editable: %+v", passwordHash)
	}

	id, ok := findField(config, "ID")
	if !ok {
		t.Fatal("expected ID field")
	}
	if !id.Readonly || !id.System || id.Editable {
		t.Fatalf("expected ID to be protected: %+v", id)
	}
}

func TestRegisterModelsOverridesDiscoveredResourcesWithoutDuplicates(t *testing.T) {
	registry := NewRegistry(models.NewClient())
	RegisterModels(registry)

	seen := map[string]bool{}
	for _, config := range registry.List() {
		key := config.Name
		if seen[key] {
			t.Fatalf("resource %s registered more than once", key)
		}
		seen[key] = true
	}

	userConfig, err := registry.Get("User")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := findField(userConfig, "Password"); !ok {
		t.Fatal("expected user override to add virtual Password field")
	}
	isActive, ok := findField(userConfig, "IsActive")
	if !ok {
		t.Fatal("expected user IsActive field")
	}
	if isActive.Default != true {
		t.Fatalf("expected User.IsActive create default to come from user override, got %#v", isActive.Default)
	}
}

func TestRegisterModelPreservesDiscoveredRelationFields(t *testing.T) {
	registry := NewRegistry(models.NewClient())
	RegisterModels(registry)

	postConfig, err := registry.Get("Post")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := findField(postConfig, "Author"); !ok {
		t.Fatal("expected post override to preserve discovered Author relation")
	}

	fields := workspaceFields(postConfig)
	found := false
	for _, field := range fields {
		if field.Name == "Author" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected Author relation to render in workspace fields")
	}
}

func TestAdminSettingKeyIsLockedForExistingRecordsButAllowedOnCreate(t *testing.T) {
	registry := NewRegistry(models.NewClient())
	settingConfig, err := registry.Get("AdminSetting")
	if err != nil {
		t.Fatal(err)
	}

	keyField, ok := findField(settingConfig, "Key")
	if !ok {
		t.Fatal("expected AdminSetting.Key field")
	}
	record := &models.AdminSetting{Key: "site.title", Value: "Gojang"}
	if canEditRecordField(settingConfig, record, keyField) {
		t.Fatal("expected existing AdminSetting.Key to be locked")
	}
	if fieldReadonlyForRecord(settingConfig, nil, keyField) {
		t.Fatal("expected AdminSetting.Key to be editable when creating a new setting")
	}
}

func TestAdminSettingValueAndDeleteAreProtected(t *testing.T) {
	config := &ModelConfig{Name: "AdminSetting"}
	record := &models.AdminSetting{Key: "admin.workspace.sidebar_order", Value: `["User"]`}
	handler := NewHandler(nil, nil, nil)

	if err := handler.rejectProtectedRecordMutations(config, record, map[string]interface{}{"Value": "[]"}); err == nil {
		t.Fatal("expected admin setting value update to be rejected")
	}
	if !isProtectedAdminSettingRecord(config, record) {
		t.Fatal("expected admin setting record to be protected")
	}
	if err := handler.rejectProtectedRecordMutations(config, nil, map[string]interface{}{"Key": "admin.resource.User.field_visibility"}); err == nil {
		t.Fatal("expected admin setting key creation to be rejected")
	}
}

func TestRegisterModelSeparatesCreateAndUpdateHooks(t *testing.T) {
	createErr := errors.New("create hook called")
	updateErr := errors.New("update hook called")
	registry := &Registry{models: map[string]*ModelConfig{}}
	err := registry.RegisterModel(ModelRegistration{
		ModelType: &models.Post{},
		BeforeCreate: func(ctx context.Context, data map[string]interface{}) error {
			data["AuthorID"] = uuid.New()
			return createErr
		},
		BeforeUpdate: func(ctx context.Context, data map[string]interface{}) error {
			return updateErr
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	config, err := registry.Get("Post")
	if err != nil {
		t.Fatal(err)
	}

	createData := map[string]interface{}{"Subject": "New"}
	if err := config.UpdateFunc(context.Background(), uuid.New(), createData); !errors.Is(err, updateErr) {
		t.Fatalf("expected update hook error, got %v", err)
	}
	if _, ok := createData["AuthorID"]; ok {
		t.Fatal("create-only hook should not run during updates")
	}

	if _, err := config.CreateFunc(context.Background(), map[string]interface{}{"Subject": "New"}); !errors.Is(err, createErr) {
		t.Fatalf("expected create hook error, got %v", err)
	}
}
