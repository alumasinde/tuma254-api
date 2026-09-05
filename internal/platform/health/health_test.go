package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLive(t *testing.T) {
	h := New(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Live(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Header().Get("Content-Type") == "" {
		t.Fatal("expected content type")
	}
}
