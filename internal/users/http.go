package users

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/alumasinde/tuma254-api/internal/identity"
	httpserver "github.com/alumasinde/tuma254-api/internal/platform/http"
)

type Authenticator interface {
	RequireAuth(http.HandlerFunc) http.HandlerFunc
}

type Handler struct {
	service *Service
	auth    Authenticator
}

func NewHandler(service *Service, auth Authenticator) *Handler {
	return &Handler{service: service, auth: auth}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/profile", h.auth.RequireAuth(h.get))
	mux.HandleFunc("PUT /api/v1/profile", h.auth.RequireAuth(h.update))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	claims, ok := identity.ClaimsFromContext(r.Context())
	if !ok {
		httpserver.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_session"})
		return
	}
	profile, err := h.service.Get(r.Context(), claims.Subject)
	if err != nil {
		httpserver.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_session"})
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, profile)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	claims, ok := identity.ClaimsFromContext(r.Context())
	if !ok {
		httpserver.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_session"})
		return
	}
	var input struct {
		AvatarURL string `json:"avatar_url"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	profile, err := h.service.Update(r.Context(), claims.Subject, input.AvatarURL)
	if err != nil {
		httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "profile_update_failed"})
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, profile)
}
