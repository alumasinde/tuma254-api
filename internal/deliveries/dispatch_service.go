package deliveries

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type DispatchService struct {
	repo     *Repository
	offers   *OfferRepository
	plans    *PlanRepository
	riders   RiderEligibility
	nearby   NearbyFinder
	offerTTL time.Duration
}

func NewDispatchService(r *Repository, o *OfferRepository, p *PlanRepository, re RiderEligibility, n NearbyFinder) *DispatchService {
	return &DispatchService{repo: r, offers: o, plans: p, riders: re, nearby: n, offerTTL: 45 * time.Second}
}

type NearbyRiderEligibility interface {
	CanAppearNearby(context.Context, string) (bool, error)
}

func (s *DispatchService) canReceiveDispatch(ctx context.Context, riderID string) (bool, error) {
	if v, ok := s.riders.(NearbyRiderEligibility); ok {
		return v.CanAppearNearby(ctx, riderID)
	}
	return s.riders.CanPublishLocation(ctx, riderID)
}

func (s *DispatchService) Offer(ctx context.Context, deliveryID, riderID, senderID string) (AssignmentOffer, error) {
	did, err := bson.ObjectIDFromHex(deliveryID)
	if err != nil {
		return AssignmentOffer{}, ErrNotFound
	}
	rid, err := bson.ObjectIDFromHex(riderID)
	if err != nil {
		return AssignmentOffer{}, ErrOffer
	}
	d, err := s.repo.Find(ctx, did)
	if err != nil {
		return AssignmentOffer{}, err
	}
	sid, err := bson.ObjectIDFromHex(senderID)
	if err != nil || d.Sender.UserID != sid {
		return AssignmentOffer{}, ErrForbidden
	}
	if d.Status != StatusRequested {
		return AssignmentOffer{}, ErrState
	}
	ok, err := s.canReceiveDispatch(ctx, rid.Hex())
	if err != nil {
		return AssignmentOffer{}, err
	}
	if !ok {
		return AssignmentOffer{}, ErrForbidden
	}

	now := time.Now().UTC()
	o := AssignmentOffer{
		ID:         bson.NewObjectID(),
		DeliveryID: did,
		RiderID:    rid,
		Status:     StatusOffered,
		CreatedAt:  now,
		ExpiresAt:  now.Add(s.offerTTL),
	}
	if err := s.offers.Create(ctx, o); err != nil {
		return AssignmentOffer{}, err
	}
	return o, nil
}

func (s *DispatchService) Accept(ctx context.Context, offerID, riderID string) (Public, error) {
	oid, err := bson.ObjectIDFromHex(offerID)
	if err != nil {
		return Public{}, ErrOffer
	}
	rid, err := bson.ObjectIDFromHex(riderID)
	if err != nil {
		return Public{}, ErrForbidden
	}
	o, err := s.offers.Find(ctx, oid)
	if err != nil {
		return Public{}, err
	}

	now := time.Now().UTC()
	if now.After(o.ExpiresAt) {
		return Public{}, ErrOfferExpired
	}
	if o.RiderID != rid || o.Status != StatusOffered {
		return Public{}, ErrOffer
	}
	ok, err := s.canReceiveDispatch(ctx, rid.Hex())
	if err != nil {
		return Public{}, err
	}
	if !ok {
		return Public{}, ErrForbidden
	}

	// Delivery assignment is the authoritative winner claim. Two concurrent
	// riders can race here, but only one can atomically move requested -> assigned.
	d, err := s.repo.Find(ctx, o.DeliveryID)
	if err != nil {
		return Public{}, err
	}
	_ = d
	ok, err = s.repo.Transition(
		ctx,
		o.DeliveryID,
		StatusRequested,
		StatusAssigned,
		bson.M{"riderId": rid, "updatedAt": now},
		Event{DeliveryID: o.DeliveryID, Type: "rider_assignment_accepted", ActorID: rid, At: now},
	)
	if err != nil {
		return Public{}, err
	}
	if !ok {
		// Another rider already won. Do not mutate this offer after the delivery
		// assignment has been lost.
		return Public{}, ErrState
	}

	// Consume the accepted offer only after the delivery claim succeeds. The
	// delivery state remains authoritative under concurrent acceptance.
	_, _ = s.offers.Respond(ctx, oid, rid, StatusOffered, StatusAssigned, now)

	if plan, err := s.plans.FindByDelivery(ctx, o.DeliveryID); err == nil {
		_, _ = s.plans.ClearActiveOffer(ctx, plan.ID, oid, now)
		_ = s.plans.Complete(ctx, plan.ID, now)
	}

	d.Status = StatusAssigned
	d.RiderID = &rid
	d.UpdatedAt = now
	return public(d), nil
}

func (s *DispatchService) Decline(ctx context.Context, offerID, riderID string) error {
	oid, err := bson.ObjectIDFromHex(offerID)
	if err != nil {
		return ErrOffer
	}
	rid, err := bson.ObjectIDFromHex(riderID)
	if err != nil {
		return ErrForbidden
	}
	o, err := s.offers.Find(ctx, oid)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	ok, err := s.offers.Respond(ctx, oid, rid, StatusOffered, StatusDeclined, now)
	if err != nil {
		return err
	}
	if !ok {
		return ErrOffer
	}
	return s.advanceAfterOffer(ctx, o, oid, now)
}

func (s *DispatchService) ExpireAndAdvance(ctx context.Context, offerID string) error {
	oid, err := bson.ObjectIDFromHex(offerID)
	if err != nil {
		return ErrOffer
	}
	o, err := s.offers.Find(ctx, oid)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if now.Before(o.ExpiresAt) {
		return ErrOffer
	}
	ok, err := s.offers.Respond(ctx, oid, o.RiderID, StatusOffered, StatusExpired, now)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return s.advanceAfterOffer(ctx, o, oid, now)
}

func (s *DispatchService) advanceAfterOffer(ctx context.Context, o AssignmentOffer, offerID bson.ObjectID, now time.Time) error {
	d, err := s.repo.Find(ctx, o.DeliveryID)
	if err != nil {
		return err
	}
	plan, err := s.plans.FindByDelivery(ctx, o.DeliveryID)
	if err != nil {
		return err
	}

	if d.Status != StatusRequested {
		_, _ = s.plans.ClearActiveOffer(ctx, plan.ID, offerID, now)
		_ = s.plans.Complete(ctx, plan.ID, now)
		return nil
	}

	cleared, err := s.plans.ClearActiveOffer(ctx, plan.ID, offerID, now)
	if err != nil {
		return err
	}
	if !cleared {
		return nil
	}
	_, err = s.advance(ctx, &plan)
	if errors.Is(err, ErrNoNearbyRiders) {
		return nil
	}
	return err
}

func (s *DispatchService) SweepExpired(ctx context.Context, limit int) error {
	values, err := s.offers.Expired(ctx, time.Now().UTC(), limit)
	if err != nil {
		return err
	}
	for _, v := range values {
		if err := s.ExpireAndAdvance(ctx, v.ID.Hex()); err != nil {
			return err
		}
	}
	return nil
}
