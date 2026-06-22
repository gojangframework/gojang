package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/gojangframework/gojang/app/gojang/models/setting"
)

// LoadModelOrder loads the saved model order from database and applies it
func (r *Registry) LoadModelOrder() {
	if r.client == nil || r.client.Setting == nil || !r.hasEntDriver() {
		return
	}
	ctx := context.Background()

	settingRecord, err := r.client.Setting.Query().
		Where(setting.KeyEQ("admin.workspace.sidebar_order")).
		Only(ctx)
	if err != nil || settingRecord == nil {
		settingRecord, err = r.client.Setting.Query().
			Where(setting.KeyEQ("admin_model_order")).
			Only(ctx)
	}

	if err != nil || settingRecord == nil {
		// No saved order, keep registration order
		return
	}

	var savedOrder []string
	if err := json.Unmarshal([]byte(settingRecord.Value), &savedOrder); err != nil || len(savedOrder) == 0 {
		return
	}

	r.modelKeys = r.normalizedModelOrder(savedOrder)
}

func (r *Registry) normalizedModelOrder(order []string) []string {
	newKeys := make([]string, 0, len(r.modelKeys))
	usedKeys := make(map[string]bool)

	for _, name := range order {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" || usedKeys[key] {
			continue
		}
		if _, exists := r.models[key]; exists {
			newKeys = append(newKeys, key)
			usedKeys[key] = true
		}
	}

	for _, key := range r.modelKeys {
		if _, exists := r.models[key]; exists && !usedKeys[key] {
			newKeys = append(newKeys, key)
			usedKeys[key] = true
		}
	}

	return newKeys
}

func (r *Registry) hasEntDriver() bool {
	clientValue := reflect.ValueOf(r.client)
	if !clientValue.IsValid() || clientValue.Kind() != reflect.Ptr || clientValue.IsNil() {
		return false
	}
	configField := clientValue.Elem().FieldByName("config")
	if !configField.IsValid() {
		return false
	}
	driverField := configField.FieldByName("driver")
	return driverField.IsValid() && driverField.Kind() == reflect.Interface && !driverField.IsNil()
}

// SaveModelOrder saves the current model order to database
func (r *Registry) SaveModelOrder(order []string) error {
	if r.client == nil || r.client.Setting == nil || !r.hasEntDriver() {
		return fmt.Errorf("settings client is not available")
	}
	ctx := context.Background()

	newKeys := r.normalizedModelOrder(order)
	orderNames := make([]string, 0, len(newKeys))
	for _, key := range newKeys {
		if config, ok := r.models[key]; ok {
			orderNames = append(orderNames, config.Name)
		}
	}

	jsonBytes, err := json.Marshal(orderNames)
	if err != nil {
		return fmt.Errorf("failed to encode model order: %w", err)
	}

	// Check if setting exists
	existing, err := r.client.Setting.Query().
		Where(setting.KeyEQ("admin.workspace.sidebar_order")).
		Only(ctx)

	if err != nil {
		// Create new setting
		_, err = r.client.Setting.Create().
			SetKey("admin.workspace.sidebar_order").
			SetValue(string(jsonBytes)).
			Save(ctx)
	} else {
		// Update existing setting
		err = r.client.Setting.UpdateOne(existing).
			SetValue(string(jsonBytes)).
			Exec(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to save model order: %w", err)
	}

	r.modelKeys = newKeys

	return nil
}
