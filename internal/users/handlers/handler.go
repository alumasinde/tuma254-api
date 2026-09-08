package handlers

import (
	"net/http"
	"strings"

	"github.com/alumasinde/tuma254-api/internal/platform/httpctx"
	"github.com/alumasinde/tuma254-api/internal/platform/httpx"
	"github.com/alumasinde/tuma254-api/internal/users/services"
)

type Handler struct{ svc *services.Service }

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpctx.UserID(r.Context())
	if !ok { httpx.Error(w, http.StatusUnauthorized, "unauthorized"); return }
	profile, err := h.svc.Get(r.Context(), userID)
	if err != nil { httpx.Error(w, http.StatusNotFound, "profile not found"); return }
	httpx.WriteJSON(w, http.StatusOK, profile)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpctx.UserID(r.Context())
	if !ok { httpx.Error(w, http.StatusUnauthorized, "unauthorized"); return }
	var input struct{ AvatarURL string `json:"avatar_url"` }
	if err := httpx.DecodeJSON(w, r, &input); err != nil { httpx.Error(w, http.StatusBadRequest, "invalid request"); return }
	profile, err := h.svc.Update(r.Context(), userID, strings.TrimSpace(input.AvatarURL))
	if err != nil { httpx.Error(w, http.StatusBadRequest, "profile update failed"); return }
	httpx.WriteJSON(w, http.StatusOK, profile)
}
