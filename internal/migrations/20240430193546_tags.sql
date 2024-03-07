-- +goose Up
SET statement_timeout = 0;

CREATE TABLE tags(
    id serial PRIMARY KEY,
    public_id varchar(36),
    tenant_id integer REFERENCES tenants(id),
    slug varchar(64),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE short_code_tags(
    id serial PRIMARY KEY,
    tag_id integer REFERENCES tags(id),
    shortcode_id integer REFERENCES short_codes(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
SET statement_timeout = 0;

DROP TABLE IF EXISTS short_code_tags;

DROP TABLE IF EXISTS tags;
