package migrations
import("context";"go.mongodb.org/mongo-driver/v2/bson";"go.mongodb.org/mongo-driver/v2/mongo")
type DispatchPlans struct{};func(DispatchPlans)Version()int{return 8};func(DispatchPlans)Name()string{return "delivery_dispatch_plans"}
func(DispatchPlans)Up(ctx context.Context,db *mongo.Database)error{_,e:=db.Collection("delivery_dispatch_plans").Indexes().CreateMany(ctx,[]mongo.IndexModel{{Keys:bson.D{{Key:"deliveryId",Value:1}},Options:&mongo.IndexOptions{Unique:true}},{Keys:bson.D{{Key:"status",Value:1},{Key:"updatedAt",Value:1}}}});return e}
