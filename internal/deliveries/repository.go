package deliveries
import("context";"errors";"go.mongodb.org/mongo-driver/v2/bson";"go.mongodb.org/mongo-driver/v2/mongo";"go.mongodb.org/mongo-driver/v2/mongo/options")
var ErrNotFound=errors.New("delivery not found")
type Repository struct{db *mongo.Database};func NewRepository(db *mongo.Database)*Repository{return &Repository{db}}
func(r *Repository)Create(ctx context.Context,d Delivery)error{_,e:=r.db.Collection("deliveries").InsertOne(ctx,d);return e}
func(r *Repository)Find(ctx context.Context,id bson.ObjectID)(Delivery,error){var d Delivery;e:=r.db.Collection("deliveries").FindOne(ctx,bson.M{"_id":id}).Decode(&d);if errors.Is(e,mongo.ErrNoDocuments){e=ErrNotFound};return d,e}
func(r *Repository)Transition(ctx context.Context,id bson.ObjectID,from,to string,set bson.M,e Event)(bool,error){set["status"]=to;res,err:=r.db.Collection("deliveries").UpdateOne(ctx,bson.M{"_id":id,"status":from},bson.M{"$set":set});if err!=nil{return false,err};if res.ModifiedCount==0{return false,nil};_,err=r.db.Collection("delivery_events").InsertOne(ctx,e);return true,err}
func(r *Repository)ListForUser(ctx context.Context,id bson.ObjectID)([]Delivery,error){c,e:=r.db.Collection("deliveries").Find(ctx,bson.M{"sender.userId":id},options.Find().SetSort(bson.D{{Key:"createdAt",Value:-1}}));if e!=nil{return nil,e};defer c.Close(ctx);var out []Delivery;e=c.All(ctx,&out);return out,e}
