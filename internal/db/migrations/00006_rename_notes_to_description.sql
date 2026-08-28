-- +goose Up
ALTER TABLE tracks RENAME COLUMN notes TO description;

-- +goose Down
ALTER TABLE tracks RENAME COLUMN description TO notes;
