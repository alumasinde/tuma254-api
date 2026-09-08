package services

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

func mapRegistrationError(err error) error {
	if err == nil {
		return nil
	}

	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		switch databaseError.Code {
		case "23505":
			switch {
			case strings.Contains(databaseError.ConstraintName, "email"):
				return ErrEmailAlreadyRegistered
			case strings.Contains(databaseError.ConstraintName, "phone"):
				return ErrPhoneAlreadyRegistered
			default:
				return ErrInvalidRegistration
			}
		}
	}

	return err
}
