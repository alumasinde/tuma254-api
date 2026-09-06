package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type UsersRiders struct{}

func (UsersRiders) Version() int { return 3 }
func (UsersRiders) Name() string { return "users_riders" }

func (UsersRiders) Up(ctx context.Context, db *mongo.Database) error {
	indexes := []struct {
		collection string
		models     []mongo.IndexModel
	}{
		{"profiles", []mongo.IndexModel{
			{Keys: bson.D{{Key: "userId", Value: 1}}, Options: options.Index().SetUnique(true)},
		}},
		{"rider_profiles", []mongo.IndexModel{
			{Keys: bson.D{{Key: "userId", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "verificationStatus", Value: 1}, {Key: "availability", Value: 1}}},
		}},
		{"rider_vehicles", []mongo.IndexModel{
			{Keys: bson.D{{Key: "registrationNumber", Value: 1}}, Options: options.Index().SetUnique(true)},
			{
				Keys: bson.D{{Key: "riderId", Value: 1}},
				Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"active": true}),
			},
			{Keys: bson.D{{Key: "riderId", Value: 1}, {Key: "createdAt", Value: -1}}},
		}},
	}
	for _, entry := range indexes {
		if _, err := db.Collection(entry.collection).Indexes().CreateMany(ctx, entry.models); err != nil {
			return err
		}
	}
	return nil
}
