package httpserver

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLoggingAddsRequestID(t *testing.T) {
	h := WithRequestLogging(slog.Default(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFromContext(r.Context()) == "" {
			t.Fatal("request ID missing from context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("request ID missing from response")
	}
}
