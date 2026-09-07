package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Client struct {
	client *mongo.Client
	db     *mongo.Database
}

func Connect(ctx context.Context, uri, database string) (*Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect mongodb: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}

	return &Client{client: client, db: client.Database(database)}, nil
}

func (c *Client) Database() *mongo.Database { return c.db }

func (c *Client) EnsureTransactionTopology(ctx context.Context) error {
	var hello struct {
		SetName string `bson:"setName"`
		Msg     string `bson:"msg"`
	}
	if err := c.db.Client().Database("admin").RunCommand(ctx, map[string]int{"hello": 1}).Decode(&hello); err != nil {
		return fmt.Errorf("inspect mongodb topology: %w", err)
	}
	if hello.Msg == "isdbgrid" || hello.SetName != "" {
		return nil
	}
	return fmt.Errorf("mongodb transactions require a replica set member or mongos")
}

func (c *Client) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}
