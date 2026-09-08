package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRejectsTrailingValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"a"} {"name":"b"}`))
	rec := httptest.NewRecorder()
	var dst struct {
		Name string `json:"name"`
	}
	if err := DecodeJSON(rec, req, &dst); err == nil {
		t.Fatal("expected trailing JSON value to fail")
	}
}

func TestWithRequestID(t *testing.T) {
	var got string
	h := WithRequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = RequestID(r.Context()) }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if got == "" || rec.Header().Get("X-Request-ID") != got {
		t.Fatal("expected request id in context and response")
	}
}
