-- +goose Up
ALTER TABLE users ADD COLUMN is_chirpy_red BOOL NOT NULL DEFAULT False;

-- +goose Down
ALTER TABLE users DROP COLUMN is_chirpy_red;
