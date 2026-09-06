package users

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Profile struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	UserID    bson.ObjectID `bson:"userId"`
	AvatarURL string        `bson:"avatarUrl,omitempty"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

type PublicProfile struct {
	UserID    string    `json:"user_id"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
