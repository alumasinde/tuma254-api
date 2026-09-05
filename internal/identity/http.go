package identity

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

type Handler struct {
	svc *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{svc: s}
}

type authReq struct {
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Password     string `json:"password"`
	Identifier   string `json:"identifier"`
	RefreshToken string `json:"refresh_token"`
}

type authResponse struct {
	User   User   `json:"user"`
	Tokens Tokens `json:"tokens"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req authReq
	if err := decodeJSON(w, r, &req); err != nil {
		errJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	user, tokens, err := h.svc.Register(
		r.Context(),
		req.FirstName,
		req.LastName,
		req.Email,
		req.Phone,
		req.Password,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRegistration):
			errJSON(w, http.StatusBadRequest, ErrInvalidRegistration.Error())
		case isUniqueViolation(err):
			errJSON(w, http.StatusConflict, ErrUserExists.Error())
		default:
			slog.Error("identity registration failed", "error", err)
		errJSON(w, http.StatusInternalServerError, "registration failed")
		}
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{User: user, Tokens: tokens})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req authReq
	if err := decodeJSON(w, r, &req); err != nil {
		errJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	user, tokens, err := h.svc.Login(r.Context(), req.Identifier, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			errJSON(w, http.StatusUnauthorized, ErrInvalidCredentials.Error())
			return
		}
		slog.Error("identity login failed", "error", err)
		errJSON(w, http.StatusInternalServerError, "login failed")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{User: user, Tokens: tokens})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req authReq
	if err := decodeJSON(w, r, &req); err != nil {
		errJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	user, tokens, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			errJSON(w, http.StatusUnauthorized, ErrInvalidRefreshToken.Error())
			return
		}
		slog.Error("identity refresh failed", "error", err)
		errJSON(w, http.StatusInternalServerError, "refresh failed")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{User: user, Tokens: tokens})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req authReq
	if err := decodeJSON(w, r, &req); err != nil {
		errJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			errJSON(w, http.StatusUnauthorized, ErrInvalidRefreshToken.Error())
			return
		}
		slog.Error("identity logout failed", "error", err)
		errJSON(w, http.StatusInternalServerError, "logout failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return errors.New("invalid json")
	}
	if decoder.More() {
		return errors.New("invalid json")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func errJSON(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": strings.TrimSpace(message)})
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
