-- +goose Up
SET statement_timeout = 0;

CREATE TABLE domains(
    id serial PRIMARY KEY,
    public_id varchar(36),
    tenant_id integer REFERENCES tenants(id),
    fqdn varchar(128) UNIQUE NOT NULL,
    fallback_url varchar(8000),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE short_codes(
    id serial PRIMARY KEY,
    public_id varchar(36),
    title varchar(512),
    slug varchar(128),
    target_url varchar(8000),
    fallback_url varchar(8000), -- When url is not valid
    is_crawlable boolean DEFAULT FALSE, -- Include in robots.txt
    forward_query boolean DEFAULT FALSE, -- Forward query string to target
    valid_since timestamptz DEFAULT NULL, -- Time period where url is valid
    valid_until timestamptz DEFAULT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE table platform_specific_urls(
    id serial PRIMARY KEY,
    public_id varchar(36),
    short_code_id integer REFERENCES short_codes(id),
    target_url varchar(8000),
    fallback_url varchar(8000), -- When url is not valid
    platform varchar(32), -- Mobile, iOS, Android, Desktop, macOS, Windows, Linux
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE short_urls(
    id serial PRIMARY KEY,
    public_id varchar(36),
    domain_id integer REFERENCES domains(id),
    short_code_id integer REFERENCES short_codes(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE visits(
    id serial PRIMARY KEY,
    public_id varchar(36),
    short_url_id integer REFERENCES short_urls(id),
    ip_address varchar(64),
    user_agent varchar(512),
    refere varchar(8000),
    country varchar(128),
    campaign varchar(128),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
SET statement_timeout = 0;

DROP TABLE IF EXISTS visits;

DROP TABLE IF EXISTS short_urls;

DROP TABLE IF EXISTS platform_specific_urls;

DROP TABLE IF EXISTS short_codes;

DROP TABLE IF EXISTS domains;
