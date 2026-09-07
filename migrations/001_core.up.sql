-- Migration 001 is intentionally limited to database prerequisites.
--
-- Identity tables are defined once in 002_identity.up.sql. The previous version of
-- this migration duplicated users, roles, user_roles and refresh_sessions, which
-- made a clean migration fail at 002 with "relation already exists".

CREATE EXTENSION IF NOT EXISTS pgcrypto;
