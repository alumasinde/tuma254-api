package favourites
import("context";"time";"go.mongodb.org/mongo-driver/v2/bson")
type RiderEligibility interface{CanPublishLocation(context.Context,string)(bool,error)}
type Service struct{repo *Repository;riders RiderEligibility};func NewService(r *Repository,re RiderEligibility)*Service{return &Service{r,re}}
func(s *Service)Add(ctx context.Context,senderID,riderID string)(Public,error){sid,e:=bson.ObjectIDFromHex(senderID);if e!=nil{return Public{},ErrNotFound};rid,e:=bson.ObjectIDFromHex(riderID);if e!=nil{return Public{},ErrNotFound};if sid==rid{return Public{},ErrNotFound};ok,e:=s.riders.CanPublishLocation(ctx,rid.Hex());if e!=nil{return Public{},e};if !ok{return Public{},ErrNotFound};now:=time.Now().UTC();v,e:=s.repo.Add(ctx,Rider{SenderID:sid,RiderID:rid,CreatedAt:now,UpdatedAt:now});if e!=nil{return Public{},e};return Public{RiderID:v.RiderID.Hex(),CreatedAt:v.CreatedAt},nil}
func(s *Service)Has(ctx context.Context,senderID,riderID string)(bool,error){sid,e:=bson.ObjectIDFromHex(senderID);if e!=nil{return false,ErrNotFound};rid,e:=bson.ObjectIDFromHex(riderID);if e!=nil{return false,ErrNotFound};return s.repo.Has(ctx,sid,rid)}
