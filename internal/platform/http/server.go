package httpserver

import (
	"encoding/json"
	"net/http"
)

func NewHandler(health func() error) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := health(); err != nil {
			WriteJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: "unhealthy"})
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		if err := health(); err != nil {
			WriteJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: "not ready"})
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	return mux
}

func DecodeJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}
