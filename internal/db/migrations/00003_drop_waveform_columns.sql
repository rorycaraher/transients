-- +goose Up
ALTER TABLE tracks DROP COLUMN peaks_json;
ALTER TABLE tracks DROP COLUMN duration_seconds;

-- +goose Down
ALTER TABLE tracks ADD COLUMN peaks_json TEXT;
ALTER TABLE tracks ADD COLUMN duration_seconds REAL;
