package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID
	Email           string
	Phone           string
	FirstName       string
	LastName        string
	Active          bool
	PhoneVerifiedAt *time.Time
	Roles           []string
	CreatedAt       time.Time
}
