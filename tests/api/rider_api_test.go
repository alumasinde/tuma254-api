package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alumasinde/tuma254-api/internal/database/migrations"
	"github.com/alumasinde/tuma254-api/internal/identity"
	"github.com/alumasinde/tuma254-api/internal/riders"
	httpserver "github.com/alumasinde/tuma254-api/internal/platform/http"
	"github.com/alumasinde/tuma254-api/testkit"
)

func TestRiderAPIApplicationLifecycle(t *testing.T) {
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("MONGODB_TEST_URI not configured")
	}

	ctx := context.Background()
	db := testkit.MongoDatabase(t, uri, "tuma254_rider_api_test")
	if err := migrations.Run(ctx, db, migrations.All()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := identity.NewRepository(db)
	now := time.Now().UTC()

	riderUser, err := repo.CreateUser(ctx, identity.User{
		Email: "rider-api@example.com", FirstName: "API", LastName: "Rider",
		PasswordHash: "hash", Status: "active", CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create rider user: %v", err)
	}
	adminUser, err := repo.CreateUser(ctx, identity.User{
		Email: "admin-api@example.com", FirstName: "API", LastName: "Admin",
		PasswordHash: "hash", Status: "active", CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	if err := repo.AssignRole(ctx, adminUser.ID, "operations_admin"); err != nil {
		t.Fatalf("assign admin role: %v", err)
	}

	riderSession, err := repo.CreateSession(ctx, identity.Session{
		UserID: riderUser.ID, FamilyID: "rider-family", TokenHash: identity.HashOpaqueToken("rider-refresh"),
		DeviceID: "rider-device", CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("create rider session: %v", err)
	}
	adminSession, err := repo.CreateSession(ctx, identity.Session{
		UserID: adminUser.ID, FamilyID: "admin-family", TokenHash: identity.HashOpaqueToken("admin-refresh"),
		DeviceID: "admin-device", CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}

	tokens := identity.NewTokenManager("01234567890123456789012345678901", time.Hour)
	riderAccess, _, err := tokens.IssueAccessToken(riderUser.ID.Hex(), riderSession.ID.Hex())
	if err != nil {
		t.Fatalf("issue rider token: %v", err)
	}
	adminAccess, _, err := tokens.IssueAccessToken(adminUser.ID.Hex(), adminSession.ID.Hex())
	if err != nil {
		t.Fatalf("issue admin token: %v", err)
	}

	auth := identity.NewHandler(identity.NewService(repo, tokens, time.Hour))
	riderService := riders.NewService(riders.NewRepository(db), repo)
	handler := httpserver.NewHandler(func() error { return nil }, auth, riders.NewHandler(riderService, auth))

	request := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var raw bytes.Buffer
		if body != nil {
			if err := json.NewEncoder(&raw).Encode(body); err != nil {
				t.Fatalf("encode body: %v", err)
			}
		}
		req := httptest.NewRequest(method, path, &raw)
		req.Header.Set("Authorization", "Bearer "+token)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res
	}

	if res := request(http.MethodPost, "/api/v1/rider/application", riderAccess, nil); res.Code != http.StatusCreated {
		t.Fatalf("create application: %d %s", res.Code, res.Body.String())
	}
	if res := request(http.MethodPost, "/api/v1/rider/vehicles", riderAccess, map[string]string{
		"type": "motorcycle", "registration_number": "KDB 123B",
	}); res.Code != http.StatusCreated {
		t.Fatalf("add vehicle: %d %s", res.Code, res.Body.String())
	}
	if res := request(http.MethodPost, "/api/v1/rider/application/submit", riderAccess, nil); res.Code != http.StatusOK {
		t.Fatalf("submit application: %d %s", res.Code, res.Body.String())
	}
	if res := request(http.MethodPost, "/api/v1/operations/riders/"+riderUser.ID.Hex()+"/approve", adminAccess, nil); res.Code != http.StatusOK {
		t.Fatalf("approve rider: %d %s", res.Code, res.Body.String())
	}
	if res := request(http.MethodPut, "/api/v1/rider/availability", riderAccess, map[string]string{
		"availability": "available",
	}); res.Code != http.StatusOK {
		t.Fatalf("set availability: %d %s", res.Code, res.Body.String())
	}
}
