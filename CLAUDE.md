# Tuma254 API

This branch is the clean V1 implementation foundation.

Architecture decisions:
- Go backend
- MongoDB as the primary operational database
- Versioned MongoDB migrations
- Domain modules for identity, users, riders, locations, deliveries, dispatch, custody, payments, notifications and operations
- Automated unit, integration, API and end-to-end tests
- Mobile persistent sessions with refresh tokens
- Flutter device biometric and optional local quick-PIN unlock; biometric data is never stored by the backend

Implementation must follow the approved Tuma254 Domain Blueprint and evolve one tested phase at a time.
