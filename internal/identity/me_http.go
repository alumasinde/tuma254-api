package identity

import (
	"errors"
	"net/http"

	"github.com/alumasinde/tuma254-api/internal/platform/auth"
)

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	principal, err := auth.PrincipalFromContext(r.Context())
	if err != nil {
		errJSON(w, http.StatusUnauthorized, "missing authenticated user")
		return
	}

	user, err := h.svc.CurrentUser(r.Context(), principal.UserID)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			errJSON(w, http.StatusUnauthorized, "user not found")
		default:
			errJSON(w, http.StatusInternalServerError, "failed to load user")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]User{"user": user})
}
