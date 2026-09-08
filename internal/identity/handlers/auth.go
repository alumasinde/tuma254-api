package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/alumasinde/tuma254-api/internal/identity/dtos"
	"github.com/alumasinde/tuma254-api/internal/identity/repositories"
	"github.com/alumasinde/tuma254-api/internal/identity/services"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in dtos.RegisterRequest
	if !decode(w, r, &in) { return }
	out, err := h.svc.Register(r.Context(), in)
	if err != nil {
		status, message := identityErrorResponse(err)
		http.Error(w, message, status)
		return
	}
	write(w, http.StatusCreated, out)
}

func (h *Handler) VerifyPhone(w http.ResponseWriter, r *http.Request) {
	var in dtos.VerifyPhoneRequest
	if !decode(w, r, &in) { return }
	out, err := h.svc.VerifyPhone(r.Context(), in, r.UserAgent(), clientIP(r))
	if err != nil {
		status, message := identityErrorResponse(err)
		http.Error(w, message, status)
		return
	}
	write(w, http.StatusOK, out)
}

func (h *Handler) ResendPhoneVerification(w http.ResponseWriter, r *http.Request) {
	var in dtos.ResendPhoneVerificationRequest
	if !decode(w, r, &in) { return }
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
	if !decode(w, r, &in) { return }
	out, err := h.svc.Login(r.Context(), in, r.UserAgent(), clientIP(r))
	if err != nil { status, message := identityErrorResponse(err); http.Error(w, message, status); return }
	write(w, http.StatusOK, out)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var in dtos.RefreshRequest
	if !decode(w, r, &in) { return }
	if strings.TrimSpace(in.RefreshToken) == "" { http.Error(w, "invalid refresh token", http.StatusUnauthorized); return }
	out, err := h.svc.Refresh(r.Context(), in.RefreshToken, r.UserAgent(), clientIP(r))
	if err != nil { status, message := identityErrorResponse(err); http.Error(w, message, status); return }
	write(w, http.StatusOK, out)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var in dtos.RefreshRequest
	if !decode(w, r, &in) { return }
	_ = h.svc.Logout(r.Context(), in.RefreshToken)
	w.WriteHeader(http.StatusNoContent)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil { http.Error(w, "invalid request", http.StatusBadRequest); return false }
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) { http.Error(w, "invalid request", http.StatusBadRequest); return false }
	return true
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil { return host }
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
	case errors.Is(err, services.ErrInvalidCredentials), errors.Is(err, services.ErrInvalidToken):
		return http.StatusUnauthorized, "authentication failed"
	case errors.Is(err, repositories.ErrOTPCooldown), errors.Is(err, repositories.ErrOTPRateLimited):
		return http.StatusTooManyRequests, "verification code cannot be sent yet"
	default:
		return http.StatusInternalServerError, "identity request failed"
	}
}
