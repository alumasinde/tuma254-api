# Tuma254 V1 - Phase 4 Location and Routing Engine

## Objective

Phase 4 establishes reusable geospatial and road-routing infrastructure without coupling delivery business rules directly to MongoDB geospatial queries or Valhalla.

## Domain boundaries

Locations owns GeoJSON validation, saved places, current rider locations, freshness, and geographic candidate lookup.

Routing owns route requests, distance, duration, route geometry, matrix calculations, and Valhalla transport.

Riders owns whether a rider is operationally eligible to publish live location.

A rider may publish location only when approved and operationally active.

## Collections

### rider_locations

Exactly one current location per rider.

The unique riderId index makes updates atomic upserts rather than append-only writes.

Indexes are riderId unique, location 2dsphere, and updatedAt descending.

### saved_places

User-owned reusable addresses.

Indexes are userId plus updatedAt, userId plus label unique, and location 2dsphere.

## Concurrency model

Location writes are independent per rider and use MongoDB atomic upsert operations. There is no application-level global lock.

Hot path:

Rider device -> validation -> eligibility check -> atomic upsert

The current location collection remains bounded because each rider owns one document.

Nearby lookup is bounded by MongoDB geospatial indexing, caller radius, maximum 100 results, and freshness.

## Freshness

LOCATION_FRESHNESS_SECONDS is configuration.

Stale location is query-time state rather than a stored boolean, avoiding background writes across all rider documents.

## Valhalla

The application depends on the Router interface. Valhalla is an adapter.

Supported capabilities are route and source-to-target matrix.

Calls use request contexts and bounded HTTP timeouts.

GPS updates never call Valhalla.

## APIs

PUT /api/v1/rider/location

GET, POST and DELETE /api/v1/places

POST /api/v1/routing/route

POST /api/v1/routing/matrix

Future delivery and dispatch modules will own their business-specific routing contracts.

## Future dispatch

MongoDB identifies geographic candidates.

Dispatch applies operational eligibility and delivery constraints.

Valhalla matrix ranks eligible candidates by road travel time.

Geographic proximity is therefore not treated as dispatch suitability.

## Tests

Phase 4 includes coordinate validation and Valhalla response mapping tests. MongoDB integration coverage should verify 2dsphere indexing and stale-location exclusion before dispatch begins.
