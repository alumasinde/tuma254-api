CREATE INDEX IF NOT EXISTS rider_profiles_status_availability_idx
    ON rider_profiles(verification_status, availability);
CREATE INDEX IF NOT EXISTS rider_vehicles_rider_created_idx
    ON rider_vehicles(rider_id, created_at DESC);
