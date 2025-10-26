package admin

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/uuid"
)

// queryAll retrieves all records for a model using Ent client with reflection
func (r *Registry) queryAll(ctx context.Context, modelName string, modifier AfterLoadHook) ([]interface{}, error) {
	// Get the model client using reflection (e.g., r.client.User)
	clientVal := reflect.ValueOf(r.client).Elem()
	modelClient := clientVal.FieldByName(modelName)

	if !modelClient.IsValid() {
		return nil, fmt.Errorf("model %s not found on client", modelName)
	}

	// Call Query() method
	queryMethod := modelClient.MethodByName("Query")
	if !queryMethod.IsValid() {
		return nil, fmt.Errorf("query method not found for model %s", modelName)
	}

	queryResults := queryMethod.Call([]reflect.Value{})
	if len(queryResults) == 0 {
		return nil, fmt.Errorf("query method returned no results for model %s", modelName)
	}

	query := queryResults[0].Interface()

	// Apply modifier if provided
	if modifier != nil {
		query = modifier(ctx, query)
	}

	// Call All(ctx) method
	queryVal := reflect.ValueOf(query)
	allMethod := queryVal.MethodByName("All")
	if !allMethod.IsValid() {
		return nil, fmt.Errorf("all method not found for model %s", modelName)
	}

	allResults := allMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(allResults) != 2 {
		return nil, fmt.Errorf("all method returned unexpected number of values for model %s", modelName)
	}

	// Check for error
	if !allResults[1].IsNil() {
		return nil, allResults[1].Interface().(error)
	}

	// Convert slice to []interface{}
	recordsVal := allResults[0]
	results := make([]interface{}, recordsVal.Len())
	for i := 0; i < recordsVal.Len(); i++ {
		results[i] = recordsVal.Index(i).Interface()
	}

	return results, nil
}

// queryAllPaginated retrieves paginated records for a model using Ent client with reflection
func (r *Registry) queryAllPaginated(ctx context.Context, modelName string, modifier AfterLoadHook, limit, offset int) ([]interface{}, error) {
	// Get the model client using reflection (e.g., r.client.User)
	clientVal := reflect.ValueOf(r.client).Elem()
	modelClient := clientVal.FieldByName(modelName)

	if !modelClient.IsValid() {
		return nil, fmt.Errorf("model %s not found on client", modelName)
	}

	// Call Query() method
	queryMethod := modelClient.MethodByName("Query")
	if !queryMethod.IsValid() {
		return nil, fmt.Errorf("query method not found for model %s", modelName)
	}

	queryResults := queryMethod.Call([]reflect.Value{})
	if len(queryResults) == 0 {
		return nil, fmt.Errorf("query method returned no results for model %s", modelName)
	}

	query := queryResults[0].Interface()

	// Apply modifier if provided
	if modifier != nil {
		query = modifier(ctx, query)
	}

	// Apply Limit and Offset
	queryVal := reflect.ValueOf(query)
	if limit > 0 {
		limitMethod := queryVal.MethodByName("Limit")
		if !limitMethod.IsValid() {
			return nil, fmt.Errorf("limit method not found for model %s", modelName)
		}
		out := limitMethod.Call([]reflect.Value{reflect.ValueOf(limit)})
		if len(out) == 0 {
			return nil, fmt.Errorf("limit method returned no results for model %s", modelName)
		}
		queryVal = out[0]
	}

	if offset > 0 {
		offsetMethod := queryVal.MethodByName("Offset")
		if !offsetMethod.IsValid() {
			return nil, fmt.Errorf("offset method not found for model %s", modelName)
		}
		out := offsetMethod.Call([]reflect.Value{reflect.ValueOf(offset)})
		if len(out) == 0 {
			return nil, fmt.Errorf("offset method returned no results for model %s", modelName)
		}
		queryVal = out[0]
	}

	// Call All(ctx)
	allMethod := queryVal.MethodByName("All")
	if !allMethod.IsValid() {
		return nil, fmt.Errorf("all method not found for model %s", modelName)
	}

	allResults := allMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(allResults) != 2 {
		return nil, fmt.Errorf("all method returned unexpected number of values for model %s", modelName)
	}

	if !allResults[1].IsNil() {
		return nil, allResults[1].Interface().(error)
	}

	// Convert to []interface{}
	recordsVal := allResults[0]
	results := make([]interface{}, recordsVal.Len())
	for i := 0; i < recordsVal.Len(); i++ {
		results[i] = recordsVal.Index(i).Interface()
	}

	return results, nil
}

// countAll returns total number of records for the model
func (r *Registry) countAll(ctx context.Context, modelName string) (int, error) {
	clientVal := reflect.ValueOf(r.client).Elem()
	modelClient := clientVal.FieldByName(modelName)
	if !modelClient.IsValid() {
		return 0, fmt.Errorf("model %s not found on client", modelName)
	}

	queryMethod := modelClient.MethodByName("Query")
	if !queryMethod.IsValid() {
		return 0, fmt.Errorf("query method not found for model %s", modelName)
	}
	queryResults := queryMethod.Call([]reflect.Value{})
	if len(queryResults) == 0 {
		return 0, fmt.Errorf("query method returned no results for model %s", modelName)
	}
	query := queryResults[0].Interface()

	queryVal := reflect.ValueOf(query)
	countMethod := queryVal.MethodByName("Count")
	if !countMethod.IsValid() {
		return 0, fmt.Errorf("count method not found for model %s", modelName)
	}
	countResults := countMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(countResults) != 2 {
		return 0, fmt.Errorf("count method returned unexpected number of values for model %s", modelName)
	}
	if !countResults[1].IsNil() {
		return 0, countResults[1].Interface().(error)
	}
	return int(countResults[0].Int()), nil
}

// queryByID retrieves a single record by ID using reflection
func (r *Registry) queryByID(ctx context.Context, modelName string, id uuid.UUID, _ AfterLoadHook) (interface{}, error) {
	// Get the model client using reflection (e.g., r.client.User)
	clientVal := reflect.ValueOf(r.client).Elem()
	modelClient := clientVal.FieldByName(modelName)

	if !modelClient.IsValid() {
		return nil, fmt.Errorf("model %s not found on client", modelName)
	}

	// Call Get(ctx, id) method
	getMethod := modelClient.MethodByName("Get")
	if !getMethod.IsValid() {
		return nil, fmt.Errorf("Get method not found for model %s", modelName)
	}

	getResults := getMethod.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(id),
	})

	if len(getResults) != 2 {
		return nil, fmt.Errorf("Get method returned unexpected number of values for model %s", modelName)
	}

	// Check for error
	if !getResults[1].IsNil() {
		return nil, getResults[1].Interface().(error)
	}

	return getResults[0].Interface(), nil
}

// genericCreate creates a new record using reflection
func (r *Registry) genericCreate(ctx context.Context, modelName string, data map[string]interface{}) (interface{}, error) {
	// Get the model client using reflection (e.g., r.client.User)
	clientVal := reflect.ValueOf(r.client).Elem()
	modelClient := clientVal.FieldByName(modelName)

	if !modelClient.IsValid() {
		return nil, fmt.Errorf("model %s not found on client", modelName)
	}

	// Call Create() method
	createMethod := modelClient.MethodByName("Create")
	if !createMethod.IsValid() {
		return nil, fmt.Errorf("Create method not found for model %s", modelName)
	}

	createResults := createMethod.Call([]reflect.Value{})
	if len(createResults) == 0 {
		return nil, fmt.Errorf("Create method returned no results for model %s", modelName)
	}

	builder := createResults[0].Interface()

	// Set fields on builder
	if err := setFieldsOnBuilder(builder, data); err != nil {
		return nil, err
	}

	// Call Save(ctx) method
	builderVal := reflect.ValueOf(builder)
	saveMethod := builderVal.MethodByName("Save")
	if !saveMethod.IsValid() {
		return nil, fmt.Errorf("save method not found for model %s", modelName)
	}

	saveResults := saveMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(saveResults) != 2 {
		return nil, fmt.Errorf("save method returned unexpected number of values for model %s", modelName)
	}

	// Check for error
	if !saveResults[1].IsNil() {
		return nil, saveResults[1].Interface().(error)
	}

	return saveResults[0].Interface(), nil
}

// genericUpdate updates a record using reflection
func (r *Registry) genericUpdate(ctx context.Context, modelName string, id uuid.UUID, data map[string]interface{}) error {
	// Get the model client using reflection (e.g., r.client.User)
	clientVal := reflect.ValueOf(r.client).Elem()
	modelClient := clientVal.FieldByName(modelName)

	if !modelClient.IsValid() {
		return fmt.Errorf("model %s not found on client", modelName)
	}

	// Call UpdateOneID(id) method
	updateMethod := modelClient.MethodByName("UpdateOneID")
	if !updateMethod.IsValid() {
		return fmt.Errorf("UpdateOneID method not found for model %s", modelName)
	}

	updateResults := updateMethod.Call([]reflect.Value{reflect.ValueOf(id)})
	if len(updateResults) == 0 {
		return fmt.Errorf("UpdateOneID method returned no results for model %s", modelName)
	}

	builder := updateResults[0].Interface()

	// Set fields on builder
	if err := setFieldsOnBuilder(builder, data); err != nil {
		return err
	}

	// Call Save(ctx) method
	builderVal := reflect.ValueOf(builder)
	saveMethod := builderVal.MethodByName("Save")
	if !saveMethod.IsValid() {
		return fmt.Errorf("save method not found for model %s", modelName)
	}

	saveResults := saveMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(saveResults) != 2 {
		return fmt.Errorf("save method returned unexpected number of values for model %s", modelName)
	}

	// Check for error
	if !saveResults[1].IsNil() {
		return saveResults[1].Interface().(error)
	}

	return nil
}

// genericDelete deletes a record using reflection
func (r *Registry) genericDelete(ctx context.Context, modelName string, id uuid.UUID) error {
	// Get the model client using reflection (e.g., r.client.User)
	clientVal := reflect.ValueOf(r.client).Elem()
	modelClient := clientVal.FieldByName(modelName)

	if !modelClient.IsValid() {
		return fmt.Errorf("model %s not found on client", modelName)
	}

	// Call DeleteOneID(id) method
	deleteMethod := modelClient.MethodByName("DeleteOneID")
	if !deleteMethod.IsValid() {
		return fmt.Errorf("DeleteOneID method not found for model %s", modelName)
	}

	deleteResults := deleteMethod.Call([]reflect.Value{reflect.ValueOf(id)})
	if len(deleteResults) == 0 {
		return fmt.Errorf("DeleteOneID method returned no results for model %s", modelName)
	}

	deleter := deleteResults[0]

	// Call Exec(ctx) method
	execMethod := deleter.MethodByName("Exec")
	if !execMethod.IsValid() {
		return fmt.Errorf("exec method not found for model %s", modelName)
	}

	execResults := execMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(execResults) != 1 {
		return fmt.Errorf("exec method returned unexpected number of values for model %s", modelName)
	}

	// Check for error
	if !execResults[0].IsNil() {
		return execResults[0].Interface().(error)
	}

	return nil
}

// setFieldsOnBuilder sets fields on an Ent builder using reflection
func setFieldsOnBuilder(builder interface{}, data map[string]interface{}) error {
	builderVal := reflect.ValueOf(builder)

	// For each field in data, call the appropriate Set method
	for fieldName, value := range data {
		// Skip nil values and empty strings for non-required fields
		if value == nil || (value == "" && fieldName != "Body") {
			continue
		}

		// Build the setter method name (e.g., "SetEmail" for "Email")
		setterName := "Set" + fieldName
		method := builderVal.MethodByName(setterName)

		if !method.IsValid() {
			// Try with "ID" suffix for foreign keys (e.g., "SetAuthorID")
			setterName = "Set" + fieldName + "ID"
			method = builderVal.MethodByName(setterName)
			if !method.IsValid() {
				continue // Skip fields without setters
			}
		}

		// Call the setter method
		valueToSet := reflect.ValueOf(value)
		method.Call([]reflect.Value{valueToSet})
	}

	return nil
}
