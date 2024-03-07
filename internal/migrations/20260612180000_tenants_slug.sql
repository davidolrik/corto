-- +goose Up
SET statement_timeout = 0;

ALTER TABLE tenants ADD COLUMN slug varchar(64) NOT NULL DEFAULT '';

-- Backfill from the name, then suffix duplicates with their id
UPDATE tenants SET slug = trim(both '-' from regexp_replace(lower(name), '[^a-z0-9]+', '-', 'g'));
UPDATE tenants t SET slug = t.slug || '-' || t.id
WHERE EXISTS (SELECT 1 FROM tenants o WHERE o.slug = t.slug AND o.id < t.id);

CREATE UNIQUE INDEX tenants_slug_key ON tenants (slug);

-- +goose Down
SET statement_timeout = 0;

DROP INDEX IF EXISTS tenants_slug_key;
ALTER TABLE tenants DROP COLUMN IF EXISTS slug;
