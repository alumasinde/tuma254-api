package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/alumasinde/tuma254-api/internal/identity/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrOTPCooldown     = errors.New("otp resend cooldown")
	ErrOTPRateLimited  = errors.New("otp resend limit")
	ErrAccountInactive = errors.New("account inactive")
)

type CreateUserParams struct{ Email, Phone, FirstName, LastName, PasswordHash string }
type IssueOTPParams struct {
	UserID                 uuid.UUID
	Phone, Purpose         string
	CodeHash               []byte
	ExpiresAt              time.Time
	MaxAttempts            int
	Cooldown, ResendWindow time.Duration
	MaxResends             int
}
type CreateSessionParams struct {
	UserID               uuid.UUID
	TokenHash            []byte
	ExpiresAt            time.Time
	UserAgent, IPAddress string
}
type RotateSessionParams struct {
	CurrentTokenHash, ReplacementTokenHash []byte
	ReplacementExpiresAt                   time.Time
	UserAgent, IPAddress                   string
}

type UsersRepository interface {
	CreateUser(context.Context, CreateUserParams) (models.User, error)
	FindByEmail(context.Context, string) (models.User, string, error)
	FindByPhone(context.Context, string) (models.User, error)
	FindByID(context.Context, uuid.UUID) (models.User, error)
}
type RolesRepository interface {
	FindRolesByUserID(context.Context, uuid.UUID) ([]string, error)
	AssignRole(context.Context, uuid.UUID, string) error
}
type OTPRepository interface {
	IssueOTP(context.Context, IssueOTPParams) error
	RevokeActiveOTP(context.Context, uuid.UUID, string) error
	VerifyOTP(context.Context, string, string, []byte) (models.OTPVerifyResult, error)
}
type SessionsRepository interface {
	CreateSession(context.Context, CreateSessionParams) error
	RotateSession(context.Context, RotateSessionParams) (models.User, error)
	RevokeSession(context.Context, []byte) error
}
type Repository interface {
	UsersRepository
	RolesRepository
	OTPRepository
	SessionsRepository
}
type Postgres struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Postgres { return &Postgres{db: db} }

var _ Repository = (*Postgres)(nil)
var _ pgx.Tx = (pgx.Tx)(nil)
