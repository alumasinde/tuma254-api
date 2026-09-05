package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

type Handler struct {
	db    *pgxpool.Pool
	redis *goredis.Client
}

type response struct {
	Status  string            `json:"status"`
	Service string            `json:"service"`
	Checks  map[string]string `json:"checks,omitempty"`
}

func New(db *pgxpool.Pool, redis *goredis.Client) *Handler {
	return &Handler{db: db, redis: redis}
}

func (h *Handler) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, response{Status: "ok", Service: "tuma254-api"})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{"database": "ok", "redis": "ok"}
	if err := h.db.Ping(ctx); err != nil {
		checks["database"] = "error"
		writeJSON(w, http.StatusServiceUnavailable, response{Status: "unhealthy", Service: "tuma254-api", Checks: checks})
		return
	}
	if err := h.redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = "error"
		writeJSON(w, http.StatusServiceUnavailable, response{Status: "unhealthy", Service: "tuma254-api", Checks: checks})
		return
	}
	writeJSON(w, http.StatusOK, response{Status: "ok", Service: "tuma254-api", Checks: checks})
}

func writeJSON(w http.ResponseWriter, status int, body response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
