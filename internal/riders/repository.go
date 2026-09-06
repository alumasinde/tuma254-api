package riders

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrNotFound = errors.New("not found")

type Repository struct{ db *mongo.Database }

func NewRepository(db *mongo.Database) *Repository { return &Repository{db: db} }

func (r *Repository) FindProfile(ctx context.Context, userID bson.ObjectID) (Profile, error) {
	var profile Profile
	err := r.db.Collection("rider_profiles").FindOne(ctx, bson.M{"userId": userID}).Decode(&profile)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Profile{}, ErrNotFound
	}
	return profile, err
}

func (r *Repository) CreateProfile(ctx context.Context, profile Profile) (Profile, error) {
	result, err := r.db.Collection("rider_profiles").InsertOne(ctx, profile)
	if err != nil {
		return Profile{}, err
	}
	profile.ID = result.InsertedID.(bson.ObjectID)
	return profile, nil
}

func (r *Repository) UpdateProfile(ctx context.Context, userID bson.ObjectID, set bson.M) (Profile, error) {
	set["updatedAt"] = time.Now().UTC()
	var profile Profile
	err := r.db.Collection("rider_profiles").FindOneAndUpdate(
		ctx,
		bson.M{"userId": userID},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&profile)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Profile{}, ErrNotFound
	}
	return profile, err
}

func (r *Repository) CreateVehicle(ctx context.Context, vehicle Vehicle) (Vehicle, error) {
	result, err := r.db.Collection("rider_vehicles").InsertOne(ctx, vehicle)
	if err != nil {
		return Vehicle{}, err
	}
	vehicle.ID = result.InsertedID.(bson.ObjectID)
	return vehicle, nil
}

func (r *Repository) ListVehicles(ctx context.Context, riderID bson.ObjectID) ([]Vehicle, error) {
	cursor, err := r.db.Collection("rider_vehicles").Find(ctx, bson.M{"riderId": riderID}, options.Find().SetSort(bson.D{{Key: "active", Value: -1}, {Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var vehicles []Vehicle
	if err := cursor.All(ctx, &vehicles); err != nil {
		return nil, err
	}
	return vehicles, nil
}

func (r *Repository) HasActiveVehicle(ctx context.Context, riderID bson.ObjectID) (bool, error) {
	count, err := r.db.Collection("rider_vehicles").CountDocuments(ctx, bson.M{"riderId": riderID, "active": true})
	return count > 0, err
}

func (r *Repository) SetActiveVehicle(ctx context.Context, riderID, vehicleID bson.ObjectID) (Vehicle, error) {
	if _, err := r.db.Collection("rider_vehicles").UpdateMany(ctx, bson.M{"riderId": riderID, "active": true}, bson.M{"$set": bson.M{"active": false, "updatedAt": time.Now().UTC()}}); err != nil {
		return Vehicle{}, err
	}
	now := time.Now().UTC()
	var vehicle Vehicle
	err := r.db.Collection("rider_vehicles").FindOneAndUpdate(
		ctx,
		bson.M{"_id": vehicleID, "riderId": riderID},
		bson.M{"$set": bson.M{"active": true, "updatedAt": now}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&vehicle)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Vehicle{}, ErrNotFound
	}
	return vehicle, err
}
