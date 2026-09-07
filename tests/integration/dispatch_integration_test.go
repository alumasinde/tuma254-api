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

type dispatchEligibility struct{mu sync.Mutex; calls map[string]int; answers map[string][]bool}
func (f *dispatchEligibility) next(id string)(bool,error){f.mu.Lock();defer f.mu.Unlock();i:=f.calls[id];f.calls[id]=i+1;v:=f.answers[id];if len(v)==0{return true,nil};if i>=len(v){return v[len(v)-1],nil};return v[i],nil}
func(f *dispatchEligibility)CanPublishLocation(_ context.Context,id string)(bool,error){return f.next(id)}
func(f *dispatchEligibility)CanAppearNearby(_ context.Context,id string)(bool,error){return f.next(id)}
type dispatchNearby struct{values []locations.PublicRiderLocation}
func(f dispatchNearby)Nearby(context.Context,float64,float64,float64,int)([]locations.PublicRiderLocation,error){return f.values,nil}

func dispatchDB(t *testing.T)*testDispatch{
 t.Helper();uri:=os.Getenv("MONGODB_TEST_URI");if uri==""{t.Skip("MONGODB_TEST_URI not configured")}
 db:=testkit.MongoDatabase(t,uri,"tuma254_dispatch_test_"+bson.NewObjectID().Hex())
 if err:=migrations.Run(context.Background(),db,migrations.All()...);err!=nil{t.Fatalf("migrate: %v",err)}
 return &testDispatch{repo:deliveries.NewRepository(db),offers:deliveries.NewOfferRepository(db),plans:deliveries.NewPlanRepository(db)}
}
type testDispatch struct{repo *deliveries.Repository;offers *deliveries.OfferRepository;plans *deliveries.PlanRepository}
func seedDelivery(t *testing.T,r *deliveries.Repository,id,sender bson.ObjectID)deliveries.Delivery{
 t.Helper();now:=time.Now().UTC();d:=deliveries.Delivery{ID:id,Sender:deliveries.Party{UserID:sender},Method:deliveries.MethodNearby,Status:deliveries.StatusRequested,Pickup:deliveries.Stop{Address:"Pickup",ContactName:"Sender",Phone:"0710000000",Location:locations.Point{Type:"Point",Coordinates:[2]float64{36.8219,-1.2921}}},Recipient:deliveries.Stop{Address:"Drop",ContactName:"Receiver",Phone:"0720000000",Location:locations.Point{Type:"Point",Coordinates:[2]float64{36.8300,-1.3000}}},Package:deliveries.Package{Description:"Parcel",WeightKG:1},CreatedAt:now,UpdatedAt:now};if err:=r.Create(context.Background(),d);err!=nil{t.Fatal(err)};return d
}

func TestDispatchConcurrentAcceptOnlyOneWins(t *testing.T){
 x:=dispatchDB(t);ctx:=context.Background();sender:=bson.NewObjectID();d:=seedDelivery(t,x.repo,bson.NewObjectID(),sender);r1,r2:=bson.NewObjectID(),bson.NewObjectID();now:=time.Now().UTC()
 o1:=deliveries.AssignmentOffer{ID:bson.NewObjectID(),DeliveryID:d.ID,RiderID:r1,Status:deliveries.StatusOffered,CreatedAt:now,ExpiresAt:now.Add(time.Minute)}
 o2:=deliveries.AssignmentOffer{ID:bson.NewObjectID(),DeliveryID:d.ID,RiderID:r2,Status:deliveries.StatusOffered,CreatedAt:now,ExpiresAt:now.Add(time.Minute)}
 if err:=x.offers.Create(ctx,o1);err!=nil{t.Fatal(err)};if err:=x.offers.Create(ctx,o2);err!=nil{t.Fatal(err)}
 elig:=&dispatchEligibility{calls:map[string]int{},answers:map[string][]bool{r1.Hex():{true},r2.Hex():{true}}};s:=deliveries.NewDispatchService(x.repo,x.offers,x.plans,elig,dispatchNearby{})
 var wg sync.WaitGroup;wg.Add(2);results:=make(chan error,2)
 for _,v:=range []struct{o,r bson.ObjectID}{{o1.ID,r1},{o2.ID,r2}}{go func(v struct{o,r bson.ObjectID}){defer wg.Done();_,e:=s.Accept(ctx,v.o.Hex(),v.r.Hex());results<-e}(v)}
 wg.Wait();close(results);wins:=0;for e:=range results{if e==nil{wins++}};if wins!=1{t.Fatalf("expected exactly one winner, got %d",wins)}
 stored,e:=x.repo.Find(ctx,d.ID);if e!=nil{t.Fatal(e)};if stored.Status!=deliveries.StatusAssigned||stored.RiderID==nil{t.Fatalf("delivery not assigned correctly: %+v",stored)}
}

func TestDispatchDeclineSkipsRiderWhoBecomesUnavailable(t *testing.T){
 x:=dispatchDB(t);ctx:=context.Background();sender:=bson.NewObjectID();d:=seedDelivery(t,x.repo,bson.NewObjectID(),sender);r1,r2,r3:=bson.NewObjectID(),bson.NewObjectID(),bson.NewObjectID()
 elig:=&dispatchEligibility{calls:map[string]int{},answers:map[string][]bool{r1.Hex():{true,true},r2.Hex():{true,false},r3.Hex():{true,true}}}
 nearby:=dispatchNearby{values:[]locations.PublicRiderLocation{{RiderID:r1.Hex(),DistanceMeters:100},{RiderID:r2.Hex(),DistanceMeters:200},{RiderID:r3.Hex(),DistanceMeters:300}}}
 s:=deliveries.NewDispatchService(x.repo,x.offers,x.plans,elig,nearby);res,e:=s.StartNearby(ctx,d.ID.Hex(),sender.Hex(),deliveries.NearbyDispatchInput{});if e!=nil{t.Fatal(e)}
 if res.RiderID!=r1.Hex(){t.Fatalf("expected first offer rider1, got %s",res.RiderID)}
 if e:=s.Decline(ctx,res.OfferID,r1.Hex());e!=nil{t.Fatal(e)}
 plan,e:=x.plans.FindByDelivery(ctx,d.ID);if e!=nil{t.Fatal(e)};if plan.ActiveOfferID==nil{t.Fatal("expected next active offer")}
 offer,e:=x.offers.Find(ctx,*plan.ActiveOfferID);if e!=nil{t.Fatal(e)};if offer.RiderID!=r3{t.Fatalf("expected unavailable rider2 skipped and rider3 offered, got %s",offer.RiderID.Hex())}
}

func TestDispatchExpiryRaceAdvancesOnce(t *testing.T){
 x:=dispatchDB(t);ctx:=context.Background();sender:=bson.NewObjectID();d:=seedDelivery(t,x.repo,bson.NewObjectID(),sender);r1,r2:=bson.NewObjectID(),bson.NewObjectID();now:=time.Now().UTC()
 elig:=&dispatchEligibility{calls:map[string]int{},answers:map[string][]bool{r1.Hex():{true},r2.Hex():{true}}};s:=deliveries.NewDispatchService(x.repo,x.offers,x.plans,elig,dispatchNearby{})
 o:=deliveries.AssignmentOffer{ID:bson.NewObjectID(),DeliveryID:d.ID,RiderID:r1,Status:deliveries.StatusOffered,CreatedAt:now.Add(-time.Minute),ExpiresAt:now.Add(-time.Second)}
 if err:=x.offers.Create(ctx,o);err!=nil{t.Fatal(err)}
 p:=deliveries.DispatchPlan{ID:bson.NewObjectID(),DeliveryID:d.ID,Candidates:[]deliveries.Candidate{{RiderID:r2}},NextIndex:0,ActiveOfferID:&o.ID,Status:"active",CreatedAt:now,UpdatedAt:now};if err:=x.plans.Create(ctx,p);err!=nil{t.Fatal(err)}
 var wg sync.WaitGroup;wg.Add(2);errs:=make(chan error,2);for i:=0;i<2;i++{go func(){defer wg.Done();errs<-s.ExpireAndAdvance(ctx,o.ID.Hex())}()};wg.Wait();close(errs)
 nextCount:=0;for e:=range errs{if e==nil{nextCount++}};if nextCount==0{t.Fatal("expected expiry processing")}
 plan,e:=x.plans.FindByDelivery(ctx,d.ID);if e!=nil{t.Fatal(e)};if plan.ActiveOfferID==nil{t.Fatal("expected one next offer")}
 offer,e:=x.offers.Find(ctx,*plan.ActiveOfferID);if e!=nil{t.Fatal(e)};if offer.RiderID!=r2{t.Fatalf("expected rider2, got %s",offer.RiderID.Hex())}
}

func TestDispatchAcceptanceStopsExpiredOfferReassignment(t *testing.T){
 x:=dispatchDB(t);ctx:=context.Background();sender:=bson.NewObjectID();d:=seedDelivery(t,x.repo,bson.NewObjectID(),sender);r1,r2:=bson.NewObjectID(),bson.NewObjectID();now:=time.Now().UTC()
 elig:=&dispatchEligibility{calls:map[string]int{},answers:map[string][]bool{r1.Hex():{true},r2.Hex():{true}}};s:=deliveries.NewDispatchService(x.repo,x.offers,x.plans,elig,dispatchNearby{})
 o:=deliveries.AssignmentOffer{ID:bson.NewObjectID(),DeliveryID:d.ID,RiderID:r1,Status:deliveries.StatusOffered,CreatedAt:now.Add(-time.Minute),ExpiresAt:now.Add(time.Minute)}
 if err:=x.offers.Create(ctx,o);err!=nil{t.Fatal(err)}
 if _,e:=s.Accept(ctx,o.ID.Hex(),r1.Hex());e!=nil{t.Fatal(e)}
 // Simulate a stale expiry worker observing the old offer after the delivery is already assigned.
 if e:=s.ExpireAndAdvance(ctx,o.ID.Hex());e==nil{t.Fatal("expected non-expired offer to be rejected")}
 stored,e:=x.repo.Find(ctx,d.ID);if e!=nil{t.Fatal(e)};if stored.RiderID==nil||*stored.RiderID!=r1{t.Fatal("assignment changed unexpectedly")}
 _=r2
}
