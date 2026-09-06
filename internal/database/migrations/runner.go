package migrations

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const collectionName = "schema_migrations"

type Applied struct {
	Version   int       `bson:"version"`
	Name      string    `bson:"name"`
	AppliedAt time.Time `bson:"appliedAt"`
}

func Run(ctx context.Context, db *mongo.Database, migrations ...Migration) error {
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version() < migrations[j].Version() })

	for _, migration := range migrations {
		var applied Applied
		err := db.Collection(collectionName).FindOne(ctx, bson.M{"version": migration.Version()}).Decode(&applied)
		if err == nil {
			continue
		}
		if err != mongo.ErrNoDocuments {
			return fmt.Errorf("check migration %d: %w", migration.Version(), err)
		}

		if err := migration.Up(ctx, db); err != nil {
			return fmt.Errorf("migration %d_%s: %w", migration.Version(), migration.Name(), err)
		}

		if _, err := db.Collection(collectionName).InsertOne(ctx, Applied{
			Version: migration.Version(),
			Name: migration.Name(),
			AppliedAt: time.Now().UTC(),
		}); err != nil {
			return fmt.Errorf("record migration %d: %w", migration.Version(), err)
		}
	}
	return nil
}
