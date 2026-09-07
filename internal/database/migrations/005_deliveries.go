package migrations
import("context";"go.mongodb.org/mongo-driver/v2/bson";"go.mongodb.org/mongo-driver/v2/mongo")
type Deliveries struct{};func(Deliveries)Version()int{return 5};func(Deliveries)Name()string{return "deliveries"}
func(Deliveries)Up(ctx context.Context,db *mongo.Database)error{for _,x:=range []struct{n string;m []mongo.IndexModel}{{"deliveries",[]mongo.IndexModel{{Keys:bson.D{{Key:"sender.userId",Value:1},{Key:"createdAt",Value:-1}}},{Keys:bson.D{{Key:"riderId",Value:1},{Key:"status",Value:1},{Key:"updatedAt",Value:-1}}}}},{"delivery_events",[]mongo.IndexModel{{Keys:bson.D{{Key:"deliveryId",Value:1},{Key:"at",Value:1}}}}}}{if _,e:=db.Collection(x.n).Indexes().CreateMany(ctx,x.m);e!=nil{return e}};return nil}
