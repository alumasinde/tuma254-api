ALTER TABLE users
  ADD COLUMN phone_verified_at TIMESTAMPTZ,
  ADD COLUMN verification_required BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE users
SET phone_verified_at = now(),
    verification_required = FALSE
WHERE is_active = TRUE;

CREATE TABLE otp_challenges (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  phone TEXT NOT NULL,
  purpose TEXT NOT NULL,
  code_hash BYTEA NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  verified_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  attempt_count INTEGER NOT NULL DEFAULT 0,
  max_attempts INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT otp_challenges_purpose_check CHECK (purpose IN ('phone_verification')),
  CONSTRAINT otp_challenges_attempts_check CHECK (attempt_count >= 0 AND max_attempts > 0)
);

CREATE INDEX otp_challenges_phone_purpose_created_idx
  ON otp_challenges(phone, purpose, created_at DESC);

CREATE INDEX otp_challenges_user_purpose_active_idx
  ON otp_challenges(user_id, purpose, created_at DESC)
  WHERE verified_at IS NULL AND revoked_at IS NULL;

CREATE INDEX otp_challenges_expires_idx
  ON otp_challenges(expires_at)
  WHERE verified_at IS NULL AND revoked_at IS NULL;