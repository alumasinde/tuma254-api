package handlers

import (
	"net/http"

	"github.com/alumasinde/tuma254-api/internal/identity/dtos"
	"github.com/alumasinde/tuma254-api/internal/platform/httpctx"
	"github.com/alumasinde/tuma254-api/internal/platform/httpx"
)

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	u, err := h.svc.Me(r.Context(), userID.String())
	if err != nil || !u.Active {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, dtos.MeResponse{
		ID: u.ID.String(), Email: u.Email, Phone: u.Phone,
		PhoneVerified: u.PhoneVerifiedAt != nil,
		FirstName: u.FirstName, LastName: u.LastName, Roles: u.Roles,
	})
}
