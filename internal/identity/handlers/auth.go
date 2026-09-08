package handlers

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/alumasinde/tuma254-api/internal/identity/dtos"
	"github.com/alumasinde/tuma254-api/internal/identity/repositories"
	"github.com/alumasinde/tuma254-api/internal/identity/services"
	"github.com/alumasinde/tuma254-api/internal/platform/httpx"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in dtos.RegisterRequest
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	out, err := h.svc.Register(r.Context(), in)
	if err != nil {
		status, message := identityErrorResponse(err)
		http.Error(w, message, status)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}

func (h *Handler) VerifyPhone(w http.ResponseWriter, r *http.Request) {
	var in dtos.VerifyPhoneRequest
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	out, err := h.svc.VerifyPhone(r.Context(), in, r.UserAgent(), clientIP(r))
	if err != nil {
		status, message := identityErrorResponse(err)
		http.Error(w, message, status)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) ResendPhoneVerification(w http.ResponseWriter, r *http.Request) {
	var in dtos.ResendPhoneVerificationRequest
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	err := h.svc.ResendPhoneVerification(r.Context(), in.Phone)
	if errors.Is(err, repositories.ErrOTPCooldown) || errors.Is(err, repositories.ErrOTPRateLimited) {
		http.Error(w, "verification code cannot be sent yet", http.StatusTooManyRequests)
		return
	}
	if err != nil {
		http.Error(w, "verification code could not be sent", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in dtos.LoginRequest
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	out, err := h.svc.Login(r.Context(), in, r.UserAgent(), clientIP(r))
	if err != nil {
		status, message := identityErrorResponse(err)
		http.Error(w, message, status)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var in dtos.RefreshRequest
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	if strings.TrimSpace(in.RefreshToken) == "" {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}
	out, err := h.svc.Refresh(r.Context(), in.RefreshToken, r.UserAgent(), clientIP(r))
	if err != nil {
		status, message := identityErrorResponse(err)
		http.Error(w, message, status)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var in dtos.RefreshRequest
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	_ = h.svc.Logout(r.Context(), in.RefreshToken)
	w.WriteHeader(http.StatusNoContent)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func identityErrorResponse(err error) (int, string) {
	switch {
	case errors.Is(err, services.ErrEmailAlreadyRegistered):
		return http.StatusConflict, "email already registered"
	case errors.Is(err, services.ErrPhoneAlreadyRegistered):
		return http.StatusConflict, "phone already registered"
	case errors.Is(err, services.ErrInvalidRegistration), errors.Is(err, services.ErrInvalidPhone):
		return http.StatusBadRequest, "invalid registration data"
	case errors.Is(err, services.ErrInvalidOTP):
		return http.StatusUnauthorized, "invalid verification code"
	case errors.Is(err, services.ErrExpiredOTP), errors.Is(err, services.ErrOTPLocked):
		return http.StatusGone, "verification code expired or locked"
	case errors.Is(err, services.ErrAccountInactive), errors.Is(err, services.ErrPhoneNotVerified):
		return http.StatusForbidden, "account is not allowed to authenticate"
	case errors.Is(err, services.ErrLoginRateLimited):
		return http.StatusTooManyRequests, "too many login attempts"
	case errors.Is(err, services.ErrInvalidCredentials), errors.Is(err, services.ErrInvalidToken):
		return http.StatusUnauthorized, "authentication failed"
	case errors.Is(err, repositories.ErrOTPCooldown), errors.Is(err, repositories.ErrOTPRateLimited):
		return http.StatusTooManyRequests, "verification code cannot be sent yet"
	default:
		return http.StatusInternalServerError, "identity request failed"
	}
}
