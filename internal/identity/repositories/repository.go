package repositories

import (
	"context"

	"github.com/alumasinde/tuma254-api/internal/identity/models"
)

var ErrNotFound = errNotFound("identity record not found")
type errNotFound string
func (e errNotFound) Error() string { return string(e) }

type UserRepository interface {
	Create(ctx context.Context, user models.User, defaultRole string) (models.User, error)
	FindByIdentifier(ctx context.Context, identifier string) (models.User, error)
	FindPublicByID(ctx context.Context, userID string) (models.PublicUser, error)
	ReplacePasswordHash(ctx context.Context, userID, passwordHash string) error
}

type SessionRepository interface {
	Create(ctx context.Context, userID, tokenHash string) error
	Consume(ctx context.Context, tokenHash string) (string, error)
	Revoke(ctx context.Context, tokenHash string) error
}
