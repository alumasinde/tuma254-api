package riders

import (
	"net/http"

	identityrepo "github.com/alumasinde/tuma254-api/internal/identity/repositories"
	"github.com/alumasinde/tuma254-api/internal/riders/handlers"
	"github.com/alumasinde/tuma254-api/internal/riders/repositories"
	"github.com/alumasinde/tuma254-api/internal/riders/services"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Authenticator interface {
	RequireAuth(http.Handler) http.Handler
}

func RegisterRoutes(mux *http.ServeMux, db *pgxpool.Pool, auth Authenticator, users identityrepo.UsersRepository, roles services.RoleAssigner) {
	svc := services.New(repositories.New(db), users, roles)
	h := handlers.New(svc)
	protect := func(pattern string, next http.HandlerFunc) { mux.Handle(pattern, auth.RequireAuth(next)) }
	protect("GET /api/v1/riders/me", h.GetMe)
	protect("POST /api/v1/riders/me/application", h.Create)
	protect("POST /api/v1/riders/me/application/submit", h.Submit)
	protect("PATCH /api/v1/riders/me/availability", h.SetAvailability)
	protect("POST /api/v1/riders/me/vehicles", h.AddVehicle)
	protect("GET /api/v1/riders/me/vehicles", h.ListVehicles)
	protect("PUT /api/v1/riders/me/vehicles/{vehicleID}/active", h.SetActiveVehicle)
}
