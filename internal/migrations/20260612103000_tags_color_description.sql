-- +goose Up
SET statement_timeout = 0;

ALTER TABLE tags ADD COLUMN color varchar(7) NOT NULL DEFAULT '';
ALTER TABLE tags ADD COLUMN description varchar(512) NOT NULL DEFAULT '';

-- +goose Down
SET statement_timeout = 0;

ALTER TABLE tags DROP COLUMN IF EXISTS description;
ALTER TABLE tags DROP COLUMN IF EXISTS color;
