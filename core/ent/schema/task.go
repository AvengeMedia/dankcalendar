package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Task struct {
	ent.Schema
}

func (Task) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			MaxLen(64).
			NotEmpty().
			Unique().
			Immutable(),
		field.String("uid").
			NotEmpty(),
		field.String("remote_id").
			Optional(),
		field.String("etag").
			Optional(),
		field.String("summary"),
		field.Text("description").
			Optional(),
		field.String("location").
			Optional(),
		field.Enum("status").
			Values("needs_action", "in_process", "completed", "cancelled").
			Default("needs_action"),
		field.Int("priority").
			Default(0),
		field.Int("percent_complete").
			Default(0),
		field.Time("due").
			Optional().
			Nillable(),
		field.Time("start").
			Optional().
			Nillable(),
		field.Time("completed").
			Optional().
			Nillable(),
		field.Bool("all_day").
			Default(false),
		field.String("due_tz").
			Optional(),
		field.String("start_tz").
			Optional(),
		field.String("parent_uid").
			Optional(),
		field.JSON("recurrence", map[string]any{}).
			Optional(),
		field.JSON("reminders", []map[string]any{}).
			Optional(),
		field.JSON("categories", []string{}).
			Optional(),
		field.Text("raw_ics").
			Optional(),
		field.Time("created").
			Default(time.Now),
		field.Time("updated").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Task) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("calendar", Calendar.Type).
			Ref("tasks").
			Unique().
			Required(),
	}
}

func (Task) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("calendar").Fields("uid").Unique(),
		index.Fields("due"),
		index.Fields("status"),
	}
}
