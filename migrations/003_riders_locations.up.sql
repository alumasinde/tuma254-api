CREATE TABLE rider_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    verification_status TEXT NOT NULL DEFAULT 'draft'
        CHECK (verification_status IN ('draft','submitted','approved','rejected','suspended')),
    availability TEXT NOT NULL DEFAULT 'offline'
        CHECK (availability IN ('offline','available','busy')),
    rejection_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rider_vehicles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rider_id UUID NOT NULL REFERENCES rider_profiles(id) ON DELETE CASCADE,
    vehicle_type TEXT NOT NULL,
    registration_number TEXT NOT NULL,
    make TEXT,
    model TEXT,
    color TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (registration_number)
);
CREATE UNIQUE INDEX rider_one_active_vehicle_idx ON rider_vehicles(rider_id) WHERE active;

CREATE TABLE rider_locations (
    rider_id UUID PRIMARY KEY REFERENCES rider_profiles(id) ON DELETE CASCADE,
    longitude DOUBLE PRECISION NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    latitude DOUBLE PRECISION NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    accuracy_meters DOUBLE PRECISION NOT NULL CHECK (accuracy_meters >= 0),
    recorded_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX rider_locations_recorded_at_idx ON rider_locations(recorded_at DESC);
