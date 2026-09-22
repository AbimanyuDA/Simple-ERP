CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL,
    display_name text NOT NULL,
    password_hash text,
    role text NOT NULL CHECK (role IN ('ADMIN', 'OPERATOR')),
    status text NOT NULL CHECK (status IN ('INVITED', 'ACTIVE', 'DEACTIVATED')),
    invited_by uuid REFERENCES users(id),
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_normalized CHECK (email = lower(btrim(email)))
);
CREATE UNIQUE INDEX users_email_unique ON users (lower(email));

CREATE TABLE sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id),
    family_id uuid NOT NULL,
    access_hash text NOT NULL UNIQUE,
    refresh_hash text NOT NULL UNIQUE,
    csrf_hash text NOT NULL,
    access_expires_at timestamptz NOT NULL,
    refresh_idle_expires_at timestamptz NOT NULL,
    refresh_absolute_expires_at timestamptz NOT NULL,
    rotated_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_active_idx ON sessions (user_id) WHERE revoked_at IS NULL;
CREATE INDEX sessions_family_idx ON sessions (family_id);

CREATE TABLE action_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id),
    kind text NOT NULL CHECK (kind IN ('INVITATION', 'PASSWORD_RESET')),
    token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    revoked_at timestamptz,
    created_by uuid REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE audit_entries (
    id bigserial PRIMARY KEY,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    actor_id uuid REFERENCES users(id),
    action text NOT NULL,
    trace_id text NOT NULL,
    entity_type text NOT NULL,
    entity_id text NOT NULL,
    previous_state text,
    new_state text,
    reason text,
    request_id text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX audit_entity_idx ON audit_entries (entity_type, entity_id, occurred_at DESC);
CREATE INDEX audit_actor_idx ON audit_entries (actor_id, occurred_at DESC);

CREATE TABLE idempotency_keys (
    actor_id uuid NOT NULL REFERENCES users(id),
    key text NOT NULL,
    request_hash text NOT NULL,
    response_status integer,
    response_body jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (actor_id, key)
);

CREATE SEQUENCE document_number_sequence;
