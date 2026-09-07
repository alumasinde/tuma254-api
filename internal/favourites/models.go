package favourites
import("time";"go.mongodb.org/mongo-driver/v2/bson")
type Rider struct{ID bson.ObjectID `bson:"_id,omitempty"`;SenderID bson.ObjectID `bson:"senderId"`;RiderID bson.ObjectID `bson:"riderId"`;CreatedAt time.Time `bson:"createdAt"`;UpdatedAt time.Time `bson:"updatedAt"`}
type Public struct{RiderID string `json:"rider_id"`;CreatedAt time.Time `json:"created_at"`}
