package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// InvitationState records that the user was notified about an unanswered
// meeting invitation, so the engine prompts once per invite rather than on
// every sync. Like ReminderState it is keyed by calendar + UID instead of an
// event edge, and stale rows are pruned once the event is well in the past.
type InvitationState struct {
	ent.Schema
}

func (InvitationState) Fields() []ent.Field {
	return []ent.Field{
		field.String("calendar_id").
			NotEmpty(),
		field.String("uid").
			NotEmpty(),
		field.Time("event_start"),
		field.Time("notified_at").
			Default(time.Now),
	}
}

func (InvitationState) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("calendar_id", "uid").Unique(),
		index.Fields("event_start"),
	}
}
