package identity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

func (s *Service) CurrentUser(ctx context.Context, userID string) (User, error) {
	var user User
	if err := s.db.QueryRow(ctx, `
		SELECT id::text, first_name, last_name, email, phone, status
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Phone,
		&user.Status,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	if err := s.loadRoles(ctx, s.db, &user); err != nil {
		return User{}, err
	}
	return user, nil
}

var _ *pgxpool.Pool
