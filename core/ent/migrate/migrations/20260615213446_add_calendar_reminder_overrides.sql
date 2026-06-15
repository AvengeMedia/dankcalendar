-- +goose Up
-- add column "reminder_overrides" to table: "calendars"
ALTER TABLE `calendars` ADD COLUMN `reminder_overrides` json NULL;

-- +goose Down
-- reverse: add column "reminder_overrides" to table: "calendars"
ALTER TABLE `calendars` DROP COLUMN `reminder_overrides`;
