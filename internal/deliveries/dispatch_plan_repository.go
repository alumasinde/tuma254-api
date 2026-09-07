package deliveries

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type PlanRepository struct{ db *mongo.Database }

func NewPlanRepository(db *mongo.Database) *PlanRepository { return &PlanRepository{db} }

func (r *PlanRepository) Create(ctx context.Context, p DispatchPlan) error {
	_, err := r.db.Collection("delivery_dispatch_plans").InsertOne(ctx, p)
	return err
}

func (r *PlanRepository) FindByDelivery(ctx context.Context, id bson.ObjectID) (DispatchPlan, error) {
	var p DispatchPlan
	err := r.db.Collection("delivery_dispatch_plans").FindOne(ctx, bson.M{"deliveryId": id, "status": "active"}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return p, ErrOfferNotFound
	}
	return p, err
}

func (r *PlanRepository) Advance(ctx context.Context, id bson.ObjectID, from, to int) (bool, error) {
	res, err := r.db.Collection("delivery_dispatch_plans").UpdateOne(
		ctx,
		bson.M{"_id": id, "status": "active", "nextIndex": from, "activeOfferId": nil},
		bson.M{"$set": bson.M{"nextIndex": to, "updatedAt": time.Now().UTC()}},
	)
	return res.ModifiedCount == 1, err
}

func (r *PlanRepository) SetActiveOffer(ctx context.Context, id, offer bson.ObjectID, now time.Time) (bool, error) {
	res, err := r.db.Collection("delivery_dispatch_plans").UpdateOne(
		ctx,
		bson.M{"_id": id, "status": "active", "activeOfferId": nil},
		bson.M{"$set": bson.M{"activeOfferId": offer, "updatedAt": now}},
	)
	return res.ModifiedCount == 1, err
}

func (r *PlanRepository) ClearActiveOffer(ctx context.Context, id, offer bson.ObjectID, now time.Time) (bool, error) {
	res, err := r.db.Collection("delivery_dispatch_plans").UpdateOne(
		ctx,
		bson.M{"_id": id, "activeOfferId": offer, "status": "active"},
		bson.M{"$set": bson.M{"activeOfferId": nil, "updatedAt": now}},
	)
	return res.ModifiedCount == 1, err
}

func (r *PlanRepository) Complete(ctx context.Context, id bson.ObjectID, now time.Time) error {
	_, err := r.db.Collection("delivery_dispatch_plans").UpdateOne(
		ctx,
		bson.M{"_id": id, "status": "active"},
		bson.M{"$set": bson.M{"status": "exhausted", "activeOfferId": nil, "updatedAt": now}},
	)
	return err
}
