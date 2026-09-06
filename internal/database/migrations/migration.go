package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Migration interface {
	Version() int
	Name() string
	Up(context.Context, *mongo.Database) error
}
