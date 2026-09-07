package identity

import (
 "net/http"
 "time"
 "github.com/alumasinde/tuma254-api/internal/identity/handlers"
 "github.com/alumasinde/tuma254-api/internal/identity/repositories"
 "github.com/alumasinde/tuma254-api/internal/identity/services"
 "github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(mux *http.ServeMux,db *pgxpool.Pool,secret string,access,refresh time.Duration){
 svc:=services.New(repositories.New(db),secret,access,refresh);h:=handlers.New(svc)
 mux.HandleFunc("POST /api/v1/auth/register",h.Register)
 mux.HandleFunc("POST /api/v1/auth/login",h.Login)
 mux.HandleFunc("POST /api/v1/auth/refresh",h.Refresh)
 mux.HandleFunc("POST /api/v1/auth/logout",h.Logout)
 mux.Handle("GET /api/v1/me",h.RequireAuth(http.HandlerFunc(h.Me)))
}
