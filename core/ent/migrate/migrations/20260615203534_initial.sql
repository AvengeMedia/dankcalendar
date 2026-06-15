-- +goose Up
-- create "accounts" table
CREATE TABLE `accounts` (
  `id` text NOT NULL,
  `kind` text NOT NULL,
  `display_name` text NOT NULL,
  `settings` json NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
);
-- create index "account_kind" to table: "accounts"
CREATE INDEX `account_kind` ON `accounts` (`kind`);
-- create "calendars" table
CREATE TABLE `calendars` (
  `id` text NOT NULL,
  `remote_id` text NOT NULL,
  `name` text NOT NULL,
  `name_override` text NULL,
  `description` text NULL,
  `color` text NULL,
  `time_zone` text NULL,
  `read_only` bool NOT NULL DEFAULT (false),
  `hidden` bool NOT NULL DEFAULT (false),
  `sync_token` text NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `account_calendars` text NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `calendars_accounts_calendars` FOREIGN KEY (`account_calendars`) REFERENCES `accounts` (`id`) ON DELETE CASCADE
);
-- create index "calendar_remote_id_account_calendars" to table: "calendars"
CREATE UNIQUE INDEX `calendar_remote_id_account_calendars` ON `calendars` (`remote_id`, `account_calendars`);
-- create "events" table
CREATE TABLE `events` (
  `id` text NOT NULL,
  `uid` text NOT NULL,
  `remote_id` text NULL,
  `etag` text NULL,
  `summary` text NOT NULL,
  `description` text NULL,
  `location` text NULL,
  `url` text NULL,
  `status` text NOT NULL DEFAULT ('confirmed'),
  `start` datetime NOT NULL,
  `end` datetime NOT NULL,
  `all_day` bool NOT NULL DEFAULT (false),
  `start_tz` text NULL,
  `end_tz` text NULL,
  `recurrence` json NULL,
  `recurring_id` text NULL,
  `original_start` datetime NULL,
  `organizer` json NULL,
  `attendees` json NULL,
  `reminders` json NULL,
  `categories` json NULL,
  `transparency` text NULL,
  `visibility` text NULL,
  `raw_ics` text NULL,
  `created` datetime NOT NULL,
  `updated` datetime NOT NULL,
  `calendar_events` text NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `events_calendars_events` FOREIGN KEY (`calendar_events`) REFERENCES `calendars` (`id`) ON DELETE CASCADE
);
-- create index "event_uid_calendar_events" to table: "events"
CREATE UNIQUE INDEX `event_uid_calendar_events` ON `events` (`uid`, `calendar_events`);
-- create index "event_start" to table: "events"
CREATE INDEX `event_start` ON `events` (`start`);
-- create index "event_end" to table: "events"
CREATE INDEX `event_end` ON `events` (`end`);
-- create index "event_recurring_id" to table: "events"
CREATE INDEX `event_recurring_id` ON `events` (`recurring_id`);
-- create "reminder_states" table
CREATE TABLE `reminder_states` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  `calendar_id` text NOT NULL,
  `uid` text NOT NULL,
  `occurrence_start` datetime NOT NULL,
  `minutes` integer NOT NULL,
  `fired_at` datetime NOT NULL,
  `snoozed_until` datetime NULL
);
-- create index "reminderstate_calendar_id_uid_occurrence_start_minutes" to table: "reminder_states"
CREATE UNIQUE INDEX `reminderstate_calendar_id_uid_occurrence_start_minutes` ON `reminder_states` (`calendar_id`, `uid`, `occurrence_start`, `minutes`);
-- create index "reminderstate_occurrence_start" to table: "reminder_states"
CREATE INDEX `reminderstate_occurrence_start` ON `reminder_states` (`occurrence_start`);
-- create index "reminderstate_snoozed_until" to table: "reminder_states"
CREATE INDEX `reminderstate_snoozed_until` ON `reminder_states` (`snoozed_until`);
-- create "secrets" table
CREATE TABLE `secrets` (
  `id` text NOT NULL,
  `key` text NOT NULL,
  `value` blob NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `account_secrets` text NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `secrets_accounts_secrets` FOREIGN KEY (`account_secrets`) REFERENCES `accounts` (`id`) ON DELETE CASCADE
);
-- create index "secret_key_account_secrets" to table: "secrets"
CREATE UNIQUE INDEX `secret_key_account_secrets` ON `secrets` (`key`, `account_secrets`);

-- +goose Down
-- reverse: create index "secret_key_account_secrets" to table: "secrets"
DROP INDEX `secret_key_account_secrets`;
-- reverse: create "secrets" table
DROP TABLE `secrets`;
-- reverse: create index "reminderstate_snoozed_until" to table: "reminder_states"
DROP INDEX `reminderstate_snoozed_until`;
-- reverse: create index "reminderstate_occurrence_start" to table: "reminder_states"
DROP INDEX `reminderstate_occurrence_start`;
-- reverse: create index "reminderstate_calendar_id_uid_occurrence_start_minutes" to table: "reminder_states"
DROP INDEX `reminderstate_calendar_id_uid_occurrence_start_minutes`;
-- reverse: create "reminder_states" table
DROP TABLE `reminder_states`;
-- reverse: create index "event_recurring_id" to table: "events"
DROP INDEX `event_recurring_id`;
-- reverse: create index "event_end" to table: "events"
DROP INDEX `event_end`;
-- reverse: create index "event_start" to table: "events"
DROP INDEX `event_start`;
-- reverse: create index "event_uid_calendar_events" to table: "events"
DROP INDEX `event_uid_calendar_events`;
-- reverse: create "events" table
DROP TABLE `events`;
-- reverse: create index "calendar_remote_id_account_calendars" to table: "calendars"
DROP INDEX `calendar_remote_id_account_calendars`;
-- reverse: create "calendars" table
DROP TABLE `calendars`;
-- reverse: create index "account_kind" to table: "accounts"
DROP INDEX `account_kind`;
-- reverse: create "accounts" table
DROP TABLE `accounts`;
