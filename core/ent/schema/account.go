package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Account struct {
	ent.Schema
}

func (Account) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			MaxLen(64).
			NotEmpty().
			Unique().
			Immutable(),
		field.Enum("kind").
			Values("local", "google", "caldav", "microsoft").
			Immutable(),
		field.String("display_name").
			NotEmpty(),
		field.JSON("settings", map[string]any{}).
			Optional(),
		field.Bool("needs_reauth").
			Default(false),
		field.String("auth_error").
			Optional(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Account) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("calendars", Calendar.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("secrets", Secret.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (Account) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("kind"),
	}
}
