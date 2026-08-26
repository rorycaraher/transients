-- +goose Up
ALTER TABLE tracks ADD COLUMN notes TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE tracks DROP COLUMN notes;
