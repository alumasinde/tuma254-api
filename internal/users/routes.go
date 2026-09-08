package users

import (
	"context"
	"net/http"

	"github.com/alumasinde/tuma254-api/internal/users/handlers"
	"github.com/alumasinde/tuma254-api/internal/users/repositories"
	"github.com/alumasinde/tuma254-api/internal/users/services"
	identityrepo "github.com/alumasinde/tuma254-api/internal/identity/repositories"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Authenticator interface{ RequireAuth(http.Handler) http.Handler }
func RegisterRoutes(mux *http.ServeMux,db *pgxpool.Pool,auth Authenticator,users identityrepo.UsersRepository){svc:=services.New(repositories.New(db),users);h:=handlers.New(svc);mux.Handle("GET /api/v1/users/me/profile",auth.RequireAuth(http.HandlerFunc(h.Get)));mux.Handle("PUT /api/v1/users/me/profile",auth.RequireAuth(http.HandlerFunc(h.Update)))}
var _ = context.Background
