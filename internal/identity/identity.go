package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidRegistration = errors.New("invalid registration data")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserExists          = errors.New("email or phone is already registered")
)

var phonePattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

type Service struct {
	db                            *pgxpool.Pool
	accessSecret, refreshSecret  string
	accessTTL, refreshTTL        time.Duration
}

func New(db *pgxpool.Pool, accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{
		db:            db,
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

type User struct {
	ID        string   `json:"id"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Email     *string  `json:"email,omitempty"`
	Phone     *string  `json:"phone,omitempty"`
	Status    string   `json:"status"`
	Roles     []string `json:"roles"`
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

func normEmail(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func normPhone(v string) string {
	v = strings.TrimSpace(v)
	replacer := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")
	v = replacer.Replace(v)

	switch {
	case strings.HasPrefix(v, "07") && len(v) == 10:
		return "+254" + v[1:]
	case strings.HasPrefix(v, "254") && len(v) == 12:
		return "+" + v
	default:
		return v
	}
}

func validEmail(v string) bool {
	if v == "" {
		return true
	}
	addr, err := mail.ParseAddress(v)
	return err == nil && addr.Address == v
}

func validPhone(v string) bool {
	return v == "" || phonePattern.MatchString(v)
}

func (s *Service) Register(ctx context.Context, first, last, email, phone, password string) (User, Tokens, error) {
	first = strings.TrimSpace(first)
	last = strings.TrimSpace(last)
	email = normEmail(email)
	phone = normPhone(phone)

	if first == "" || last == "" || len(password) < 8 || (email == "" && phone == "") || !validEmail(email) || !validPhone(phone) {
		return User{}, Tokens{}, ErrInvalidRegistration
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, Tokens{}, err
	}

	var user User
	var tokens Tokens

	err = pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO users (first_name, last_name, email, phone, password_hash)
			VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5)
			RETURNING id, first_name, last_name, email, phone, status
		`, first, last, email, phone, string(hash)).Scan(
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Email,
			&user.Phone,
			&user.Status,
		)
		if err != nil {
			return err
		}

		tag, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id)
			SELECT $1, id
			FROM roles
			WHERE code = 'customer'
		`, user.ID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errors.New("default customer role is not configured")
		}

		user.Roles = []string{"customer"}

		tokens, err = s.issueWithStore(ctx, tx, user)
		return err
	})
	if err != nil {
		return User{}, Tokens{}, err
	}

	return user, tokens, nil
}

func (s *Service) Login(ctx context.Context, identifier, password string) (User, Tokens, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" || password == "" {
		return User{}, Tokens{}, ErrInvalidCredentials
	}

	email := normEmail(identifier)
	phone := normPhone(identifier)

	var user User
	var passwordHash string

	err := s.db.QueryRow(ctx, `
		SELECT id::text, first_name, last_name, email, phone, password_hash, status
		FROM users
		WHERE (lower(email) = lower($1) OR phone = $2)
		  AND status = 'active'
	`, email, phone).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Phone,
		&passwordHash,
		&user.Status,
	)
	if err != nil {
		return User{}, Tokens{}, ErrInvalidCredentials
	}

	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return User{}, Tokens{}, ErrInvalidCredentials
	}

	if err := s.loadRoles(ctx, s.db, &user); err != nil {
		return User{}, Tokens{}, fmt.Errorf("load login roles: %w", err)
	}

	tokens, err := s.issue(ctx, user)
	if err != nil {
		return User{}, Tokens{}, fmt.Errorf("issue login tokens: %w", err)
	}

	return user, tokens, nil
}

func (s *Service) Refresh(ctx context.Context, raw string) (User, Tokens, error) {
	claims, err := s.parseRefreshToken(raw)
	if err != nil {
		return User{}, Tokens{}, ErrInvalidRefreshToken
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return User{}, Tokens{}, ErrInvalidRefreshToken
	}

	hash := tokenHash(raw)
	var user User
	var tokens Tokens

	err = pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var storedUserID string
		err := tx.QueryRow(ctx, `
			UPDATE refresh_tokens
			SET revoked_at = now()
			WHERE token_hash = $1
			  AND revoked_at IS NULL
			  AND expires_at > now()
			RETURNING user_id::text
		`, hash).Scan(&storedUserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInvalidRefreshToken
			}
			return fmt.Errorf("revoke refresh token: %w", err)
		}
		if storedUserID != userID {
			return ErrInvalidRefreshToken
		}

		err = tx.QueryRow(ctx, `
			SELECT id::text, first_name, last_name, email, phone, status
			FROM users
			WHERE id = $1 AND status = 'active'
		`, userID).Scan(
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Email,
			&user.Phone,
			&user.Status,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInvalidRefreshToken
			}
			return fmt.Errorf("load refresh user: %w", err)
		}

		if err := s.loadRoles(ctx, tx, &user); err != nil {
			return fmt.Errorf("load refresh roles: %w", err)
		}

		tokens, err = s.issueWithStore(ctx, tx, user)
		if err != nil {
			return fmt.Errorf("issue rotated refresh token: %w", err)
		}
		return nil
	})
	if err != nil {
		return User{}, Tokens{}, err
	}

	return user, tokens, nil
}

func (s *Service) Logout(ctx context.Context, raw string) error {
	claims, err := s.parseRefreshToken(raw)
	if err != nil {
		return ErrInvalidRefreshToken
	}

	if userID, ok := claims["sub"].(string); !ok || userID == "" {
		return ErrInvalidRefreshToken
	}

	tag, err := s.db.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > now()
	`, tokenHash(raw))
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrInvalidRefreshToken
	}

	return nil
}

func (s *Service) loadRoles(ctx context.Context, db interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, user *User) error {
	rows, err := db.Query(ctx, `
		SELECT r.code
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.code
	`, user.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	user.Roles = user.Roles[:0]
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return err
		}
		user.Roles = append(user.Roles, role)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(user.Roles) == 0 {
		return errors.New("user has no assigned roles")
	}

	return nil
}

func (s *Service) issue(ctx context.Context, user User) (Tokens, error) {
	return s.issueWithStore(ctx, s.db, user)
}

func (s *Service) issueWithStore(ctx context.Context, db interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, user User) (Tokens, error) {
	now := time.Now().UTC()

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   user.ID,
		"roles": user.Roles,
		"typ":   "access",
		"iat":   now.Unix(),
		"exp":   now.Add(s.accessTTL).Unix(),
	}).SignedString([]byte(s.accessSecret))
	if err != nil {
		return Tokens{}, err
	}

	refreshExpiresAt := now.Add(s.refreshTTL)
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"typ": "refresh",
		"iat": now.Unix(),
		"exp": refreshExpiresAt.Unix(),
	}).SignedString([]byte(s.refreshSecret))
	if err != nil {
		return Tokens{}, err
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, user.ID, tokenHash(refreshToken), refreshExpiresAt); err != nil {
		return Tokens{}, err
	}

	return Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

func (s *Service) parseRefreshToken(raw string) (jwt.MapClaims, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, ErrInvalidRefreshToken
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(s.refreshSecret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return nil, ErrInvalidRefreshToken
	}
	if claims["typ"] != "refresh" {
		return nil, ErrInvalidRefreshToken
	}

	return claims, nil
}

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
