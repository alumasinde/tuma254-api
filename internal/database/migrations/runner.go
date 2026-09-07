package migrations

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	collectionName       = "schema_migrations"
	migrationTimeout     = 30 * time.Second
)

type Applied struct {
	Version   int       `bson:"version"`
	Name      string    `bson:"name"`
	AppliedAt time.Time `bson:"appliedAt"`
}

func Run(ctx context.Context, db *mongo.Database, migrations ...Migration) error {
	if ctx == nil {
		ctx = context.Background()
	}

	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version() < migrations[j].Version() })

	// Protect the migration history from duplicate records when tests or
	// application instances are started concurrently.
	indexCtx, cancel := context.WithTimeout(ctx, migrationTimeout)
	defer cancel()
	if _, err := db.Collection(collectionName).Indexes().CreateOne(indexCtx, mongo.IndexModel{
		Keys:    bson.D{{Key: "version", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return fmt.Errorf("create migration history index: %w", err)
	}

	for _, migration := range migrations {
		checkCtx, cancel := context.WithTimeout(ctx, migrationTimeout)
		var applied Applied
		err := db.Collection(collectionName).FindOne(checkCtx, bson.M{"version": migration.Version()}).Decode(&applied)
		cancel()

		if err == nil {
			continue
		}
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("check migration %d: %w", migration.Version(), err)
		}

		// A migration must never be allowed to hang the entire process.
		// The migration itself receives the bounded context so all of its
		// collection/index operations inherit the same deadline.
		migrationCtx, cancel := context.WithTimeout(ctx, migrationTimeout)
		err = migration.Up(migrationCtx, db)
		cancel()
		if err != nil {
			return fmt.Errorf("migration %d_%s: %w", migration.Version(), migration.Name(), err)
		}

		recordCtx, cancel := context.WithTimeout(ctx, migrationTimeout)
		_, err = db.Collection(collectionName).InsertOne(recordCtx, Applied{
			Version:   migration.Version(),
			Name:      migration.Name(),
			AppliedAt: time.Now().UTC(),
		})
		cancel()
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				continue
			}
			return fmt.Errorf("record migration %d: %w", migration.Version(), err)
		}
	}
	return nil
}
