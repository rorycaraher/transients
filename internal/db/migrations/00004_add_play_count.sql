-- +goose Up
ALTER TABLE tracks ADD COLUMN play_count INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tracks DROP COLUMN play_count;
