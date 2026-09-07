package deliveries

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrNotFound = errors.New("delivery not found")

type Repository struct{ db *mongo.Database }

func NewRepository(db *mongo.Database) *Repository { return &Repository{db} }

func (r *Repository) Create(ctx context.Context, d Delivery) error {
	_, err := r.db.Collection("deliveries").InsertOne(ctx, d)
	return err
}

func (r *Repository) Find(ctx context.Context, id bson.ObjectID) (Delivery, error) {
	var d Delivery
	err := r.db.Collection("deliveries").FindOne(ctx, bson.M{"_id": id}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		err = ErrNotFound
	}
	return d, err
}

func (r *Repository) Transition(ctx context.Context, id bson.ObjectID, from, to string, set bson.M, e Event) (bool, error) {
	set["status"] = to
	session, err := r.db.Client().StartSession()
	if err != nil { return false, err }
	defer session.EndSession(ctx)

	claimed := false
	_, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
		res, err := r.db.Collection("deliveries").UpdateOne(sc, bson.M{"_id": id, "status": from}, bson.M{"$set": set})
		if err != nil { return nil, err }
		if res.ModifiedCount != 1 { return nil, nil }
		claimed = true
		if _, err := r.db.Collection("delivery_events").InsertOne(sc, e); err != nil { return nil, err }
		return nil, nil
	})
	if err != nil { return false, err }
	return claimed, nil
}

// ClaimAssignment is the single-writer concurrency gate for rider assignment.
// Exactly one caller can move a requested delivery into assigned state.
func (r *Repository) ClaimAssignment(ctx context.Context, id, rider bson.ObjectID, now time.Time) (bool, error) {
	session, err := r.db.Client().StartSession()
	if err != nil { return false, err }
	defer session.EndSession(ctx)

	claimed := false
	_, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
		res, err := r.db.Collection("deliveries").UpdateOne(sc,
			bson.M{"_id": id, "status": StatusRequested},
			bson.M{"$set": bson.M{"riderId": rider, "status": StatusAssigned, "updatedAt": now}},
		)
		if err != nil { return nil, err }
		if res.ModifiedCount != 1 { return nil, nil }
		claimed = true
		_, err = r.db.Collection("delivery_events").InsertOne(sc, Event{ID:bson.NewObjectID(),DeliveryID:id,Type:"rider_assignment_accepted",ActorID:rider,At:now})
		return nil, err
	})
	if err != nil { return false, err }
	return claimed, nil
}

func (r *Repository) CountActiveForRider(ctx context.Context, id bson.ObjectID) (int64, error) {
	return r.db.Collection("deliveries").CountDocuments(ctx, bson.M{
		"riderId": id,
		"status": bson.M{"$in": []string{StatusAssigned, StatusAwaitingPickupOTP, StatusPickedUp, StatusInTransit, StatusAwaitingDeliveryOTP}},
	})
}

func (r *Repository) ListForUser(ctx context.Context, id bson.ObjectID) ([]Delivery, error) {
	c, err := r.db.Collection("deliveries").Find(ctx, bson.M{"sender.userId": id}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer c.Close(ctx)
	var out []Delivery
	err = c.All(ctx, &out)
	return out, err
}

func (r *Repository) RecordOTPFailure(ctx context.Context, id bson.ObjectID, field string, expected string, now time.Time, max int) (bool, bool, error) {
	path := field + ".attempts"
	lock := field + ".lockedAt"
	filter := bson.M{"_id": id, "status": expected, field + ".verifiedAt": bson.M{"$exists": false}, field + ".lockedAt": bson.M{"$exists": false}}
	update := bson.M{"$inc": bson.M{path: 1}, "$set": bson.M{"updatedAt": now}}
	res, err := r.db.Collection("deliveries").UpdateOne(ctx, filter, update)
	if err != nil || res.ModifiedCount == 0 {
		return false, false, err
	}
	var d Delivery
	if err := r.db.Collection("deliveries").FindOne(ctx, bson.M{"_id": id}).Decode(&d); err != nil {
		return true, false, err
	}
	var o *OTP
	if field == "pickupOtp" {
		o = d.PickupOTP
	} else {
		o = d.DeliveryOTP
	}
	locked := o != nil && o.Attempts >= max
	if locked {
		_, err = r.db.Collection("deliveries").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{lock: now, "updatedAt": now}})
	}
	return true, locked, err
}

// TransitionWithCustody atomically consumes the OTP, advances the delivery,
// writes its lifecycle event, and writes the custody handover event. It must
// run against a replica set member or mongos because it uses a MongoDB
// multi-document transaction.
func (r *Repository) TransitionWithCustody(ctx context.Context, id bson.ObjectID, from, to, otpField, otpHash string, set bson.M, e Event, c CustodyEvent) (bool, error) {
	set["status"] = to
	set[otpField+".verifiedAt"] = e.At
	filter := bson.M{
		"_id":                    id,
		"status":                 from,
		otpField + ".hash":       otpHash,
		otpField + ".verifiedAt": bson.M{"$exists": false},
		otpField + ".lockedAt":   bson.M{"$exists": false},
		otpField + ".expiresAt":  bson.M{"$gt": e.At},
	}

	session, err := r.db.Client().StartSession()
	if err != nil {
		return false, err
	}
	defer session.EndSession(ctx)

	claimed := false
	_, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
		res, err := r.db.Collection("deliveries").UpdateOne(sc, filter, bson.M{"$set": set})
		if err != nil {
			return nil, err
		}
		if res.ModifiedCount != 1 {
			return nil, nil
		}
		claimed = true
		if _, err := r.db.Collection("delivery_events").InsertOne(sc, e); err != nil {
			return nil, err
		}
		if _, err := r.db.Collection("delivery_custody_events").InsertOne(sc, c); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		return false, err
	}
	return claimed, nil
}

func (r *Repository) CreateIncident(ctx context.Context, i Incident) error {
	_, err := r.db.Collection("delivery_incidents").InsertOne(ctx, i)
	return err
}

func (r *Repository) FailWithIncident(ctx context.Context, id bson.ObjectID, from, to string, set bson.M, e Event, incident Incident) (bool, error) {
	set["status"] = to
	session, err := r.db.Client().StartSession()
	if err != nil { return false, err }
	defer session.EndSession(ctx)
	claimed := false
	_, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
		res, err := r.db.Collection("deliveries").UpdateOne(sc, bson.M{"_id":id,"status":from}, bson.M{"$set":set})
		if err != nil { return nil, err }
		if res.ModifiedCount != 1 { return nil, nil }
		claimed = true
		if _, err := r.db.Collection("delivery_events").InsertOne(sc,e); err != nil { return nil, err }
		if _, err := r.db.Collection("delivery_incidents").InsertOne(sc,incident); err != nil { return nil, err }
		return nil,nil
	})
	if err != nil { return false, err }
	return claimed,nil
}
