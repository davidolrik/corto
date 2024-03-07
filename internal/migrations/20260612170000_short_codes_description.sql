-- +goose Up
SET statement_timeout = 0;

ALTER TABLE short_codes ADD COLUMN description varchar(512) NOT NULL DEFAULT '';

-- +goose Down
SET statement_timeout = 0;

ALTER TABLE short_codes DROP COLUMN IF EXISTS description;
