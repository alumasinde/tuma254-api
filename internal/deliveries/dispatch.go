package deliveries
import("context";"errors";"time";"go.mongodb.org/mongo-driver/v2/bson")
const(StatusOffered="offered";StatusDeclined="declined";StatusExpired="expired")
var(ErrOffer=errors.New("invalid assignment offer");ErrOfferExpired=errors.New("assignment offer expired"))
type AssignmentOffer struct{ID bson.ObjectID `bson:"_id,omitempty"`;DeliveryID bson.ObjectID `bson:"deliveryId"`;RiderID bson.ObjectID `bson:"riderId"`;Status string `bson:"status"`;ExpiresAt time.Time `bson:"expiresAt"`;CreatedAt time.Time `bson:"createdAt"`;RespondedAt *time.Time `bson:"respondedAt,omitempty"`}
type DispatchRepository struct{db interface{}}
