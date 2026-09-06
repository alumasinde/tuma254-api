# Tuma254 V1 - Phase 3 Users and Rider Domain

## Boundary

Phase 3 establishes the operational identity of a Tuma254 account as a sender-capable user and, optionally, as a rider. It does not implement dispatch, live tracking, nearby search, delivery lifecycle, package custody, delivery OTPs, payments, or KYC document storage.

The user account remains owned by the Identity domain. Profile data is separated so authentication records do not become a dumping ground for application concerns.

## Collections

- profiles — one application profile per user.
- rider_profiles — one rider lifecycle per user.
- rider_vehicles — rider vehicle history and current active vehicle.

## Rider lifecycle

draft -> submitted -> approved -> suspended

A rejected rider can correct the application and submit again.

Availability is independent from verification:

offline <-> available <-> busy

A rider can only become available when approved and an active vehicle exists. This prevents an unapproved or incomplete rider account from entering future dispatch pools.

## Vehicle model

Vehicle registration numbers are normalized before persistence. A unique database index prevents the same registration from being attached to multiple riders.

A partial unique index allows at most one active vehicle per rider. This is a database invariant, not only an application convention.

## Operations approval

Operations endpoints are protected by the existing operations.users.manage permission. Approval assigns the rider role after the rider profile becomes approved.

The rider profile remains the operational source of truth for whether a rider can become available. An RBAC role alone is not proof of operational readiness.

## Location readiness

Phase 3 intentionally does not store live coordinates inside the rider profile. The next Location domain will own current location, location freshness, and history.

That separation is deliberate because rider profiles change slowly while location updates can be high frequency. The future location collection will use GeoJSON points and a 2dsphere index for nearby-rider queries.

## API

User profile:
- GET /api/v1/profile
- PUT /api/v1/profile

Rider self-service:
- GET /api/v1/rider/profile
- POST /api/v1/rider/application
- POST /api/v1/rider/application/submit
- PUT /api/v1/rider/availability
- GET /api/v1/rider/vehicles
- POST /api/v1/rider/vehicles
- PUT /api/v1/rider/vehicles/{vehicleID}/active

Operations:
- POST /api/v1/operations/riders/{userID}/approve
- POST /api/v1/operations/riders/{userID}/reject
- POST /api/v1/operations/riders/{userID}/suspend

## Tests

Phase 3 adds rider availability transition tests, vehicle validation and normalization tests, and MongoDB integration coverage for:

draft -> vehicle -> submitted -> approved -> available

## Why this supports Tuma254's differentiators

Favourite rider and nearby rider are dispatch features, but both require a trustworthy rider domain.

Future favourite-rider dispatch will ask whether the rider exists, is approved, is available, has a valid active vehicle, and has a fresh location.

Future nearby-rider discovery will query only riders that are operationally eligible and location-fresh.

Phase 3 provides the first half of those guarantees. The Location and Dispatch phases will provide the rest.
