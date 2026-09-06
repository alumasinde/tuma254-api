# Tuma254 V1 - Phase 2 Identity, Authentication and Security

## Boundary

This phase owns users, authentication, authorization and sessions. Delivery OTPs, rider verification, KYC, payments and package custody are intentionally separate domain concerns.

## Authentication design

Password login produces a short-lived JWT access token and an opaque refresh token.

The access token is short-lived. The refresh token is random, is never stored in plaintext, and is persisted only as a SHA-256 hash.

Every refresh token belongs to a session family. Refresh rotates the token by revoking the old session record and creating a new record in the same family. Reuse of an already-revoked refresh token is treated as a compromise and revokes the active family.

## Collections

users
roles
permissions
user_roles
role_permissions
sessions
auth_events

Migration 002 creates indexes and seeds baseline roles and permissions.

## Mobile PIN and biometrics

The optional quick PIN belongs to the mobile device. Biometric templates and the local PIN are never sent to MongoDB.

Flutter secure storage holds the backend session material. Android/iOS biometric or local PIN protection unlocks that secure material locally. The backend remains responsible for access-token expiry, refresh-token rotation and server-side session revocation.

## API

POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
GET /api/v1/auth/me

Protected requests use Authorization: Bearer access-token.

## Authorization

Roles and permissions are stored separately from users. Permission middleware is available for future domain routes. This avoids putting long-lived authorization decisions into JWT claims.

## Rate limiting

Authentication endpoints use an in-process fixed-window limiter in this phase. It is suitable for one API process. Before horizontal scaling, rate limiting must move to a shared store or API gateway so all instances enforce the same limit.

## Security properties delivered

- bcrypt password hashing
- minimum password policy
- signed short-lived access tokens
- opaque refresh tokens
- hashed refresh-token storage
- refresh-token rotation
- refresh-token reuse detection
- session-family revocation
- server-side access-session validation
- logout revocation
- authentication audit events
- baseline RBAC
- authentication rate limiting
- unit and integration test foundations
