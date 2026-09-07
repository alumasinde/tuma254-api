package testkit

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoTransactionDatabase connects to a transaction-capable MongoDB endpoint.
// A standalone mongod cannot execute multi-document transactions; these tests
// therefore fail clearly rather than silently downgrading the consistency model.
func MongoTransactionDatabase(t *testing.T, uri, name string) *mongo.Database {
	t.Helper()

	connectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect transactional test mongodb: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })

	if err := client.Ping(connectCtx, nil); err != nil {
		t.Fatalf("ping transactional test mongodb: %v", err)
	}

	var hello bson.M
	if err := client.Database("admin").RunCommand(connectCtx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err != nil {
		t.Fatalf("inspect MongoDB topology: %v", err)
	}
	if msg, _ := hello["msg"].(string); msg == "isdbgrid" {
		// mongos is transaction-capable.
	} else if _, ok := hello["setName"].(string); !ok {
		t.Fatalf("transactional custody tests require MongoDB replica set or mongos; connected endpoint is standalone")
	}

	db := client.Database(name)
	t.Cleanup(func() {
		dropCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = db.Drop(dropCtx)
	})
	return db
}

type _ = mongo.Database
