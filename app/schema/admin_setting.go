package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// AdminSetting holds the schema definition for the AdminSetting entity.
type AdminSetting struct {
	ent.Schema
}

// Fields of the AdminSetting.
func (AdminSetting) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("key").
			Unique().
			NotEmpty().
			Comment("Admin preference key (e.g., 'admin.workspace.sidebar_order')"),
		field.Text("value").
			Comment("Admin preference value (JSON or plain text)"),
	}
}
