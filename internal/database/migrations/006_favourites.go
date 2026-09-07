package migrations
import("context";"go.mongodb.org/mongo-driver/v2/bson";"go.mongodb.org/mongo-driver/v2/mongo")
type Favourites struct{};func(Favourites)Version()int{return 6};func(Favourites)Name()string{return "favourite_riders"}
func(Favourites)Up(ctx context.Context,db *mongo.Database)error{_,e:=db.Collection("favourite_riders").Indexes().CreateOne(ctx,mongo.IndexModel{Keys:bson.D{{Key:"senderId",Value:1},{Key:"riderId",Value:1}},Options:&mongo.IndexOptions{Unique:true}});return e}
