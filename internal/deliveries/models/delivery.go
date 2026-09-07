package models

import "time"

type Point struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

type Stop struct {
	Address     string `json:"address"`
	ContactName string `json:"contact_name"`
	Phone       string `json:"phone"`
	Location    Point  `json:"location"`
}

type Package struct {
	Description string  `json:"description"`
	WeightKG    float64 `json:"weight_kg"`
}

type Delivery struct {
	ID           string
	SenderUserID string
	Pickup       Stop
	Recipient    Stop
	Package      Package
	Method       string
	RiderID      *string
	Status       DeliveryStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
