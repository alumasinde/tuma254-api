package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONStrictRejectsUnknownAndMultipleValues(t *testing.T) {
	var dst struct{ Name string `json:"name"` }
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok","extra":true}`))
	req.Header.Set("Content-Type", "application/json")
	if err := DecodeJSONStrict(httptest.NewRecorder(), req, &dst); err == nil {
		t.Fatal("expected unknown field rejection")
	}

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"} {"name":"second"}`))
	req.Header.Set("Content-Type", "application/json")
	if err := DecodeJSONStrict(httptest.NewRecorder(), req, &dst); err == nil {
		t.Fatal("expected multiple JSON value rejection")
	}
}

func TestDecodeJSONStrictRequiresJSONWhenContentTypeProvided(t *testing.T) {
	var dst struct{ Name string `json:"name"` }
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}`))
	req.Header.Set("Content-Type", "text/plain")
	if err := DecodeJSONStrict(httptest.NewRecorder(), req, &dst); err == nil {
		t.Fatal("expected content type rejection")
	}
}
