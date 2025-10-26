package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Setting holds the schema definition for the Setting entity.
type Setting struct {
	ent.Schema
}

// Fields of the Setting.
func (Setting) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("key").
			Unique().
			NotEmpty().
			Comment("Setting key (e.g., 'admin_model_order')"),
		field.Text("value").
			Comment("Setting value (JSON or plain text)"),
	}
}
