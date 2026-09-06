package riders

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	VerificationDraft     = "draft"
	VerificationSubmitted = "submitted"
	VerificationApproved  = "approved"
	VerificationRejected  = "rejected"
	VerificationSuspended = "suspended"

	AvailabilityOffline   = "offline"
	AvailabilityAvailable = "available"
	AvailabilityBusy      = "busy"
)

type Profile struct {
	ID                 bson.ObjectID `bson:"_id,omitempty"`
	UserID             bson.ObjectID `bson:"userId"`
	VerificationStatus string        `bson:"verificationStatus"`
	Availability       string        `bson:"availability"`
	RejectionReason    string        `bson:"rejectionReason,omitempty"`
	CreatedAt          time.Time     `bson:"createdAt"`
	UpdatedAt          time.Time     `bson:"updatedAt"`
}

type Vehicle struct {
	ID                 bson.ObjectID `bson:"_id,omitempty"`
	RiderID            bson.ObjectID `bson:"riderId"`
	Type               string        `bson:"type"`
	RegistrationNumber string        `bson:"registrationNumber"`
	Make               string        `bson:"make,omitempty"`
	Model              string        `bson:"model,omitempty"`
	Color              string        `bson:"color,omitempty"`
	Active             bool          `bson:"active"`
	CreatedAt          time.Time     `bson:"createdAt"`
	UpdatedAt          time.Time     `bson:"updatedAt"`
}

type PublicProfile struct {
	UserID             string    `json:"user_id"`
	VerificationStatus string    `json:"verification_status"`
	Availability       string    `json:"availability"`
	RejectionReason    string    `json:"rejection_reason,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type PublicVehicle struct {
	ID                 string    `json:"id"`
	Type               string    `json:"type"`
	RegistrationNumber string    `json:"registration_number"`
	Make               string    `json:"make,omitempty"`
	Model              string    `json:"model,omitempty"`
	Color              string    `json:"color,omitempty"`
	Active             bool      `json:"active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
