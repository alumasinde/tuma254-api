package identity

import (
	"net/http"

	"github.com/alumasinde/tuma254-api/internal/platform/auth"
	"github.com/alumasinde/tuma254-api/internal/routes"
)

func (h *Handler) Routes(api *routes.API, validator *auth.Validator) {
	api.HandleV1Func("/auth/register", h.Register)
	api.HandleV1Func("/auth/login", h.Login)
	api.HandleV1Func("/auth/refresh", h.Refresh)
	api.HandleV1Func("/auth/logout", h.Logout)
	api.HandleV1("/auth/me", validator.Authenticate(http.HandlerFunc(h.Me)))
}
