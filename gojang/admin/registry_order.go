package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/gojangframework/gojang/gojang/models/setting"
)

// LoadModelOrder loads the saved model order from database and applies it
func (r *Registry) LoadModelOrder() {
	ctx := context.Background()

	// Try to load saved order from settings
	settingRecord, err := r.client.Setting.Query().
		Where(setting.KeyEQ("admin_model_order")).
		Only(ctx)

	if err != nil || settingRecord == nil {
		// No saved order, keep registration order
		return
	}

	// Parse JSON array from value
	var savedOrder []string
	value := strings.Trim(settingRecord.Value, "[]\"")
	if value == "" {
		return
	}

	// Simple JSON array parsing
	savedOrder = strings.Split(value, "\",\"")
	for i := range savedOrder {
		savedOrder[i] = strings.Trim(savedOrder[i], "\"")
	}

	// Build new order based on saved preferences
	newKeys := make([]string, 0, len(r.modelKeys))
	usedKeys := make(map[string]bool)

	// First, add models in saved order
	for _, name := range savedOrder {
		key := strings.ToLower(name)
		if _, exists := r.models[key]; exists {
			newKeys = append(newKeys, key)
			usedKeys[key] = true
		}
	}

	// Then add any new models not in saved order
	for _, key := range r.modelKeys {
		if !usedKeys[key] {
			newKeys = append(newKeys, key)
		}
	}

	r.modelKeys = newKeys
}

// SaveModelOrder saves the current model order to database
func (r *Registry) SaveModelOrder(order []string) error {
	ctx := context.Background()

	// Convert order to JSON array string
	var orderNames []string
	for _, key := range order {
		if config, ok := r.models[strings.ToLower(key)]; ok {
			orderNames = append(orderNames, config.Name)
		}
	}

	// Build JSON manually
	jsonValue := "[\"" + strings.Join(orderNames, "\",\"") + "\"]"

	// Check if setting exists
	existing, err := r.client.Setting.Query().
		Where(setting.KeyEQ("admin_model_order")).
		Only(ctx)

	if err != nil {
		// Create new setting
		_, err = r.client.Setting.Create().
			SetKey("admin_model_order").
			SetValue(jsonValue).
			Save(ctx)
	} else {
		// Update existing setting
		err = r.client.Setting.UpdateOne(existing).
			SetValue(jsonValue).
			Exec(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to save model order: %w", err)
	}

	// Update in-memory order
	r.modelKeys = make([]string, len(order))
	for i, name := range order {
		r.modelKeys[i] = strings.ToLower(name)
	}

	return nil
}
