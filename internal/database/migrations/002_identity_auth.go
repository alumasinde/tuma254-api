package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type IdentityAuth struct{}

func (IdentityAuth) Version() int { return 2 }
func (IdentityAuth) Name() string { return "identity_auth" }

func (IdentityAuth) Up(ctx context.Context, db *mongo.Database) error {
	indexes := []struct { collection string; models []mongo.IndexModel }{
		{"users", []mongo.IndexModel{
			{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
			{Keys: bson.D{{Key: "phone", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		}},
		{"roles", []mongo.IndexModel{{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)}}},
		{"permissions", []mongo.IndexModel{{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)}}},
		{"user_roles", []mongo.IndexModel{
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "roleCode", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "userId", Value: 1}}},
		}},
		{"role_permissions", []mongo.IndexModel{{Keys: bson.D{{Key: "roleCode", Value: 1}, {Key: "permissionCode", Value: 1}}, Options: options.Index().SetUnique(true)}}},
		{"sessions", []mongo.IndexModel{
			{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
			{Keys: bson.D{{Key: "familyId", Value: 1}}},
			{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		}},
		{"auth_events", []mongo.IndexModel{
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
			{Keys: bson.D{{Key: "createdAt", Value: 1}}},
		}},
	}
	for _, entry := range indexes {
		if _, err := db.Collection(entry.collection).Indexes().CreateMany(ctx, entry.models); err != nil { return err }
	}
	now := bson.M{"createdAt": bson.DateTime(0)}
	for _, role := range []bson.M{
		{"code":"sender","name":"Sender","description":"Can create and manage own deliveries"},
		{"code":"rider","name":"Rider","description":"Can perform assigned delivery work"},
		{"code":"operations_admin","name":"Operations Admin","description":"Operational administration"},
	} {
		if _, err := db.Collection("roles").UpdateOne(ctx, bson.M{"code":role["code"]}, bson.M{"$setOnInsert": merge(role, now)}, options.UpdateOne().SetUpsert(true)); err != nil { return err }
	}
	for _, permission := range []bson.M{
		{"code":"deliveries.create","description":"Create deliveries"},
		{"code":"deliveries.read.own","description":"Read own deliveries"},
		{"code":"rider.availability.update","description":"Update rider availability"},
		{"code":"operations.users.manage","description":"Manage users"},
	} {
		if _, err := db.Collection("permissions").UpdateOne(ctx, bson.M{"code":permission["code"]}, bson.M{"$setOnInsert": merge(permission, now)}, options.UpdateOne().SetUpsert(true)); err != nil { return err }
	}
	for _, a := range []struct{role,permission string}{
		{"sender","deliveries.create"},{"sender","deliveries.read.own"},{"rider","rider.availability.update"},{"operations_admin","operations.users.manage"},
	} {
		if _, err := db.Collection("role_permissions").UpdateOne(ctx, bson.M{"roleCode":a.role,"permissionCode":a.permission}, bson.M{"$setOnInsert":bson.M{"roleCode":a.role,"permissionCode":a.permission}}, options.UpdateOne().SetUpsert(true)); err != nil { return err }
	}
	return nil
}

func merge(left, right bson.M) bson.M {
	out := bson.M{}
	for k,v := range left { out[k]=v }
	for k,v := range right { out[k]=v }
	return out
}
