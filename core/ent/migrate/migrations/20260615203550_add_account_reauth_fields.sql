-- +goose Up
-- disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- create "new_accounts" table
CREATE TABLE `new_accounts` (
  `id` text NOT NULL,
  `kind` text NOT NULL,
  `display_name` text NOT NULL,
  `settings` json NULL,
  `needs_reauth` bool NOT NULL DEFAULT (false),
  `auth_error` text NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
);
-- copy rows from old table "accounts" to new temporary table "new_accounts"
INSERT INTO `new_accounts` (`id`, `kind`, `display_name`, `settings`, `created_at`, `updated_at`) SELECT `id`, `kind`, `display_name`, `settings`, `created_at`, `updated_at` FROM `accounts`;
-- drop "accounts" table after copying rows
DROP TABLE `accounts`;
-- rename temporary table "new_accounts" to "accounts"
ALTER TABLE `new_accounts` RENAME TO `accounts`;
-- create index "account_kind" to table: "accounts"
CREATE INDEX `account_kind` ON `accounts` (`kind`);
-- enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;

-- +goose Down
-- reverse: create index "account_kind" to table: "accounts"
DROP INDEX `account_kind`;
-- reverse: create "new_accounts" table
DROP TABLE `new_accounts`;
