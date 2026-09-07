package deliveries

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var ErrOfferNotFound = errors.New("assignment offer not found")

type OfferRepository struct{ db *mongo.Database }

func NewOfferRepository(db *mongo.Database) *OfferRepository { return &OfferRepository{db} }

func (r *OfferRepository) Create(ctx context.Context, o AssignmentOffer) error {
	_, err := r.db.Collection("delivery_assignment_offers").InsertOne(ctx, o)
	return err
}

func (r *OfferRepository) Find(ctx context.Context, id bson.ObjectID) (AssignmentOffer, error) {
	var o AssignmentOffer
	err := r.db.Collection("delivery_assignment_offers").FindOne(ctx, bson.M{"_id": id}).Decode(&o)
	if errors.Is(err, mongo.ErrNoDocuments) {
		err = ErrOfferNotFound
	}
	return o, err
}

func (r *OfferRepository) Expired(ctx context.Context, now time.Time, limit int) ([]AssignmentOffer, error) {
	cur, err := r.db.Collection("delivery_assignment_offers").Find(ctx, bson.M{
		"status":    StatusOffered,
		"expiresAt": bson.M{"$lte": now},
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []AssignmentOffer
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Respond atomically consumes an offer exactly once. Expiry is part of the
// filter so a late response can never mutate an already-expired offer.
func (r *OfferRepository) Respond(ctx context.Context, id, rider bson.ObjectID, from, to string, now time.Time) (bool, error) {
	res, err := r.db.Collection("delivery_assignment_offers").UpdateOne(
		ctx,
		bson.M{
			"_id":       id,
			"riderId":   rider,
			"status":    from,
			"expiresAt": bson.M{"$gt": now},
		},
		bson.M{"$set": bson.M{"status": to, "respondedAt": now}},
	)
	if err != nil {
		return false, err
	}
	return res.ModifiedCount == 1, nil
}
