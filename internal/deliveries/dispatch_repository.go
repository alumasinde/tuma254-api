package deliveries
import("context";"errors";"time";"go.mongodb.org/mongo-driver/v2/bson";"go.mongodb.org/mongo-driver/v2/mongo")
var ErrOfferNotFound=errors.New("assignment offer not found")
type OfferRepository struct{db *mongo.Database};func NewOfferRepository(db *mongo.Database)*OfferRepository{return &OfferRepository{db}}
func(r *OfferRepository)Create(ctx context.Context,o AssignmentOffer)error{_,e:=r.db.Collection("delivery_assignment_offers").InsertOne(ctx,o);return e}
func(r *OfferRepository)Find(ctx context.Context,id bson.ObjectID)(AssignmentOffer,error){var o AssignmentOffer;e:=r.db.Collection("delivery_assignment_offers").FindOne(ctx,bson.M{"_id":id}).Decode(&o);if errors.Is(e,mongo.ErrNoDocuments){e=ErrOfferNotFound};return o,e}
func(r *OfferRepository)Respond(ctx context.Context,id,rider bson.ObjectID,from,to string,now time.Time)(bool,error){res,e:=r.db.Collection("delivery_assignment_offers").UpdateOne(ctx,bson.M{"_id":id,"riderId":rider,"status":from,"expiresAt":bson.M{"$gt":now}},bson.M{"$set":bson.M{"status":to,"respondedAt":now}});return res.ModifiedCount==1,e}
