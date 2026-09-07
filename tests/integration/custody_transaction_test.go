package integration

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alumasinde/tuma254-api/internal/database/migrations"
	"github.com/alumasinde/tuma254-api/internal/deliveries"
	"github.com/alumasinde/tuma254-api/internal/locations"
	"github.com/alumasinde/tuma254-api/testkit"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestTransactionalCustodyConcurrentClaim(t *testing.T) {
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("MONGODB_TEST_URI not configured")
	}

	db := testkit.MongoTransactionDatabase(t, uri, "tuma254_custody_concurrency_test_"+bson.NewObjectID().Hex())
	if err := migrations.Run(context.Background(), db, migrations.All()...); err != nil {
		t.Fatal(err)
	}

	repo := deliveries.NewRepository(db)
	id := bson.NewObjectID()
	rid := bson.NewObjectID()
	now := time.Now().UTC()
	otp := deliveries.OTP{Hash: "same", ExpiresAt: now.Add(time.Minute), MaxAttempts: 5, Version: 1}
	d := deliveries.Delivery{
		ID:        id,
		Status:    deliveries.StatusAwaitingPickupOTP,
		RiderID:   &rid,
		PickupOTP: &otp,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Create(context.Background(), d); err != nil {
		t.Fatal(err)
	}

	run := func() bool {
		ok, err := repo.TransitionWithCustody(
			context.Background(),
			id,
			deliveries.StatusAwaitingPickupOTP,
			deliveries.StatusPickedUp,
			"pickupOtp",
			"same",
			bson.M{"updatedAt": now},
			deliveries.Event{ID: bson.NewObjectID(), DeliveryID: id, Type: "pickup_custody_transferred", ActorID: rid, At: now},
			deliveries.CustodyEvent{
				ID:         bson.NewObjectID(),
				DeliveryID: id,
				Stage:      "pickup",
				FromParty:  "sender",
				ToParty:    "rider",
				ActorID:    rid,
				Location:   locations.Point{Type: "Point", Coordinates: [2]float64{36.8, -1.2}},
				At:         now,
			},
		)
		if err != nil {
			t.Errorf("transition: %v", err)
		}
		return ok
	}

	const attempts = 32
	var wg sync.WaitGroup
	wins := make(chan bool, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wins <- run()
		}()
	}
	wg.Wait()
	close(wins)

	count := 0
	for ok := range wins {
		if ok {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one winner, got %d", count)
	}

	stored, err := repo.Find(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != deliveries.StatusPickedUp {
		t.Fatalf("expected picked_up status, got %s", stored.Status)
	}
	if stored.PickupOTP == nil || stored.PickupOTP.VerifiedAt == nil {
		t.Fatal("expected pickup OTP to be atomically consumed")
	}

	events, err := db.Collection("delivery_events").CountDocuments(context.Background(), bson.M{"deliveryId": id})
	if err != nil {
		t.Fatal(err)
	}
	custody, err := db.Collection("delivery_custody_events").CountDocuments(context.Background(), bson.M{"deliveryId": id})
	if err != nil {
		t.Fatal(err)
	}
	if events != 1 || custody != 1 {
		t.Fatalf("expected one event and one custody event, got %d/%d", events, custody)
	}
}
