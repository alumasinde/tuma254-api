package repositories

import (
	"context"
	"net"
	"strings"

	"github.com/alumasinde/tuma254-api/internal/identity/models"
	"github.com/google/uuid"
)

func (r *Postgres) CreateSession(ctx context.Context, p CreateSessionParams) error {
	_, err := r.db.Exec(ctx, "INSERT INTO refresh_sessions(user_id,token_hash,expires_at,user_agent,ip_address) VALUES($1,$2,$3,$4,$5)", p.UserID, p.TokenHash, p.ExpiresAt, p.UserAgent, parseIPAddress(p.IPAddress))
	return err
}

func (r *Postgres) RotateSession(ctx context.Context, p RotateSessionParams) (models.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.User{}, err
	}
	defer tx.Rollback(ctx)
	var userID uuid.UUID
	if err = tx.QueryRow(ctx, "UPDATE refresh_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>now() RETURNING user_id", p.CurrentTokenHash).Scan(&userID); err != nil {
		return models.User{}, err
	}
	u, err := loadUserWithRolesTx(ctx, tx, userID)
	if err != nil {
		return models.User{}, err
	}
	if !u.Active {
		return models.User{}, ErrAccountInactive
	}
	if _, err = tx.Exec(ctx, "INSERT INTO refresh_sessions(user_id,token_hash,expires_at,user_agent,ip_address) VALUES($1,$2,$3,$4,$5)", userID, p.ReplacementTokenHash, p.ReplacementExpiresAt, p.UserAgent, parseIPAddress(p.IPAddress)); err != nil {
		return models.User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return models.User{}, err
	}
	return u, nil
}

func (r *Postgres) RevokeSession(ctx context.Context, tokenHash []byte) error {
	_, err := r.db.Exec(ctx, "UPDATE refresh_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL", tokenHash)
	return err
}
func parseIPAddress(value string) any {
	if parsed := net.ParseIP(strings.TrimSpace(value)); parsed != nil {
		return parsed.String()
	}
	return nil
}
