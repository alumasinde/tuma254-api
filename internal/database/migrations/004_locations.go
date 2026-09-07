package migrations

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Locations struct{}

func (Locations) Version() int { return 4 }
func (Locations) Name() string { return "locations" }

func (Locations) Up(ctx context.Context, db *mongo.Database) error {
	entries := []struct {
		collection string
		models     []mongo.IndexModel
	}{
		{"saved_places", []mongo.IndexModel{
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "updatedAt", Value: -1}}},
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "label", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "location", Value: "2dsphere"}}},
		}},
		{"rider_locations", []mongo.IndexModel{
			{Keys: bson.D{{Key: "riderId", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "location", Value: "2dsphere"}}},
			{Keys: bson.D{{Key: "updatedAt", Value: -1}}},
		}},
	}

	for _, entry := range entries {
		// Create indexes one at a time. This gives deterministic failure
		// reporting and avoids one stalled CreateMany call hiding which
		// collection/index is responsible.
		for _, model := range entry.models {
			if _, err := db.Collection(entry.collection).Indexes().CreateOne(ctx, model); err != nil {
				return fmt.Errorf("create index on %s: %w", entry.collection, err)
			}
		}
	}
	return nil
}
