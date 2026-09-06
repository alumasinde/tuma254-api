package locations

import (
 "time"
 "go.mongodb.org/mongo-driver/v2/bson"
)

type Point struct { Type string `bson:"type" json:"type"`; Coordinates [2]float64 `bson:"coordinates" json:"coordinates"` }

type RiderLocation struct {
 ID bson.ObjectID `bson:"_id,omitempty"`
 RiderID bson.ObjectID `bson:"riderId"`
 Location Point `bson:"location"`
 AccuracyMeters float64 `bson:"accuracyMeters,omitempty"`
 HeadingDegrees float64 `bson:"headingDegrees,omitempty"`
 SpeedMPS float64 `bson:"speedMps,omitempty"`
 UpdatedAt time.Time `bson:"updatedAt"`
}

type SavedPlace struct {
 ID bson.ObjectID `bson:"_id,omitempty"`
 UserID bson.ObjectID `bson:"userId"`
 Label string `bson:"label"`
 ContactName string `bson:"contactName,omitempty"`
 PhoneNumber string `bson:"phoneNumber,omitempty"`
 Address string `bson:"address,omitempty"`
 Location Point `bson:"location"`
 CreatedAt time.Time `bson:"createdAt"`
 UpdatedAt time.Time `bson:"updatedAt"`
}

type UpdateRiderLocationInput struct { Latitude float64 `json:"latitude"`; Longitude float64 `json:"longitude"`; AccuracyMeters float64 `json:"accuracy_meters,omitempty"`; HeadingDegrees float64 `json:"heading_degrees,omitempty"`; SpeedMPS float64 `json:"speed_mps,omitempty"` }
type SavePlaceInput struct { Label string `json:"label"`; ContactName string `json:"contact_name,omitempty"`; PhoneNumber string `json:"phone_number,omitempty"`; Address string `json:"address,omitempty"`; Latitude float64 `json:"latitude"`; Longitude float64 `json:"longitude"` }

type PublicRiderLocation struct { RiderID string `json:"rider_id"`; Location Point `json:"location"`; AccuracyMeters float64 `json:"accuracy_meters,omitempty"`; HeadingDegrees float64 `json:"heading_degrees,omitempty"`; SpeedMPS float64 `json:"speed_mps,omitempty"`; UpdatedAt time.Time `json:"updated_at"`; DistanceMeters float64 `json:"distance_meters,omitempty"` }
type PublicSavedPlace struct { ID string `json:"id"`; Label string `json:"label"`; ContactName string `json:"contact_name,omitempty"`; PhoneNumber string `json:"phone_number,omitempty"`; Address string `json:"address,omitempty"`; Location Point `json:"location"`; CreatedAt time.Time `json:"created_at"`; UpdatedAt time.Time `json:"updated_at"` }
