-- +goose Up
SET statement_timeout = 0;

CREATE TABLE users(
    id serial PRIMARY KEY,
    public_id varchar(36) UNIQUE NOT NULL,
    username varchar(128),
    password varchar(128),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tenants(
    id serial PRIMARY KEY,
    public_id varchar(36) UNIQUE NOT NULL,
    owner_id integer REFERENCES users(id),
    name varchar(128),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tenant_user_access(
    id serial PRIMARY KEY,
    tenant_id integer REFERENCES tenants(id),
    user_id integer REFERENCES users(id),
    is_admin boolean DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
SET statement_timeout = 0;

DROP TABLE IF EXISTS tenant_user_access;

DROP TABLE IF EXISTS tenants;

DROP TABLE IF EXISTS users;
