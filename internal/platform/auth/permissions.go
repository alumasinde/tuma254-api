package auth

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, permissionCode string) (bool, error)
}

type DBPermissionChecker struct {
	db interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	}
}

func NewDBPermissionChecker(db interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) *DBPermissionChecker {
	return &DBPermissionChecker{db: db}
}

func (c *DBPermissionChecker) HasPermission(ctx context.Context, userID, permissionCode string) (bool, error) {
	var allowed bool
	err := c.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_roles ur
			JOIN role_permissions rp ON rp.role_id = ur.role_id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ur.user_id = $1
			  AND p.code = $2
		)
	`, userID, permissionCode).Scan(&allowed)
	return allowed, err
}

func RequirePermission(checker PermissionChecker, permissionCode string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := PrincipalFromContext(r.Context())
		if err != nil {
			writeError(w, http.StatusUnauthorized, ErrMissingToken)
			return
		}
		allowed, err := checker.HasPermission(r.Context(), principal.UserID, permissionCode)
		if err != nil {
			writeError(w, http.StatusInternalServerError, ErrAuthorizationCheckFailed)
			return
		}
		if !allowed {
			writeError(w, http.StatusForbidden, ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
