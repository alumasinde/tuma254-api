package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alumasinde/tuma254-api/internal/database/migrations"
	"github.com/alumasinde/tuma254-api/internal/deliveries"
	"github.com/alumasinde/tuma254-api/testkit"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestTransitionWritesStateAndEventAtomically(t *testing.T) {
	uri:=os.Getenv("MONGODB_TEST_URI"); if uri=="" { t.Skip("MONGODB_TEST_URI not configured") }
	db:=testkit.MongoTransactionDatabase(t,uri,"tuma254_transition_atomic_"+bson.NewObjectID().Hex())
	if err:=migrations.Run(context.Background(),db,migrations.All()...); err!=nil { t.Fatal(err) }
	repo:=deliveries.NewRepository(db)
	id:=bson.NewObjectID(); actor:=bson.NewObjectID(); now:=time.Now().UTC()
	if err:=repo.Create(context.Background(),deliveries.Delivery{ID:id,Sender:deliveries.Party{UserID:actor},Status:deliveries.StatusRequested,CreatedAt:now,UpdatedAt:now}); err!=nil { t.Fatal(err) }
	ok,err:=repo.Transition(context.Background(),id,deliveries.StatusRequested,deliveries.StatusAssigned,bson.M{"riderId":actor,"updatedAt":now},deliveries.Event{ID:bson.NewObjectID(),DeliveryID:id,Type:"rider_assigned",ActorID:actor,At:now})
	if err!=nil||!ok { t.Fatalf("transition failed: ok=%v err=%v",ok,err) }
	stored,err:=repo.Find(context.Background(),id); if err!=nil { t.Fatal(err) }
	if stored.Status!=deliveries.StatusAssigned { t.Fatalf("unexpected status: %s",stored.Status) }
	count,err:=db.Collection("delivery_events").CountDocuments(context.Background(),bson.M{"deliveryId":id}); if err!=nil { t.Fatal(err) }
	if count!=1 { t.Fatalf("expected one lifecycle event, got %d",count) }
}
