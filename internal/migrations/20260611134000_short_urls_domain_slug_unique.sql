-- +goose Up
SET statement_timeout = 0;

-- Denormalize the slug onto short_urls so uniqueness per domain can be
-- enforced by the database. The service keeps it in sync with short_codes.
ALTER TABLE short_urls ADD COLUMN slug varchar(128);

UPDATE short_urls SET slug = sc.slug FROM short_codes sc WHERE sc.id = short_urls.short_code_id;

ALTER TABLE short_urls ALTER COLUMN slug SET NOT NULL;

CREATE UNIQUE INDEX short_urls_domain_slug_key ON short_urls (domain_id, slug);

-- +goose Down
SET statement_timeout = 0;

DROP INDEX IF EXISTS short_urls_domain_slug_key;

ALTER TABLE short_urls DROP COLUMN IF EXISTS slug;
