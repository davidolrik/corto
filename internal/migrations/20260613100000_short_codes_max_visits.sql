-- +goose Up
SET statement_timeout = 0;

-- NULL means unlimited
ALTER TABLE short_codes ADD COLUMN max_visits integer;

-- +goose Down
SET statement_timeout = 0;

ALTER TABLE short_codes DROP COLUMN IF EXISTS max_visits;
