-- +goose Up
-- disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- create "new_calendars" table
CREATE TABLE `new_calendars` (
  `id` text NOT NULL,
  `remote_id` text NOT NULL,
  `name` text NOT NULL,
  `name_override` text NULL,
  `description` text NULL,
  `color` text NULL,
  `time_zone` text NULL,
  `read_only` bool NOT NULL DEFAULT (false),
  `hidden` bool NOT NULL DEFAULT (false),
  `sync_disabled` bool NOT NULL DEFAULT (false),
  `reminder_overrides` json NULL,
  `sync_token` text NULL,
  `supported_components` json NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `account_calendars` text NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `calendars_accounts_calendars` FOREIGN KEY (`account_calendars`) REFERENCES `accounts` (`id`) ON DELETE CASCADE
);
-- copy rows from old table "calendars" to new temporary table "new_calendars"
INSERT INTO `new_calendars` (`id`, `remote_id`, `name`, `name_override`, `description`, `color`, `time_zone`, `read_only`, `hidden`, `reminder_overrides`, `sync_token`, `supported_components`, `created_at`, `updated_at`, `account_calendars`) SELECT `id`, `remote_id`, `name`, `name_override`, `description`, `color`, `time_zone`, `read_only`, `hidden`, `reminder_overrides`, `sync_token`, `supported_components`, `created_at`, `updated_at`, `account_calendars` FROM `calendars`;
-- drop "calendars" table after copying rows
DROP TABLE `calendars`;
-- rename temporary table "new_calendars" to "calendars"
ALTER TABLE `new_calendars` RENAME TO `calendars`;
-- create index "calendar_remote_id_account_calendars" to table: "calendars"
CREATE UNIQUE INDEX `calendar_remote_id_account_calendars` ON `calendars` (`remote_id`, `account_calendars`);
-- enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;

-- +goose Down
-- reverse: create index "calendar_remote_id_account_calendars" to table: "calendars"
DROP INDEX `calendar_remote_id_account_calendars`;
-- reverse: create "new_calendars" table
DROP TABLE `new_calendars`;
