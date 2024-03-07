-- +goose Up
SET statement_timeout = 0;

ALTER TABLE domains ADD COLUMN description varchar(512) NOT NULL DEFAULT '';

-- +goose Down
SET statement_timeout = 0;

ALTER TABLE domains DROP COLUMN IF EXISTS description;
