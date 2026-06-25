-- +goose Up
-- create "invitation_states" table
CREATE TABLE `invitation_states` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  `calendar_id` text NOT NULL,
  `uid` text NOT NULL,
  `event_start` datetime NOT NULL,
  `notified_at` datetime NOT NULL
);
-- create index "invitationstate_calendar_id_uid" to table: "invitation_states"
CREATE UNIQUE INDEX `invitationstate_calendar_id_uid` ON `invitation_states` (`calendar_id`, `uid`);
-- create index "invitationstate_event_start" to table: "invitation_states"
CREATE INDEX `invitationstate_event_start` ON `invitation_states` (`event_start`);

-- +goose Down
-- reverse: create index "invitationstate_event_start" to table: "invitation_states"
DROP INDEX `invitationstate_event_start`;
-- reverse: create index "invitationstate_calendar_id_uid" to table: "invitation_states"
DROP INDEX `invitationstate_calendar_id_uid`;
-- reverse: create "invitation_states" table
DROP TABLE `invitation_states`;
