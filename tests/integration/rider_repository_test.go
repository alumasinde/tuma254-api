package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alumasinde/tuma254-api/internal/database/migrations"
	"github.com/alumasinde/tuma254-api/internal/identity"
	"github.com/alumasinde/tuma254-api/internal/riders"
	"github.com/alumasinde/tuma254-api/testkit"
)

func TestRiderApplicationLifecycle(t *testing.T) {
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("MONGODB_TEST_URI not configured")
	}
	db := testkit.MongoDatabase(t, uri, "tuma254_rider_repository_test")
	if err := migrations.Run(context.Background(), db, migrations.All()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	identityRepo := identity.NewRepository(db)
	now := time.Now().UTC()
	user, err := identityRepo.CreateUser(context.Background(), identity.User{
		Email: "rider-lifecycle@example.com", FirstName: "Test", LastName: "Rider",
		PasswordHash: "hash", Status: "active", CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	service := riders.NewService(riders.NewRepository(db), identityRepo)
	profile, err := service.CreateApplication(context.Background(), user.ID.Hex())
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	if profile.VerificationStatus != riders.VerificationDraft {
		t.Fatalf("expected draft, got %s", profile.VerificationStatus)
	}
	if _, err := service.AddVehicle(context.Background(), user.ID.Hex(), riders.Vehicle{Type: "motorcycle", RegistrationNumber: "KDA 123A"}); err != nil {
		t.Fatalf("add vehicle: %v", err)
	}
	profile, err = service.SubmitApplication(context.Background(), user.ID.Hex())
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if profile.VerificationStatus != riders.VerificationSubmitted {
		t.Fatalf("expected submitted, got %s", profile.VerificationStatus)
	}
	profile, err = service.Approve(context.Background(), user.ID.Hex())
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if profile.VerificationStatus != riders.VerificationApproved {
		t.Fatalf("expected approved, got %s", profile.VerificationStatus)
	}
	profile, err = service.SetAvailability(context.Background(), user.ID.Hex(), riders.AvailabilityAvailable)
	if err != nil {
		t.Fatalf("availability: %v", err)
	}
	if profile.Availability != riders.AvailabilityAvailable {
		t.Fatalf("expected available, got %s", profile.Availability)
	}
}
