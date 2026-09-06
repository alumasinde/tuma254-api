package locations

import (
 "context"
 "errors"
 "time"
 "go.mongodb.org/mongo-driver/v2/bson"
 "go.mongodb.org/mongo-driver/v2/mongo"
 "go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrNotFound = errors.New("location not found")
type Repository struct{ db *mongo.Database }
func NewRepository(db *mongo.Database)*Repository{return &Repository{db:db}}

func (r *Repository) UpsertRiderLocation(ctx context.Context, value RiderLocation)(RiderLocation,error){
 var out RiderLocation
 err:=r.db.Collection("rider_locations").FindOneAndUpdate(ctx,bson.M{"riderId":value.RiderID},bson.M{"$set":bson.M{"location":value.Location,"accuracyMeters":value.AccuracyMeters,"headingDegrees":value.HeadingDegrees,"speedMps":value.SpeedMPS,"updatedAt":value.UpdatedAt}},options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&out)
 return out,err
}
func (r *Repository) FindRiderLocation(ctx context.Context,id bson.ObjectID)(RiderLocation,error){
 var out RiderLocation;err:=r.db.Collection("rider_locations").FindOne(ctx,bson.M{"riderId":id}).Decode(&out);if errors.Is(err,mongo.ErrNoDocuments){return out,ErrNotFound};return out,err
}
func (r *Repository) Nearby(ctx context.Context, point Point, maxDistance float64, limit int, freshAfter time.Time)([]RiderLocation,error){
 filter:=bson.M{"updatedAt":bson.M{"$gte":freshAfter},"location":bson.M{"$near":bson.M{"$geometry":point,"$maxDistance":maxDistance}}}
 cur,err:=r.db.Collection("rider_locations").Find(ctx,filter,options.Find().SetLimit(int64(limit)));if err!=nil{return nil,err};defer cur.Close(ctx)
 var out []RiderLocation;err=cur.All(ctx,&out);return out,err
}
func (r *Repository) CreatePlace(ctx context.Context,p SavedPlace)(SavedPlace,error){res,err:=r.db.Collection("saved_places").InsertOne(ctx,p);if err!=nil{return p,err};p.ID=res.InsertedID.(bson.ObjectID);return p,nil}
func (r *Repository) ListPlaces(ctx context.Context,userID bson.ObjectID)([]SavedPlace,error){cur,err:=r.db.Collection("saved_places").Find(ctx,bson.M{"userId":userID},options.Find().SetSort(bson.D{{Key:"updatedAt",Value:-1}}));if err!=nil{return nil,err};defer cur.Close(ctx);var out []SavedPlace;err=cur.All(ctx,&out);return out,err}
func (r *Repository) DeletePlace(ctx context.Context,userID,id bson.ObjectID)error{res,err:=r.db.Collection("saved_places").DeleteOne(ctx,bson.M{"_id":id,"userId":userID});if err!=nil{return err};if res.DeletedCount==0{return ErrNotFound};return nil}
