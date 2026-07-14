-- +goose Up
ALTER TABLE tracks ADD COLUMN downloadable BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tracks DROP COLUMN downloadable;
