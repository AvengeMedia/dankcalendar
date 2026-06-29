-- +goose Up
-- add column "supported_components" to table: "calendars"
ALTER TABLE `calendars` ADD COLUMN `supported_components` json NULL;
-- create "tasks" table
CREATE TABLE `tasks` (
  `id` text NOT NULL,
  `uid` text NOT NULL,
  `remote_id` text NULL,
  `etag` text NULL,
  `summary` text NOT NULL,
  `description` text NULL,
  `location` text NULL,
  `status` text NOT NULL DEFAULT ('needs_action'),
  `priority` integer NOT NULL DEFAULT (0),
  `percent_complete` integer NOT NULL DEFAULT (0),
  `due` datetime NULL,
  `start` datetime NULL,
  `completed` datetime NULL,
  `all_day` bool NOT NULL DEFAULT (false),
  `due_tz` text NULL,
  `start_tz` text NULL,
  `parent_uid` text NULL,
  `recurrence` json NULL,
  `reminders` json NULL,
  `categories` json NULL,
  `raw_ics` text NULL,
  `created` datetime NOT NULL,
  `updated` datetime NOT NULL,
  `calendar_tasks` text NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `tasks_calendars_tasks` FOREIGN KEY (`calendar_tasks`) REFERENCES `calendars` (`id`) ON DELETE CASCADE
);
-- create index "task_uid_calendar_tasks" to table: "tasks"
CREATE UNIQUE INDEX `task_uid_calendar_tasks` ON `tasks` (`uid`, `calendar_tasks`);
-- create index "task_due" to table: "tasks"
CREATE INDEX `task_due` ON `tasks` (`due`);
-- create index "task_status" to table: "tasks"
CREATE INDEX `task_status` ON `tasks` (`status`);

-- +goose Down
-- reverse: create index "task_status" to table: "tasks"
DROP INDEX `task_status`;
-- reverse: create index "task_due" to table: "tasks"
DROP INDEX `task_due`;
-- reverse: create index "task_uid_calendar_tasks" to table: "tasks"
DROP INDEX `task_uid_calendar_tasks`;
-- reverse: create "tasks" table
DROP TABLE `tasks`;
-- reverse: add column "supported_components" to table: "calendars"
ALTER TABLE `calendars` DROP COLUMN `supported_components`;
