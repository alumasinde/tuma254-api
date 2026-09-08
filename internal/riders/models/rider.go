package models

import (
	"github.com/google/uuid"
	"time"
)

type Availability string
type VerificationStatus string

const (
	VerificationDraft     VerificationStatus = "draft"
	VerificationSubmitted VerificationStatus = "submitted"
	VerificationApproved  VerificationStatus = "approved"
	VerificationRejected  VerificationStatus = "rejected"
	VerificationSuspended VerificationStatus = "suspended"
	AvailabilityOffline   Availability       = "offline"
	AvailabilityAvailable Availability       = "available"
	AvailabilityBusy      Availability       = "busy"
)

type Profile struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	VerificationStatus VerificationStatus
	Availability       Availability
	RejectionReason    string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Vehicle struct {
	ID                 uuid.UUID
	RiderID            uuid.UUID
	Type               string
	RegistrationNumber string
	Make               string
	Model              string
	Color              string
	Active             bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
