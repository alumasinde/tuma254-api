package models

import (
	"time"

	"github.com/google/uuid"
)

const OTPPurposePhoneVerification = "phone_verification"

type OTPChallenge struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Phone        string
	Purpose      string
	ExpiresAt    time.Time
	AttemptCount int
	MaxAttempts  int
	CreatedAt    time.Time
}

type OTPVerifyResult struct {
	User      User
	Verified  bool
	Expired   bool
	Exhausted bool
}