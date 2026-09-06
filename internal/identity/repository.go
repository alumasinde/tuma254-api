package identity

import (
	"context"
	"errors"
	"strings"
	"time"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrNotFound = errors.New("not found")

type Repository struct { db *mongo.Database }
func NewRepository(db *mongo.Database) *Repository { return &Repository{db:db} }

func (r *Repository) CreateUser(ctx context.Context, user User) (User,error) {
	result,err:=r.db.Collection("users").InsertOne(ctx,user)
	if err!=nil { return User{},err }
	user.ID=result.InsertedID.(bson.ObjectID)
	return user,nil
}

func (r *Repository) FindUserByIdentifier(ctx context.Context, identifier string) (User,error) {
	identifier=strings.TrimSpace(identifier)
	var user User
	err:=r.db.Collection("users").FindOne(ctx,bson.M{"$or":bson.A{bson.M{"email":strings.ToLower(identifier)},bson.M{"phone":identifier}}}).Decode(&user)
	if errors.Is(err,mongo.ErrNoDocuments) { return User{},ErrNotFound }
	return user,err
}

func (r *Repository) FindUserByID(ctx context.Context,id bson.ObjectID)(User,error){
	var user User
	err:=r.db.Collection("users").FindOne(ctx,bson.M{"_id":id}).Decode(&user)
	if errors.Is(err,mongo.ErrNoDocuments){return User{},ErrNotFound}
	return user,err
}

func (r *Repository) AssignRole(ctx context.Context,userID bson.ObjectID,roleCode string) error {
	_,err:=r.db.Collection("user_roles").UpdateOne(ctx,bson.M{"userId":userID,"roleCode":roleCode},bson.M{"$setOnInsert":bson.M{"userId":userID,"roleCode":roleCode,"assignedAt":time.Now().UTC()}},options.UpdateOne().SetUpsert(true))
	return err
}

func (r *Repository) CreateSession(ctx context.Context,session Session)(Session,error){
	result,err:=r.db.Collection("sessions").InsertOne(ctx,session)
	if err!=nil{return Session{},err}
	session.ID=result.InsertedID.(bson.ObjectID)
	return session,nil
}

func (r *Repository) FindSessionByTokenHash(ctx context.Context,hash string)(Session,error){
	var session Session
	err:=r.db.Collection("sessions").FindOne(ctx,bson.M{"tokenHash":hash}).Decode(&session)
	if errors.Is(err,mongo.ErrNoDocuments){return Session{},ErrNotFound}
	return session,err
}

func (r *Repository) FindSessionByID(ctx context.Context,id bson.ObjectID)(Session,error){
	var session Session
	err:=r.db.Collection("sessions").FindOne(ctx,bson.M{"_id":id}).Decode(&session)
	if errors.Is(err,mongo.ErrNoDocuments){return Session{},ErrNotFound}
	return session,err
}

func (r *Repository) RevokeSession(ctx context.Context,id bson.ObjectID,reason string) error {
	now:=time.Now().UTC()
	_,err:=r.db.Collection("sessions").UpdateOne(ctx,bson.M{"_id":id,"revokedAt":bson.M{"$exists":false}},bson.M{"$set":bson.M{"revokedAt":now,"revokedReason":reason}})
	return err
}

func (r *Repository) RotateSession(ctx context.Context,oldID bson.ObjectID,next Session)(Session,bool,error){
	now:=time.Now().UTC()
	result,err:=r.db.Collection("sessions").UpdateOne(ctx,bson.M{"_id":oldID,"revokedAt":bson.M{"$exists":false}},bson.M{"$set":bson.M{"revokedAt":now,"revokedReason":"rotated"}})
	if err!=nil{return Session{},false,err}
	if result.ModifiedCount!=1{return Session{},false,nil}
	next.CreatedAt=now
	created,err:=r.CreateSession(ctx,next)
	return created,true,err
}

func (r *Repository) RevokeFamily(ctx context.Context,familyID,reason string) error {
	now:=time.Now().UTC()
	_,err:=r.db.Collection("sessions").UpdateMany(ctx,bson.M{"familyId":familyID,"revokedAt":bson.M{"$exists":false}},bson.M{"$set":bson.M{"revokedAt":now,"revokedReason":reason}})
	return err
}

func (r *Repository) RecordAuthEvent(ctx context.Context,event AuthEvent){_,_=r.db.Collection("auth_events").InsertOne(ctx,event)}

func (r *Repository) UserPermissions(ctx context.Context,userID bson.ObjectID)([]string,error){
	cursor,err:=r.db.Collection("user_roles").Aggregate(ctx,mongo.Pipeline{
		{{Key:"$match",Value:bson.M{"userId":userID}}},
		{{Key:"$lookup",Value:bson.M{"from":"role_permissions","localField":"roleCode","foreignField":"roleCode","as":"permissions"}}},
		{{Key:"$unwind",Value:"$permissions"}},
		{{Key:"$project",Value:bson.M{"_id":0,"permission":"$permissions.permissionCode"}}},
	})
	if err!=nil{return nil,err}
	defer cursor.Close(ctx)
	var rows []struct{Permission string `bson:"permission"`}
	if err:=cursor.All(ctx,&rows);err!=nil{return nil,err}
	out:=make([]string,0,len(rows))
	for _,row:=range rows{out=append(out,row.Permission)}
	return out,nil
}
