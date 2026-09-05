package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrincipalFromContextMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, err := PrincipalFromContext(req.Context()); err == nil {
		t.Fatal("expected missing principal error")
	}
}

func TestHasRole(t *testing.T) {
	p := Principal{UserID: "u1", Roles: []string{"customer", "seller"}}
	if !HasRole(p, "seller") {
		t.Fatal("expected seller role")
	}
	if HasRole(p, "admin") {
		t.Fatal("did not expect admin role")
	}
	if !HasAnyRole(p, "dispatcher", "customer") {
		t.Fatal("expected any-role match")
	}
}

func TestRequireRole(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithPrincipal(req.Context(), Principal{UserID: "u1", Roles: []string{"customer"}}))
	rec := httptest.NewRecorder()
	RequireRole("customer", h).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithPrincipal(req.Context(), Principal{UserID: "u1", Roles: []string{"customer"}}))
	rec = httptest.NewRecorder()
	RequireRole("admin", h).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
