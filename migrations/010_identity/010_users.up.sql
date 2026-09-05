CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE users (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 first_name TEXT NOT NULL,
 last_name TEXT NOT NULL,
 email TEXT UNIQUE,
 phone TEXT UNIQUE,
 password_hash TEXT NOT NULL,
 email_verified_at TIMESTAMPTZ,
 phone_verified_at TIMESTAMPTZ,
 status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended','disabled')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK (email IS NOT NULL OR phone IS NOT NULL)
);
CREATE UNIQUE INDEX users_email_normalized_unique ON users ((lower(email))) WHERE email IS NOT NULL;
