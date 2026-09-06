package users

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrNotFound = errors.New("not found")

type Repository struct{ db *mongo.Database }

func NewRepository(db *mongo.Database) *Repository { return &Repository{db: db} }

func (r *Repository) FindByUserID(ctx context.Context, userID bson.ObjectID) (Profile, error) {
	var profile Profile
	err := r.db.Collection("profiles").FindOne(ctx, bson.M{"userId": userID}).Decode(&profile)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Profile{}, ErrNotFound
	}
	return profile, err
}

func (r *Repository) Upsert(ctx context.Context, userID bson.ObjectID, avatarURL string) (Profile, error) {
	now := time.Now().UTC()
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var profile Profile
	err := r.db.Collection("profiles").FindOneAndUpdate(
		ctx,
		bson.M{"userId": userID},
		bson.M{
			"$set": bson.M{"avatarUrl": avatarURL, "updatedAt": now},
			"$setOnInsert": bson.M{"userId": userID, "createdAt": now},
		},
		opts,
	).Decode(&profile)
	return profile, err
}
