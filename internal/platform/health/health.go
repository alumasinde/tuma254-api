package health

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

type Handler struct { db *pgxpool.Pool; redis *goredis.Client }
func New(db *pgxpool.Pool, redis *goredis.Client) *Handler { return &Handler{db: db, redis: redis} }

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := h.db.Ping(ctx); err != nil { http.Error(w, `{"status":"unhealthy"}`, http.StatusServiceUnavailable); return }
	if err := h.redis.Ping(ctx).Err(); err != nil { http.Error(w, `{"status":"unhealthy"}`, http.StatusServiceUnavailable); return }
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status":"healthy"})
}
