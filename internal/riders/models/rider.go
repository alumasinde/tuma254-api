package models

import "time"

type Availability string
const (
	AvailabilityOffline Availability = "offline"
	AvailabilityAvailable Availability = "available"
	AvailabilityBusy Availability = "busy"
)

type Profile struct {
	ID                 string
	UserID             string
	VerificationStatus string
	Availability       Availability
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
