package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/alumasinde/tuma254-api/internal/identity/repositories"
	"github.com/alumasinde/tuma254-api/internal/identity/services"
)

func TestIdentityErrorResponse(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{services.ErrEmailAlreadyRegistered, http.StatusConflict},
		{services.ErrPhoneAlreadyRegistered, http.StatusConflict},
		{services.ErrInvalidRegistration, http.StatusBadRequest},
		{services.ErrInvalidOTP, http.StatusUnauthorized},
		{services.ErrExpiredOTP, http.StatusGone},
		{services.ErrAccountInactive, http.StatusForbidden},
		{services.ErrInvalidCredentials, http.StatusUnauthorized},
		{repositories.ErrOTPCooldown, http.StatusTooManyRequests},
		{errors.New("unexpected"), http.StatusInternalServerError},
	}

	for _, test := range tests {
		status, _ := identityErrorResponse(test.err)
		if status != test.status {
			t.Fatalf("status = %d, expected %d", status, test.status)
		}
	}
}
