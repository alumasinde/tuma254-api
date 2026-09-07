package repositories

import (
	"context"
	"crypto/subtle"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/alumasinde/tuma254-api/internal/identity/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Postgres) CreateUser(ctx context.Context, email, phone, first, last, hash string) (models.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil { return models.User{}, err }
	defer tx.Rollback(ctx)

	var u models.User
	err = tx.QueryRow(ctx, `INSERT INTO users(email,phone,password_hash,first_name,last_name,is_active,verification_required)
		VALUES($1,$2,$3,$4,$5,FALSE,TRUE)
		RETURNING id,email,phone,first_name,last_name,is_active,phone_verified_at,created_at`,
		email, phone, hash, first, last,
	).Scan(&u.ID, &u.Email, &u.Phone, &u.FirstName, &u.LastName, &u.Active, &u.PhoneVerifiedAt, &u.CreatedAt)
	if err != nil { return models.User{}, err }

	if _, err = tx.Exec(ctx, `INSERT INTO user_roles(user_id,role_id)
		SELECT $1,id FROM roles WHERE name='customer'`, u.ID); err != nil {
		return models.User{}, err
	}
	if err = tx.Commit(ctx); err != nil { return models.User{}, err }
	u.Roles = []string{"customer"}
	return u, nil
}

func (r *Postgres) FindByEmail(ctx context.Context, email string) (models.User, string, error) {
	var u models.User
	var hash string
	err := r.db.QueryRow(ctx, `SELECT id,email,phone,password_hash,first_name,last_name,is_active,phone_verified_at,created_at
		FROM users WHERE lower(email)=lower($1)`, email,
	).Scan(&u.ID, &u.Email, &u.Phone, &hash, &u.FirstName, &u.LastName, &u.Active, &u.PhoneVerifiedAt, &u.CreatedAt)
	if err != nil { return models.User{}, "", err }
	rs, err := r.roles(ctx, u.ID)
	u.Roles = rs
	return u, hash, err
}

func (r *Postgres) FindByPhone(ctx context.Context, phone string) (models.User, error) {
	var u models.User
	err := r.db.QueryRow(ctx, `SELECT id,email,phone,first_name,last_name,is_active,phone_verified_at,created_at
		FROM users WHERE phone=$1`, phone,
	).Scan(&u.ID, &u.Email, &u.Phone, &u.FirstName, &u.LastName, &u.Active, &u.PhoneVerifiedAt, &u.CreatedAt)
	if err != nil { return models.User{}, err }
	rs, err := r.roles(ctx, u.ID)
	u.Roles = rs
	return u, err
}

func (r *Postgres) FindByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	var u models.User
	err := r.db.QueryRow(ctx, `SELECT id,email,phone,first_name,last_name,is_active,phone_verified_at,created_at
		FROM users WHERE id=$1`, id,
	).Scan(&u.ID, &u.Email, &u.Phone, &u.FirstName, &u.LastName, &u.Active, &u.PhoneVerifiedAt, &u.CreatedAt)
	if err != nil { return models.User{}, err }
	rs, err := r.roles(ctx, id)
	u.Roles = rs
	return u, err
}

func (r *Postgres) roles(ctx context.Context, id uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT ro.name FROM roles ro
		JOIN user_roles ur ON ur.role_id=ro.id
		WHERE ur.user_id=$1 ORDER BY ro.name`, id)
	if err != nil { return nil, err }
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err = rows.Scan(&s); err != nil { return nil, err }
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Postgres) IssueOTP(ctx context.Context, userID uuid.UUID, phone, purpose string, codeHash []byte, expiresAt time.Time, maxAttempts int, cooldown, window time.Duration, maxResends int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)

	// Lock the user row so concurrent resend requests serialize per identity.
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT TRUE FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&exists); err != nil {
		return err
	}

	var createdAt time.Time
	err = tx.QueryRow(ctx, `SELECT created_at FROM otp_challenges
		WHERE user_id=$1 AND purpose=$2
		ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, userID, purpose).Scan(&createdAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) { return err }
	if err == nil && time.Since(createdAt) < cooldown { return ErrOTPCooldown }

	var count int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM otp_challenges
		WHERE user_id=$1 AND purpose=$2 AND created_at >= now() - ($3 * interval '1 second')`,
		userID, purpose, int64(window.Seconds())).Scan(&count); err != nil {
		return err
	}
	if count >= maxResends { return ErrOTPRateLimited }

	if _, err = tx.Exec(ctx, `UPDATE otp_challenges SET revoked_at=now()
		WHERE user_id=$1 AND purpose=$2 AND verified_at IS NULL AND revoked_at IS NULL`, userID, purpose); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO otp_challenges(user_id,phone,purpose,code_hash,expires_at,max_attempts)
		VALUES($1,$2,$3,$4,$5,$6)`, userID, phone, purpose, codeHash, expiresAt, maxAttempts); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Postgres) RevokeActiveOTP(ctx context.Context, userID uuid.UUID, purpose string) error {
	_, err := r.db.Exec(ctx, `UPDATE otp_challenges SET revoked_at=now()
		WHERE user_id=$1 AND purpose=$2 AND verified_at IS NULL AND revoked_at IS NULL`, userID, purpose)
	return err
}

func (r *Postgres) VerifyOTP(ctx context.Context, phone, purpose string, codeHash []byte) (models.OTPVerifyResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil { return models.OTPVerifyResult{}, err }
	defer tx.Rollback(ctx)

	var challengeID uuid.UUID
	var userID uuid.UUID
	var stored []byte
	var expiresAt time.Time
	var attempts, maxAttempts int
	err = tx.QueryRow(ctx, `SELECT id,user_id,code_hash,expires_at,attempt_count,max_attempts
		FROM otp_challenges
		WHERE phone=$1 AND purpose=$2 AND verified_at IS NULL AND revoked_at IS NULL
		ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, phone, purpose,
	).Scan(&challengeID, &userID, &stored, &expiresAt, &attempts, &maxAttempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.OTPVerifyResult{}, tx.Commit(ctx)
	}
	if err != nil { return models.OTPVerifyResult{}, err }

	if !time.Now().Before(expiresAt) {
		if _, err = tx.Exec(ctx, `UPDATE otp_challenges SET revoked_at=now() WHERE id=$1`, challengeID); err != nil {
			return models.OTPVerifyResult{}, err
		}
		if err = tx.Commit(ctx); err != nil { return models.OTPVerifyResult{}, err }
		return models.OTPVerifyResult{Expired: true}, nil
	}

	if subtle.ConstantTimeCompare(stored, codeHash) != 1 {
		attempts++
		exhausted := attempts >= maxAttempts
		if exhausted {
			_, err = tx.Exec(ctx, `UPDATE otp_challenges SET attempt_count=$2,revoked_at=now() WHERE id=$1`, challengeID, attempts)
		} else {
			_, err = tx.Exec(ctx, `UPDATE otp_challenges SET attempt_count=$2 WHERE id=$1`, challengeID, attempts)
		}
		if err != nil { return models.OTPVerifyResult{}, err }
		if err = tx.Commit(ctx); err != nil { return models.OTPVerifyResult{}, err }
		return models.OTPVerifyResult{Exhausted: exhausted}, nil
	}

	if _, err = tx.Exec(ctx, `UPDATE otp_challenges SET verified_at=now() WHERE id=$1`, challengeID); err != nil {
		return models.OTPVerifyResult{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE users
		SET phone_verified_at=COALESCE(phone_verified_at,now()),verification_required=FALSE,is_active=TRUE,updated_at=now()
		WHERE id=$1 AND phone=$2`, userID, phone); err != nil {
		return models.OTPVerifyResult{}, err
	}

	var u models.User
	err = tx.QueryRow(ctx, `SELECT id,email,phone,first_name,last_name,is_active,phone_verified_at,created_at
		FROM users WHERE id=$1`, userID,
	).Scan(&u.ID, &u.Email, &u.Phone, &u.FirstName, &u.LastName, &u.Active, &u.PhoneVerifiedAt, &u.CreatedAt)
	if err != nil { return models.OTPVerifyResult{}, err }

	rows, err := tx.Query(ctx, `SELECT ro.name FROM roles ro
		JOIN user_roles ur ON ur.role_id=ro.id
		WHERE ur.user_id=$1 ORDER BY ro.name`, userID)
	if err != nil { return models.OTPVerifyResult{}, err }
	for rows.Next() {
		var role string
		if err = rows.Scan(&role); err != nil { rows.Close(); return models.OTPVerifyResult{}, err }
		u.Roles = append(u.Roles, role)
	}
	rows.Close()
	if err = rows.Err(); err != nil { return models.OTPVerifyResult{}, err }
	if err = tx.Commit(ctx); err != nil { return models.OTPVerifyResult{}, err }
	return models.OTPVerifyResult{User: u, Verified: true}, nil
}

func (r *Postgres) CreateSession(ctx context.Context, id uuid.UUID, h []byte, exp time.Time, ua, ip string) error {
	var addr any
	if p := net.ParseIP(strings.TrimSpace(ip)); p != nil { addr = p.String() }
	_, err := r.db.Exec(ctx, `INSERT INTO refresh_sessions(user_id,token_hash,expires_at,user_agent,ip_address)
		VALUES($1,$2,$3,$4,$5)`, id, h, exp, ua, addr)
	return err
}

func (r *Postgres) ConsumeSession(ctx context.Context, h []byte) (models.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil { return models.User{}, err }
	defer tx.Rollback(ctx)
	var id uuid.UUID
	err = tx.QueryRow(ctx, `UPDATE refresh_sessions SET revoked_at=now()
		WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>now()
		RETURNING user_id`, h).Scan(&id)
	if err != nil { return models.User{}, err }

	var u models.User
	err = tx.QueryRow(ctx, `SELECT id,email,phone,first_name,last_name,is_active,phone_verified_at,created_at
		FROM users WHERE id=$1`, id,
	).Scan(&u.ID, &u.Email, &u.Phone, &u.FirstName, &u.LastName, &u.Active, &u.PhoneVerifiedAt, &u.CreatedAt)
	if err != nil || !u.Active { return models.User{}, errors.New("inactive") }

	rows, err := tx.Query(ctx, `SELECT ro.name FROM roles ro
		JOIN user_roles ur ON ur.role_id=ro.id
		WHERE ur.user_id=$1 ORDER BY ro.name`, id)
	if err != nil { return models.User{}, err }
	for rows.Next() {
		var role string
		if err = rows.Scan(&role); err != nil { rows.Close(); return models.User{}, err }
		u.Roles = append(u.Roles, role)
	}
	rows.Close()
	if err = rows.Err(); err != nil { return models.User{}, err }
	if err = tx.Commit(ctx); err != nil { return models.User{}, err }
	return u, nil
}

func (r *Postgres) RevokeSession(ctx context.Context, h []byte) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_sessions SET revoked_at=now()
		WHERE token_hash=$1 AND revoked_at IS NULL`, h)
	return err
}