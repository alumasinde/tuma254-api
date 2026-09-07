package deliveries

import ("time";"github.com/alumasinde/tuma254-api/internal/locations";"go.mongodb.org/mongo-driver/v2/bson")
const (
StatusDraft="draft";StatusRequested="requested";StatusAssigned="assigned";StatusAwaitingPickupOTP="awaiting_pickup_otp";StatusPickedUp="picked_up";StatusInTransit="in_transit";StatusAwaitingDeliveryOTP="awaiting_delivery_otp";StatusCompleted="completed";StatusFailed="failed";StatusCancelled="cancelled"
MethodFavourite="favourite_rider";MethodNearby="nearby_rider"
)
type Party struct{UserID bson.ObjectID `bson:"userId"`;Name string `bson:"name"`;Phone string `bson:"phone,omitempty"`}
type Stop struct{Address string `bson:"address"`;Location locations.Point `bson:"location"`;ContactName string `bson:"contactName"`;Phone string `bson:"phone"`}
type Package struct{Description string `bson:"description"`;WeightKG float64 `bson:"weightKg"`}
type Delivery struct{ID bson.ObjectID `bson:"_id,omitempty"`;Sender Party `bson:"sender"`;Recipient Stop `bson:"recipient"`;Pickup Stop `bson:"pickup"`;Package Package `bson:"package"`;Method string `bson:"method"`;RiderID *bson.ObjectID `bson:"riderId,omitempty"`;Status string `bson:"status"`;PickupOTP *OTP `bson:"pickupOtp,omitempty"`;DeliveryOTP *OTP `bson:"deliveryOtp,omitempty"`;PickupEvidence *CustodyEvidence `bson:"pickupEvidence,omitempty"`;DeliveryEvidence *CustodyEvidence `bson:"deliveryEvidence,omitempty"`;FailureReason string `bson:"failureReason,omitempty"`;CreatedAt time.Time `bson:"createdAt"`;UpdatedAt time.Time `bson:"updatedAt"`}
type OTP struct{Hash string `bson:"hash"`;ExpiresAt time.Time `bson:"expiresAt"`;VerifiedAt *time.Time `bson:"verifiedAt,omitempty"`;Attempts int `bson:"attempts"`;MaxAttempts int `bson:"maxAttempts"`;ResendCount int `bson:"resendCount"`;LastSentAt time.Time `bson:"lastSentAt"`;LockedAt *time.Time `bson:"lockedAt,omitempty"`;Version int `bson:"version"`}
type CustodyEvidence struct{Kind string `bson:"kind"`;Reference string `bson:"reference,omitempty"`;Note string `bson:"note,omitempty"`;CapturedAt time.Time `bson:"capturedAt"`}
type Incident struct{ID bson.ObjectID `bson:"_id,omitempty"`;DeliveryID bson.ObjectID `bson:"deliveryId"`;Type string `bson:"type"`;Reason string `bson:"reason"`;ActorID bson.ObjectID `bson:"actorId"`;Status string `bson:"status"`;CreatedAt time.Time `bson:"createdAt"`}
type Event struct{ID bson.ObjectID `bson:"_id,omitempty"`;DeliveryID bson.ObjectID `bson:"deliveryId"`;Type string `bson:"type"`;ActorID bson.ObjectID `bson:"actorId"`;At time.Time `bson:"at"`;Meta map[string]string `bson:"meta,omitempty"`}
type CreateInput struct{Pickup Stop `json:"pickup"`;Recipient Stop `json:"recipient"`;Package Package `json:"package"`;Method string `json:"method"`;RiderID string `json:"rider_id,omitempty"`}
type Public struct{ID string `json:"id"`;Status string `json:"status"`;Method string `json:"method"`;RiderID string `json:"rider_id,omitempty"`;Pickup Stop `json:"pickup"`;Recipient Stop `json:"recipient"`;Package Package `json:"package"`;CreatedAt time.Time `json:"created_at"`;UpdatedAt time.Time `json:"updated_at"`}
