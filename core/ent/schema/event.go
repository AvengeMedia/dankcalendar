package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Event struct {
	ent.Schema
}

func (Event) Fields() []ent.Field {
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
		field.String("url").
			Optional(),
		field.String("meeting_url").
			Optional(),
		field.Enum("status").
			Values("confirmed", "tentative", "cancelled").
			Default("confirmed"),
		field.Time("start"),
		field.Time("end"),
		field.Bool("all_day").
			Default(false),
		field.String("start_tz").
			Optional(),
		field.String("end_tz").
			Optional(),
		field.JSON("recurrence", map[string]any{}).
			Optional(),
		field.String("recurring_id").
			Optional(),
		field.Time("original_start").
			Optional().
			Nillable(),
		field.JSON("organizer", map[string]any{}).
			Optional(),
		field.JSON("attendees", []map[string]any{}).
			Optional(),
		field.JSON("reminders", []map[string]any{}).
			Optional(),
		field.JSON("categories", []string{}).
			Optional(),
		field.String("transparency").
			Optional(),
		field.String("visibility").
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

func (Event) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("calendar", Calendar.Type).
			Ref("events").
			Unique().
			Required(),
	}
}

func (Event) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("calendar").Fields("uid").Unique(),
		index.Fields("start"),
		index.Fields("end"),
		// Partial indexes: exceptions and recurring masters are each a tiny
		// fraction of rows, so index only them rather than the whole table.
		index.Fields("recurring_id").
			Annotations(entsql.IndexWhere("recurring_id <> ''")).
			StorageKey("event_exceptions"),
		index.Fields("start").
			Annotations(entsql.IndexWhere("recurrence IS NOT NULL")).
			StorageKey("event_recurring_masters"),
	}
}
