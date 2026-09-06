package testkit

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func MongoDatabase(t *testing.T, uri, name string) *mongo.Database {
	t.Helper()

	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(connectCancel)

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect test mongodb: %v", err)
	}

	if err := client.Ping(connectCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		t.Fatalf("ping test mongodb: %v", err)
	}

	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })

	db := client.Database(name)
	t.Cleanup(func() {
		dropCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = db.Drop(dropCtx)
	})

	return db
}
