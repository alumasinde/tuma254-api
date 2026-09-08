package models

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	UserID    uuid.UUID
	AvatarURL string
	CreatedAt time.Time
	UpdatedAt time.Time
}
