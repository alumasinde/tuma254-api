package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/alumasinde/tuma254-api/internal/users/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("profile not found")

type Repository struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) FindByUserID(ctx context.Context, userID uuid.UUID) (models.Profile, error) {
	var p models.Profile
	err := r.db.QueryRow(ctx, `SELECT user_id, avatar_url, created_at, updated_at FROM user_profiles WHERE user_id=$1`, userID).
		Scan(&p.UserID, &p.AvatarURL, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) { return models.Profile{}, ErrNotFound }
	if err != nil { return models.Profile{}, err }
	return p, nil
}

func (r *Repository) Upsert(ctx context.Context, userID uuid.UUID, avatarURL string) (models.Profile, error) {
	var p models.Profile
	now := time.Now().UTC()
	err := r.db.QueryRow(ctx, `
INSERT INTO user_profiles(user_id, avatar_url, created_at, updated_at)
VALUES($1,$2,$3,$3)
ON CONFLICT(user_id) DO UPDATE SET avatar_url=EXCLUDED.avatar_url, updated_at=EXCLUDED.updated_at
RETURNING user_id, avatar_url, created_at, updated_at`, userID, avatarURL, now).
		Scan(&p.UserID, &p.AvatarURL, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}
