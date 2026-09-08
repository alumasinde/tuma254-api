package models

type DeliveryStatus string

const (
	StatusDraft               DeliveryStatus = "draft"
	StatusRequested           DeliveryStatus = "requested"
	StatusAssigned            DeliveryStatus = "assigned"
	StatusAwaitingPickupOTP   DeliveryStatus = "awaiting_pickup_otp"
	StatusPickedUp            DeliveryStatus = "picked_up"
	StatusInTransit           DeliveryStatus = "in_transit"
	StatusAwaitingDeliveryOTP DeliveryStatus = "awaiting_delivery_otp"
	StatusCompleted           DeliveryStatus = "completed"
	StatusFailed              DeliveryStatus = "failed"
	StatusCancelled           DeliveryStatus = "cancelled"
)

func (s DeliveryStatus) Terminal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusCancelled
}
