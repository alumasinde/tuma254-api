CREATE TABLE deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_user_id UUID NOT NULL REFERENCES users(id),
    pickup_address TEXT NOT NULL,
    pickup_contact_name TEXT NOT NULL,
    pickup_phone TEXT NOT NULL,
    pickup_longitude DOUBLE PRECISION NOT NULL CHECK (pickup_longitude BETWEEN -180 AND 180),
    pickup_latitude DOUBLE PRECISION NOT NULL CHECK (pickup_latitude BETWEEN -90 AND 90),
    recipient_address TEXT NOT NULL,
    recipient_contact_name TEXT NOT NULL,
    recipient_phone TEXT NOT NULL,
    recipient_longitude DOUBLE PRECISION NOT NULL CHECK (recipient_longitude BETWEEN -180 AND 180),
    recipient_latitude DOUBLE PRECISION NOT NULL CHECK (recipient_latitude BETWEEN -90 AND 90),
    package_description TEXT NOT NULL,
    package_weight_kg NUMERIC(10,2) NOT NULL CHECK (package_weight_kg > 0),
    dispatch_method TEXT NOT NULL CHECK (dispatch_method IN ('favourite_rider','nearby_rider')),
    rider_id UUID REFERENCES rider_profiles(id),
    status TEXT NOT NULL CHECK (status IN (
      'draft','requested','assigned','awaiting_pickup_otp','picked_up',
      'in_transit','awaiting_delivery_otp','completed','failed','cancelled'
    )),
    failure_reason TEXT,
    pickup_otp_hash TEXT,
    pickup_otp_expires_at TIMESTAMPTZ,
    pickup_otp_verified_at TIMESTAMPTZ,
    pickup_otp_attempts INTEGER NOT NULL DEFAULT 0 CHECK (pickup_otp_attempts >= 0),
    pickup_otp_resends INTEGER NOT NULL DEFAULT 0 CHECK (pickup_otp_resends >= 0),
    pickup_otp_locked_at TIMESTAMPTZ,
    delivery_otp_hash TEXT,
    delivery_otp_expires_at TIMESTAMPTZ,
    delivery_otp_verified_at TIMESTAMPTZ,
    delivery_otp_attempts INTEGER NOT NULL DEFAULT 0 CHECK (delivery_otp_attempts >= 0),
    delivery_otp_resends INTEGER NOT NULL DEFAULT 0 CHECK (delivery_otp_resends >= 0),
    delivery_otp_locked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX deliveries_sender_created_idx ON deliveries(sender_user_id, created_at DESC);
CREATE INDEX deliveries_rider_status_idx ON deliveries(rider_id, status) WHERE rider_id IS NOT NULL;
CREATE INDEX deliveries_requested_idx ON deliveries(created_at) WHERE status = 'requested';

CREATE TABLE delivery_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id UUID NOT NULL REFERENCES deliveries(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    actor_user_id UUID REFERENCES users(id),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX delivery_events_delivery_created_idx ON delivery_events(delivery_id, created_at);

CREATE TABLE delivery_custody_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id UUID NOT NULL REFERENCES deliveries(id) ON DELETE CASCADE,
    stage TEXT NOT NULL CHECK (stage IN ('pickup','delivery')),
    actor_user_id UUID NOT NULL REFERENCES users(id),
    evidence_kind TEXT NOT NULL,
    evidence_reference TEXT,
    evidence_note TEXT,
    longitude DOUBLE PRECISION NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    latitude DOUBLE PRECISION NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    captured_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX delivery_custody_events_delivery_idx ON delivery_custody_events(delivery_id, captured_at);

CREATE TABLE delivery_incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id UUID NOT NULL REFERENCES deliveries(id) ON DELETE CASCADE,
    incident_type TEXT NOT NULL,
    reason TEXT NOT NULL,
    actor_user_id UUID NOT NULL REFERENCES users(id),
    delivery_status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE dispatch_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id UUID NOT NULL UNIQUE REFERENCES deliveries(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','completed','exhausted')),
    next_candidate_position INTEGER NOT NULL DEFAULT 0 CHECK (next_candidate_position >= 0),
    active_offer_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE dispatch_candidates (
    plan_id UUID NOT NULL REFERENCES dispatch_plans(id) ON DELETE CASCADE,
    rider_id UUID NOT NULL REFERENCES rider_profiles(id),
    position INTEGER NOT NULL CHECK (position >= 0),
    score DOUBLE PRECISION NOT NULL,
    distance_meters DOUBLE PRECISION NOT NULL CHECK (distance_meters >= 0),
    duration_seconds DOUBLE PRECISION NOT NULL CHECK (duration_seconds >= 0),
    PRIMARY KEY (plan_id, rider_id),
    UNIQUE (plan_id, position)
);

CREATE TABLE assignment_offers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id UUID NOT NULL REFERENCES deliveries(id) ON DELETE CASCADE,
    rider_id UUID NOT NULL REFERENCES rider_profiles(id),
    status TEXT NOT NULL CHECK (status IN ('offered','assigned','declined','expired','cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    responded_at TIMESTAMPTZ
);
CREATE INDEX assignment_offers_delivery_live_idx ON assignment_offers(delivery_id)
WHERE status = 'offered';
CREATE INDEX assignment_offers_expiry_idx ON assignment_offers(expires_at)
WHERE status = 'offered';
