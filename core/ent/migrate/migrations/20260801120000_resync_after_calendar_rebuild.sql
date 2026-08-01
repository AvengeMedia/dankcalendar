-- +goose Up
-- The calendars rebuild in 20260731135440 ran inside a transaction, where its
-- foreign_keys=off pragma was a no-op, so DROP TABLE cascade-deleted every
-- event and task. The copied sync cursors still claim those rows are synced;
-- invalidate them once so the next sync refetches everything.
UPDATE `calendars` SET `sync_token` = NULL WHERE `sync_token` IS NOT NULL;

-- +goose Down
-- Sync cursors are remote, ephemeral state and cannot be reconstructed.
SELECT 1;
