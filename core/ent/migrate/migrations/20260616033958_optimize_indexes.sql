-- +goose Up
-- drop index "event_recurring_id" from table: "events"
DROP INDEX `event_recurring_id`;
-- create index "event_exceptions" to table: "events"
CREATE INDEX `event_exceptions` ON `events` (`recurring_id`) WHERE recurring_id <> '';
-- create index "event_recurring_masters" to table: "events"
CREATE INDEX `event_recurring_masters` ON `events` (`start`) WHERE recurrence IS NOT NULL;
-- drop index "reminderstate_snoozed_until" from table: "reminder_states"
DROP INDEX `reminderstate_snoozed_until`;

-- +goose Down
-- reverse: drop index "reminderstate_snoozed_until" from table: "reminder_states"
CREATE INDEX `reminderstate_snoozed_until` ON `reminder_states` (`snoozed_until`);
-- reverse: create index "event_recurring_masters" to table: "events"
DROP INDEX `event_recurring_masters`;
-- reverse: create index "event_exceptions" to table: "events"
DROP INDEX `event_exceptions`;
-- reverse: drop index "event_recurring_id" from table: "events"
CREATE INDEX `event_recurring_id` ON `events` (`recurring_id`);
