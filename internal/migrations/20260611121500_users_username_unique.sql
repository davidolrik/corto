-- +goose Up
SET statement_timeout = 0;

CREATE UNIQUE INDEX users_username_key ON users (username);

-- +goose Down
SET statement_timeout = 0;

DROP INDEX IF EXISTS users_username_key;
