package repositories

import (
	"context"
	"time"

	"github.com/alumasinde/tuma254-api/internal/deliveries/models"
)

type DeliveryRepository interface {
	Create(ctx context.Context, delivery models.Delivery) error
	FindByID(ctx context.Context, id string) (models.Delivery, error)
	ListBySender(ctx context.Context, senderUserID string, limit, offset int) ([]models.Delivery, error)
	ClaimAssignment(ctx context.Context, deliveryID, riderID string, at time.Time) (bool, error)
	CountActiveForRider(ctx context.Context, riderID string) (int64, error)
}

type TxRunner interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}
