package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Secret struct {
	ent.Schema
}

func (Secret) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			MaxLen(64).
			NotEmpty().
			Unique().
			Immutable(),
		field.String("key").
			NotEmpty(),
		field.Bytes("value").
			Sensitive(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Secret) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", Account.Type).
			Ref("secrets").
			Unique().
			Required(),
	}
}

func (Secret) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("account").Fields("key").Unique(),
	}
}
