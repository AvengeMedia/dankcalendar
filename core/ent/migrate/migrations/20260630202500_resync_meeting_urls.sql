-- +goose Up
-- Meeting URLs are derived from provider data during sync. Existing cursors
-- prevent events stored before the meeting_url field was added from being
-- fetched again, so invalidate them once to rebuild that field.
UPDATE `calendars` SET `sync_token` = NULL WHERE `sync_token` IS NOT NULL;

-- +goose Down
-- Sync cursors are remote, ephemeral state and cannot be reconstructed.
SELECT 1;
