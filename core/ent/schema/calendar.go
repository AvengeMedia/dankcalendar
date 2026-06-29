package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/AvengeMedia/dankcalendar/core/internal/settings"
)

type Calendar struct {
	ent.Schema
}

func (Calendar) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			MaxLen(64).
			NotEmpty().
			Unique().
			Immutable(),
		field.String("remote_id").
			NotEmpty(),
		field.String("name").
			NotEmpty(),
		// User-set display name; owned locally like hidden, never
		// touched by provider sync.
		field.String("name_override").
			Optional(),
		field.String("description").
			Optional(),
		field.String("color").
			Optional(),
		field.String("time_zone").
			Optional(),
		field.Bool("read_only").
			Default(false),
		field.Bool("hidden").
			Default(false),
		// Per-calendar reminder overrides; owned locally like hidden,
		// nil means the calendar follows the global reminder settings.
		field.JSON("reminder_overrides", &settings.ReminderOverride{}).
			Optional(),
		field.String("sync_token").
			Optional(),
		// iCalendar component types the collection holds (VEVENT, VTODO).
		// Empty means an event calendar, for back-compat with rows that
		// predate task support.
		field.JSON("supported_components", []string{}).
			Optional(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Calendar) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", Account.Type).
			Ref("calendars").
			Unique().
			Required(),
		edge.To("events", Event.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("tasks", Task.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (Calendar) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("account").Fields("remote_id").Unique(),
	}
}
