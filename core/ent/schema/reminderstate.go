package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ReminderState records that a reminder fired (or was snoozed) for one
// occurrence of an event. It is keyed by calendar + UID rather than an event
// edge because recurring occurrences are materialized at query time and have
// no row of their own; stale rows are pruned by the reminders engine.
type ReminderState struct {
	ent.Schema
}

func (ReminderState) Fields() []ent.Field {
	return []ent.Field{
		field.String("calendar_id").
			NotEmpty(),
		field.String("uid").
			NotEmpty(),
		field.Time("occurrence_start"),
		field.Int("minutes"),
		field.Time("fired_at").
			Default(time.Now),
		field.Time("snoozed_until").
			Optional().
			Nillable(),
	}
}

func (ReminderState) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("calendar_id", "uid", "occurrence_start", "minutes").Unique(),
		index.Fields("occurrence_start"),
	}
}
