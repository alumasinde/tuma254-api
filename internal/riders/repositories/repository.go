package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/alumasinde/tuma254-api/internal/riders/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("rider not found")

type Repository struct{ db *pgxpool.Pool }
func New(db *pgxpool.Pool) *Repository { return &Repository{db:db} }

func (r *Repository) FindProfile(ctx context.Context, userID uuid.UUID) (models.Profile,error) {
	return r.scanProfile(r.db.QueryRow(ctx,`SELECT id,user_id,verification_status,availability,COALESCE(rejection_reason,''),created_at,updated_at FROM rider_profiles WHERE user_id=$1`,userID))
}

func (r *Repository) CreateProfile(ctx context.Context,userID uuid.UUID)(models.Profile,error){
	return r.scanProfile(r.db.QueryRow(ctx,`INSERT INTO rider_profiles(user_id) VALUES($1) RETURNING id,user_id,verification_status,availability,COALESCE(rejection_reason,''),created_at,updated_at`,userID))
}

func (r *Repository) UpdateProfile(ctx context.Context,userID uuid.UUID,status models.VerificationStatus,availability models.Availability,reason string)(models.Profile,error){
	return r.scanProfile(r.db.QueryRow(ctx,`UPDATE rider_profiles SET verification_status=$2,availability=$3,rejection_reason=NULLIF($4,''),updated_at=now() WHERE user_id=$1 RETURNING id,user_id,verification_status,availability,COALESCE(rejection_reason,''),created_at,updated_at`,userID,status,availability,reason))
}

func (r *Repository) scanProfile(row pgx.Row)(models.Profile,error){var p models.Profile;err:=row.Scan(&p.ID,&p.UserID,&p.VerificationStatus,&p.Availability,&p.RejectionReason,&p.CreatedAt,&p.UpdatedAt);if errors.Is(err,pgx.ErrNoRows){return models.Profile{},ErrNotFound};return p,err}

func (r *Repository) HasActiveVehicle(ctx context.Context,riderID uuid.UUID)(bool,error){var ok bool;err:=r.db.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM rider_vehicles WHERE rider_id=$1 AND active)`,riderID).Scan(&ok);return ok,err}

func (r *Repository) CreateVehicle(ctx context.Context,v models.Vehicle)(models.Vehicle,error){return r.scanVehicle(r.db.QueryRow(ctx,`INSERT INTO rider_vehicles(rider_id,vehicle_type,registration_number,make,model,color,active) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,rider_id,vehicle_type,registration_number,COALESCE(make,''),COALESCE(model,''),COALESCE(color,''),active,created_at,updated_at`,v.RiderID,v.Type,v.RegistrationNumber,v.Make,v.Model,v.Color,v.Active))}

func (r *Repository) ListVehicles(ctx context.Context,riderID uuid.UUID)([]models.Vehicle,error){rows,err:=r.db.Query(ctx,`SELECT id,rider_id,vehicle_type,registration_number,COALESCE(make,''),COALESCE(model,''),COALESCE(color,''),active,created_at,updated_at FROM rider_vehicles WHERE rider_id=$1 ORDER BY active DESC,created_at DESC`,riderID);if err!=nil{return nil,err};defer rows.Close();var out []models.Vehicle;for rows.Next(){v,err:=r.scanVehicle(rows);if err!=nil{return nil,err};out=append(out,v)};return out,rows.Err()}

func (r *Repository) SetActiveVehicle(ctx context.Context,riderID,vehicleID uuid.UUID)(models.Vehicle,error){tx,err:=r.db.Begin(ctx);if err!=nil{return models.Vehicle{},err};defer tx.Rollback(ctx);var exists bool;if err=tx.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM rider_vehicles WHERE id=$1 AND rider_id=$2)`,vehicleID,riderID).Scan(&exists);err!=nil{return models.Vehicle{},err};if !exists{return models.Vehicle{},ErrNotFound};if _,err=tx.Exec(ctx,`UPDATE rider_vehicles SET active=false,updated_at=now() WHERE rider_id=$1 AND active`,riderID);err!=nil{return models.Vehicle{},err};v,err:=r.scanVehicle(tx.QueryRow(ctx,`UPDATE rider_vehicles SET active=true,updated_at=now() WHERE id=$1 AND rider_id=$2 RETURNING id,rider_id,vehicle_type,registration_number,COALESCE(make,''),COALESCE(model,''),COALESCE(color,''),active,created_at,updated_at`,vehicleID,riderID));if err!=nil{return models.Vehicle{},err};if err=tx.Commit(ctx);err!=nil{return models.Vehicle{},err};return v,nil}

func (r *Repository) scanVehicle(row pgx.Row)(models.Vehicle,error){var v models.Vehicle;err:=row.Scan(&v.ID,&v.RiderID,&v.Type,&v.RegistrationNumber,&v.Make,&v.Model,&v.Color,&v.Active,&v.CreatedAt,&v.UpdatedAt);if errors.Is(err,pgx.ErrNoRows){return models.Vehicle{},ErrNotFound};return v,err}

var _ = time.Now
