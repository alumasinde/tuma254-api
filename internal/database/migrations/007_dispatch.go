package migrations
import("context";"go.mongodb.org/mongo-driver/v2/bson";"go.mongodb.org/mongo-driver/v2/mongo";"go.mongodb.org/mongo-driver/v2/mongo/options")
type Dispatch struct{};func(Dispatch)Version()int{return 7};func(Dispatch)Name()string{return "delivery_dispatch"}
func(Dispatch)Up(ctx context.Context,db *mongo.Database)error{_,e:=db.Collection("delivery_assignment_offers").Indexes().CreateMany(ctx,[]mongo.IndexModel{
 {Keys:bson.D{{Key:"deliveryId",Value:1},{Key:"status",Value:1}}},
 {Keys:bson.D{{Key:"riderId",Value:1},{Key:"status",Value:1},{Key:"expiresAt",Value:1}}},
 {Keys:bson.D{{Key:"deliveryId",Value:1}},Options: options.Index()Unique:true,PartialFilterExpression:bson.M{"status":"offered"}},
});return e}
