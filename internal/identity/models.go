package identity

import (
	"time"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID bson.ObjectID `bson:"_id,omitempty"`
	Email string `bson:"email,omitempty"`
	Phone string `bson:"phone,omitempty"`
	FirstName string `bson:"firstName"`
	LastName string `bson:"lastName"`
	PasswordHash string `bson:"passwordHash"`
	Status string `bson:"status"`
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

type PublicUser struct {
	ID string `json:"id"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	Status string `json:"status"`
}

type Session struct {
	ID bson.ObjectID `bson:"_id,omitempty"`
	UserID bson.ObjectID `bson:"userId"`
	FamilyID string `bson:"familyId"`
	TokenHash string `bson:"tokenHash"`
	DeviceID string `bson:"deviceId"`
	DeviceName string `bson:"deviceName,omitempty"`
	CreatedAt time.Time `bson:"createdAt"`
	ExpiresAt time.Time `bson:"expiresAt"`
	RevokedAt *time.Time `bson:"revokedAt,omitempty"`
	RevokedReason string `bson:"revokedReason,omitempty"`
}

type AuthEvent struct {
	ID bson.ObjectID `bson:"_id,omitempty"`
	UserID *bson.ObjectID `bson:"userId,omitempty"`
	Type string `bson:"type"`
	Success bool `bson:"success"`
	IP string `bson:"ip,omitempty"`
	CreatedAt time.Time `bson:"createdAt"`
}
