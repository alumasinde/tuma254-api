package favourites
import("context";"errors";"go.mongodb.org/mongo-driver/v2/bson";"go.mongodb.org/mongo-driver/v2/mongo";"go.mongodb.org/mongo-driver/v2/mongo/options")
var ErrNotFound=errors.New("favourite rider not found")
type Repository struct{db *mongo.Database};func NewRepository(db *mongo.Database)*Repository{return &Repository{db}}
func(r *Repository)Add(ctx context.Context,v Rider)(Rider,error){_,e:=r.db.Collection("favourite_riders").UpdateOne(ctx,bson.M{"senderId":v.SenderID,"riderId":v.RiderID},bson.M{"$setOnInsert":v},options.UpdateOne().SetUpsert(true));if e!=nil{return v,e};return v,nil}
func(r *Repository)Has(ctx context.Context,sender,rider bson.ObjectID)(bool,error){n,e:=r.db.Collection("favourite_riders").CountDocuments(ctx,bson.M{"senderId":sender,"riderId":rider});return n>0,e}
func(r *Repository)Remove(ctx context.Context,sender,rider bson.ObjectID)error{res,e:=r.db.Collection("favourite_riders").DeleteOne(ctx,bson.M{"senderId":sender,"riderId":rider});if e!=nil{return e};if res.DeletedCount==0{return ErrNotFound};return nil}
func(r *Repository)List(ctx context.Context,sender bson.ObjectID)([]Rider,error){cur,e:=r.db.Collection("favourite_riders").Find(ctx,bson.M{"senderId":sender});if e!=nil{return nil,e};defer cur.Close(ctx);var out []Rider;e=cur.All(ctx,&out);return out,e}
