package deliveries

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/alumasinde/tuma254-api/internal/locations"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var ErrNoNearbyRiders = errors.New("no eligible nearby riders")

type NearbyFinder interface {
	Nearby(context.Context, float64, float64, float64, int) ([]locations.PublicRiderLocation, error)
}

type Candidate struct {
	RiderID         bson.ObjectID `bson:"riderId"`
	DistanceMeters  float64       `bson:"distanceMeters"`
	ActiveDeliveries int64        `bson:"activeDeliveries"`
	Score           float64       `bson:"score"`
}

type DispatchPlan struct {
	ID            bson.ObjectID `bson:"_id,omitempty"`
	DeliveryID    bson.ObjectID `bson:"deliveryId"`
	Candidates    []Candidate   `bson:"candidates"`
	NextIndex     int           `bson:"nextIndex"`
	ActiveOfferID *bson.ObjectID `bson:"activeOfferId,omitempty"`
	Status        string        `bson:"status"`
	CreatedAt     time.Time     `bson:"createdAt"`
	UpdatedAt     time.Time     `bson:"updatedAt"`
}

type NearbyDispatchInput struct {
	RadiusMeters   float64 `json:"radius_meters,omitempty"`
	CandidateLimit int     `json:"candidate_limit,omitempty"`
}

type DispatchResult struct {
	PlanID    string     `json:"plan_id"`
	OfferID   string     `json:"offer_id,omitempty"`
	RiderID   string     `json:"rider_id,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Candidates int       `json:"candidates"`
}

func (s *DispatchService) StartNearby(ctx context.Context, deliveryID, senderID string, in NearbyDispatchInput) (DispatchResult, error) {
	did, err := bson.ObjectIDFromHex(deliveryID)
	if err != nil { return DispatchResult{}, ErrNotFound }
	d, err := s.repo.Find(ctx, did)
	if err != nil { return DispatchResult{}, err }
	sid, err := bson.ObjectIDFromHex(senderID)
	if err != nil || d.Sender.UserID != sid { return DispatchResult{}, ErrForbidden }
	if d.Method != MethodNearby || d.Status != StatusRequested { return DispatchResult{}, ErrState }

	radius := in.RadiusMeters
	if radius <= 0 { radius = 5000 }
	limit := in.CandidateLimit
	if limit <= 0 { limit = 12 }
	if limit > 50 { limit = 50 }

	near, err := s.nearby.Nearby(ctx, d.Pickup.Location.Coordinates[1], d.Pickup.Location.Coordinates[0], radius, limit)
	if err != nil { return DispatchResult{}, err }

	candidates := make([]Candidate, 0, len(near))
	for _, n := range near {
		rid, err := bson.ObjectIDFromHex(n.RiderID)
		if err != nil { continue }
		ok, err := s.canReceiveDispatch(ctx, n.RiderID)
		if err != nil || !ok { continue }
		load, err := s.repo.CountActiveForRider(ctx, rid)
		if err != nil { return DispatchResult{}, err }
		candidates = append(candidates, rankCandidate(rid, n.DistanceMeters, load))
	}
	if len(candidates) == 0 { return DispatchResult{}, ErrNoNearbyRiders }
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Score < candidates[j].Score })

	now := time.Now().UTC()
	plan := DispatchPlan{ID: bson.NewObjectID(), DeliveryID: did, Candidates: candidates, Status: "active", CreatedAt: now, UpdatedAt: now}
	if err := s.plans.Create(ctx, plan); err != nil { return DispatchResult{}, err }
	offer, err := s.advance(ctx, &plan)
	if err != nil { return DispatchResult{}, err }
	return DispatchResult{PlanID: plan.ID.Hex(), OfferID: offer.ID.Hex(), RiderID: offer.RiderID.Hex(), ExpiresAt: &offer.ExpiresAt, Candidates: len(candidates)}, nil
}

func rankCandidate(id bson.ObjectID, distance float64, load int64) Candidate {
	return Candidate{RiderID: id, DistanceMeters: distance, ActiveDeliveries: load, Score: distance + float64(load)*5000}
}

func (s *DispatchService) advance(ctx context.Context, plan *DispatchPlan) (AssignmentOffer, error) {
	for plan.NextIndex < len(plan.Candidates) {
		candidate := plan.Candidates[plan.NextIndex]
		from := plan.NextIndex
		next := from + 1
		ok, err := s.plans.Advance(ctx, plan.ID, from, next)
		if err != nil { return AssignmentOffer{}, err }
		if !ok { return AssignmentOffer{}, ErrOffer }
		plan.NextIndex = next

		ok, err = s.canReceiveDispatch(ctx, candidate.RiderID.Hex())
		if err != nil { return AssignmentOffer{}, err }
		if !ok { continue }

		now := time.Now().UTC()
		offer := AssignmentOffer{ID: bson.NewObjectID(), DeliveryID: plan.DeliveryID, RiderID: candidate.RiderID, Status: StatusOffered, CreatedAt: now, ExpiresAt: now.Add(s.offerTTL)}
		if err := s.offers.Create(ctx, offer); err != nil { return AssignmentOffer{}, err }
		ok, err = s.plans.SetActiveOffer(ctx, plan.ID, offer.ID, now)
		if err != nil { return AssignmentOffer{}, err }
		if !ok {
			// Another concurrent dispatcher already populated the active offer.
			// The unique live-offer index prevents a second live offer per delivery.
			return AssignmentOffer{}, ErrOffer
		}
		plan.ActiveOfferID = &offer.ID
		return offer, nil
	}
	_ = s.plans.Complete(ctx, plan.ID, time.Now().UTC())
	return AssignmentOffer{}, ErrNoNearbyRiders
}
