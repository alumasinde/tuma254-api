package httpserver

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	h := NewHandler(func() error { return nil })
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/health", nil))
	if r.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", r.Code)
	}
}

func TestHealthFailure(t *testing.T) {
	h := NewHandler(func() error { return errors.New("down") })
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/health", nil))
	if r.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", r.Code)
	}
}
