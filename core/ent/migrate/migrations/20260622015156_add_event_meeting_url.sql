-- +goose Up
-- add column "meeting_url" to table: "events"
ALTER TABLE `events` ADD COLUMN `meeting_url` text NULL;

-- +goose Down
-- reverse: add column "meeting_url" to table: "events"
ALTER TABLE `events` DROP COLUMN `meeting_url`;
