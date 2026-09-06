package mongodb

import (
	"context"
	"time"
)

func (c *Client) Health(ctx context.Context) error {
	healthCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return c.client.Ping(healthCtx, nil)
}
