package migrations

import (
 "context"
 "go.mongodb.org/mongo-driver/v2/bson"
 "go.mongodb.org/mongo-driver/v2/mongo"
 "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Locations struct{}
func(Locations)Version()int{return 4}
func(Locations)Name()string{return "locations"}
func(Locations)Up(ctx context.Context,db *mongo.Database)error{
 entries:=[]struct{collection string;models []mongo.IndexModel}{
  {"saved_places",[]mongo.IndexModel{
   {Keys:bson.D{{Key:"userId",Value:1},{Key:"updatedAt",Value:-1}}},
   {Keys:bson.D{{Key:"userId",Value:1},{Key:"label",Value:1}},Options:options.Index().SetUnique(true)},
   {Keys:bson.D{{Key:"location",Value:"2dsphere"}}},
  }},
  {"rider_locations",[]mongo.IndexModel{
   {Keys:bson.D{{Key:"riderId",Value:1}},Options:options.Index().SetUnique(true)},
   {Keys:bson.D{{Key:"location",Value:"2dsphere"}}},
   {Keys:bson.D{{Key:"updatedAt",Value:-1}}},
  }},
 }
 for _,e:=range entries{if _,err:=db.Collection(e.collection).Indexes().CreateMany(ctx,e.models);err!=nil{return err}}
 return nil
}
