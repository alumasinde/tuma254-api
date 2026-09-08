package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alumasinde/tuma254-api/internal/users/services"
	"github.com/google/uuid"
)

type Handler struct{ svc *services.Service }

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := uuidFromContext(r)
	if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	profile, err := h.svc.Get(r.Context(), id)
	if err != nil { http.Error(w, "profile not found", http.StatusNotFound); return }
	write(w, http.StatusOK, profile)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := uuidFromContext(r)
	if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	var input struct{ AvatarURL string `json:"avatar_url"` }
	if !decode(w, r, &input) { return }
	profile, err := h.svc.Update(r.Context(), id, strings.TrimSpace(input.AvatarURL))
	if err != nil { http.Error(w, "profile update failed", http.StatusBadRequest); return }
	write(w, http.StatusOK, profile)
}

func uuidFromContext(r *http.Request) (uuid.UUID, bool) {
	value, ok := r.Context().Value("tuma254.user_id").(string)
	if !ok { return uuid.Nil, false }
	id, err := uuid.Parse(value)
	return id, err == nil
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil { http.Error(w, "invalid request", http.StatusBadRequest); return false }
	return true
}

func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
