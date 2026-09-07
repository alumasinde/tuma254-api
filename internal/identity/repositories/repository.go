package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/alumasinde/tuma254-api/internal/identity/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrOTPCooldown    = errors.New("otp resend cooldown")
	ErrOTPRateLimited = errors.New("otp resend limit")
)

type Repository interface {
	CreateUser(context.Context, string, string, string, string, string) (models.User, error)
	FindByEmail(context.Context, string) (models.User, string, error)
	FindByPhone(context.Context, string) (models.User, error)
	FindByID(context.Context, uuid.UUID) (models.User, error)

	IssueOTP(context.Context, uuid.UUID, string, string, []byte, time.Time, int, time.Duration, time.Duration, int) error
	RevokeActiveOTP(context.Context, uuid.UUID, string) error
	VerifyOTP(context.Context, string, string, []byte) (models.OTPVerifyResult, error)

	CreateSession(context.Context, uuid.UUID, []byte, time.Time, string, string) error
	RotateSession(context.Context, []byte, []byte, time.Time, string, string) (models.User, error)
	RevokeSession(context.Context, []byte) error
}

type Postgres struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Postgres { return &Postgres{db: db} }
