-- Own schema, not the shared database's public one — see docs/adr/0014.
-- Runs in its own psql session (docker-entrypoint-initdb.d invokes each
-- script separately), so this SET only affects this script.
CREATE SCHEMA IF NOT EXISTS anagrafica;
SET search_path TO anagrafica;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL UNIQUE,
    -- Null until the invite is accepted (see tokens) and the user
    -- sets their own password.
    password_hash TEXT,
    -- System permission to manage accounts, distinct from the organizational
    -- roles below (a "president" isn't necessarily an account administrator).
    is_admin BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Roles are seed data, not a fixed set: they're committee-specific
-- (e.g. "trainer_tssa" means nothing outside CRI) and are upserted at
-- startup from a config file — see docs/adr/0012.
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- Generic single-use token table, not invite-specific: purpose discriminates
-- (today "invite" and "refresh" — see docs/adr/0016 — password-reset is a
-- future phase) instead of duplicating a near-identical table per use case.
CREATE TABLE IF NOT EXISTS tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose TEXT NOT NULL DEFAULT 'invite',
    -- Only the hash is stored: the raw token is only ever in the URL we
    -- hand back to the caller, never persisted.
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
