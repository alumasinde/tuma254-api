package testkit

import (
	"context"
	"fmt"
	"testing"
	"time"

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

	// Transactions are supported only by replica-set members and mongos. This
	// command gives the test a deterministic prerequisite check before running
	// a concurrent workload that would otherwise produce 32 identical errors.
	var hello bsonM
	if err := client.Database("admin").RunCommand(connectCtx, bsonM{"hello": 1}).Decode(&hello); err != nil {
		t.Fatalf("inspect MongoDB topology: %v", err)
	}
	if hello.String("msg") == "isdbgrid" {
		// mongos is transaction-capable.
	} else if hello.String("setName") == "" {
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

type bsonM map[string]any

func (m bsonM) String(key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

var _ = fmt.Sprintf
