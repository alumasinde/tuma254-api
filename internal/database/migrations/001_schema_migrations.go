package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type SchemaMigrations struct{}

func (SchemaMigrations) Version() int { return 1 }
func (SchemaMigrations) Name() string { return "schema_migrations" }

func (SchemaMigrations) Up(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection(collectionName).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "version", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}
