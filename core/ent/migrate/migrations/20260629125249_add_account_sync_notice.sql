-- +goose Up
-- add column "sync_notice" to table: "accounts"
ALTER TABLE `accounts` ADD COLUMN `sync_notice` text NULL;

-- +goose Down
-- reverse: add column "sync_notice" to table: "accounts"
ALTER TABLE `accounts` DROP COLUMN `sync_notice`;
