package repositories

import (
	"context"

	"github.com/alumasinde/tuma254-api/internal/identity/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func loadUserWithRolesTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (models.User, error) {
	var user models.User
	if err := tx.QueryRow(ctx, `SELECT id,email,phone,first_name,last_name,is_active,phone_verified_at,created_at FROM users WHERE id=$1`, userID).
		Scan(&user.ID, &user.Email, &user.Phone, &user.FirstName, &user.LastName, &user.Active, &user.PhoneVerifiedAt, &user.CreatedAt); err != nil {
		return models.User{}, err
	}
	rows, err := tx.Query(ctx, `SELECT ro.name FROM roles ro JOIN user_roles ur ON ur.role_id=ro.id WHERE ur.user_id=$1 ORDER BY ro.name`, userID)
	if err != nil {
		return models.User{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return models.User{}, err
		}
		user.Roles = append(user.Roles, role)
	}
	if err := rows.Err(); err != nil {
		return models.User{}, err
	}
	return user, nil
}
