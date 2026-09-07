package riders

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alumasinde/tuma254-api/internal/identity"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	ErrInvalidState   = errors.New("invalid rider state transition")
	ErrNotApproved    = errors.New("rider is not approved")
	ErrActiveVehicle  = errors.New("active vehicle required")
	ErrInvalidVehicle = errors.New("invalid vehicle")
)

type Service struct { repo *Repository; identity *identity.Repository }
func NewService(repo *Repository, identityRepo *identity.Repository) *Service { return &Service{repo:repo,identity:identityRepo} }

func (s *Service) CanPublishLocation(ctx context.Context,userID string)(bool,error){
	id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return false,nil}
	p,err:=s.repo.FindProfile(ctx,id);if errors.Is(err,ErrNotFound){return false,nil};if err!=nil{return false,err}
	return p.VerificationStatus==VerificationApproved,nil
}
func (s *Service) CanAppearNearby(ctx context.Context,userID string)(bool,error){
	id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return false,nil}
	p,err:=s.repo.FindProfile(ctx,id);if errors.Is(err,ErrNotFound){return false,nil};if err!=nil{return false,err}
	return p.VerificationStatus==VerificationApproved&&p.Availability==AvailabilityAvailable,nil
}

func (s *Service) Get(ctx context.Context,userID string)(PublicProfile,error){id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return PublicProfile{},ErrNotFound};p,err:=s.repo.FindProfile(ctx,id);if err!=nil{return PublicProfile{},err};return publicProfile(p),nil}
func (s *Service) CreateApplication(ctx context.Context,userID string)(PublicProfile,error){id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return PublicProfile{},ErrNotFound};if _,err=s.identity.FindUserByID(ctx,id);err!=nil{return PublicProfile{},err};p,err:=s.repo.FindProfile(ctx,id);if err==nil{return publicProfile(p),nil};if err!=ErrNotFound{return PublicProfile{},err};now:=time.Now().UTC();p,err=s.repo.CreateProfile(ctx,Profile{UserID:id,VerificationStatus:VerificationDraft,Availability:AvailabilityOffline,CreatedAt:now,UpdatedAt:now});if err!=nil{return PublicProfile{},err};return publicProfile(p),nil}
func (s *Service) SubmitApplication(ctx context.Context,userID string)(PublicProfile,error){id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return PublicProfile{},ErrNotFound};p,err:=s.repo.FindProfile(ctx,id);if err!=nil{return PublicProfile{},err};if p.VerificationStatus!=VerificationDraft&&p.VerificationStatus!=VerificationRejected{return PublicProfile{},ErrInvalidState};has,err:=s.repo.HasActiveVehicle(ctx,id);if err!=nil{return PublicProfile{},err};if !has{return PublicProfile{},ErrActiveVehicle};p,err=s.repo.UpdateProfile(ctx,id,bson.M{"verificationStatus":VerificationSubmitted,"availability":AvailabilityOffline,"rejectionReason":""});if err!=nil{return PublicProfile{},err};return publicProfile(p),nil}
func (s *Service) SetAvailability(ctx context.Context,userID,availability string)(PublicProfile,error){id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return PublicProfile{},ErrNotFound};p,err:=s.repo.FindProfile(ctx,id);if err!=nil{return PublicProfile{},err};availability=strings.ToLower(strings.TrimSpace(availability));if availability!=AvailabilityOffline&&availability!=AvailabilityAvailable&&availability!=AvailabilityBusy{return PublicProfile{},ErrInvalidState};if p.VerificationStatus!=VerificationApproved{return PublicProfile{},ErrNotApproved};if availability==AvailabilityAvailable{has,err:=s.repo.HasActiveVehicle(ctx,id);if err!=nil{return PublicProfile{},err};if !has{return PublicProfile{},ErrActiveVehicle}};if !validAvailabilityTransition(p.Availability,availability){return PublicProfile{},ErrInvalidState};p,err=s.repo.UpdateProfile(ctx,id,bson.M{"availability":availability});if err!=nil{return PublicProfile{},err};return publicProfile(p),nil}
func (s *Service) AddVehicle(ctx context.Context,userID string,vehicle Vehicle)(PublicVehicle,error){id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return PublicVehicle{},ErrNotFound};p,err:=s.repo.FindProfile(ctx,id);if err!=nil{return PublicVehicle{},err};if p.VerificationStatus!=VerificationDraft&&p.VerificationStatus!=VerificationRejected{return PublicVehicle{},ErrInvalidState};vehicle.RiderID=id;vehicle.Type=strings.ToLower(strings.TrimSpace(vehicle.Type));vehicle.RegistrationNumber=normalizeRegistration(vehicle.RegistrationNumber);vehicle.Make=strings.TrimSpace(vehicle.Make);vehicle.Model=strings.TrimSpace(vehicle.Model);vehicle.Color=strings.TrimSpace(vehicle.Color);if !validVehicleType(vehicle.Type)||vehicle.RegistrationNumber==""{return PublicVehicle{},ErrInvalidVehicle};now:=time.Now().UTC();vehicle.Active=false;vehicle.CreatedAt=now;vehicle.UpdatedAt=now;created,err:=s.repo.CreateVehicle(ctx,vehicle);if err!=nil{return PublicVehicle{},err};created,err=s.repo.SetActiveVehicle(ctx,id,created.ID);if err!=nil{return PublicVehicle{},err};return publicVehicle(created),nil}
func (s *Service) ListVehicles(ctx context.Context,userID string)([]PublicVehicle,error){id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return nil,ErrNotFound};vs,err:=s.repo.ListVehicles(ctx,id);if err!=nil{return nil,err};out:=make([]PublicVehicle,0,len(vs));for _,v:=range vs{out=append(out,publicVehicle(v))};return out,nil}
func (s *Service) SetActiveVehicle(ctx context.Context,userID,vehicleID string)(PublicVehicle,error){id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return PublicVehicle{},ErrNotFound};p,err:=s.repo.FindProfile(ctx,id);if err!=nil{return PublicVehicle{},err};if p.VerificationStatus!=VerificationDraft&&p.VerificationStatus!=VerificationRejected{return PublicVehicle{},ErrInvalidState};vid,err:=bson.ObjectIDFromHex(vehicleID);if err!=nil{return PublicVehicle{},ErrNotFound};v,err:=s.repo.SetActiveVehicle(ctx,id,vid);if err!=nil{return PublicVehicle{},err};return publicVehicle(v),nil}
func (s *Service) Reject(ctx context.Context,userID,reason string)(PublicProfile,error){return s.transitionByOperations(ctx,userID,VerificationRejected,reason)}
func (s *Service) Approve(ctx context.Context,userID string)(PublicProfile,error){id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return PublicProfile{},ErrNotFound};p,err:=s.repo.FindProfile(ctx,id);if err!=nil{return PublicProfile{},err};if p.VerificationStatus!=VerificationSubmitted&&p.VerificationStatus!=VerificationApproved{return PublicProfile{},ErrInvalidState};if p.VerificationStatus!=VerificationApproved{p,err=s.repo.UpdateProfile(ctx,id,bson.M{"verificationStatus":VerificationApproved,"availability":AvailabilityOffline,"rejectionReason":""});if err!=nil{return PublicProfile{},err}};if err:=s.identity.AssignRole(ctx,id,"rider");err!=nil{return PublicProfile{},fmt.Errorf("assign rider role: %w",err)};return publicProfile(p),nil}
func (s *Service) Suspend(ctx context.Context,userID string)(PublicProfile,error){return s.transitionByOperations(ctx,userID,VerificationSuspended,"")}
func (s *Service) transitionByOperations(ctx context.Context,userID,target,reason string)(PublicProfile,error){id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return PublicProfile{},ErrNotFound};p,err:=s.repo.FindProfile(ctx,id);if err!=nil{return PublicProfile{},err};if target==VerificationRejected&&p.VerificationStatus!=VerificationSubmitted{return PublicProfile{},ErrInvalidState};if target==VerificationSuspended&&p.VerificationStatus!=VerificationApproved{return PublicProfile{},ErrInvalidState};p,err=s.repo.UpdateProfile(ctx,id,bson.M{"verificationStatus":target,"availability":AvailabilityOffline,"rejectionReason":strings.TrimSpace(reason)});if err!=nil{return PublicProfile{},err};return publicProfile(p),nil}
func validAvailabilityTransition(from,to string)bool{if from==to{return true};switch from{case AvailabilityOffline:return to==AvailabilityAvailable;case AvailabilityAvailable:return to==AvailabilityOffline||to==AvailabilityBusy;case AvailabilityBusy:return to==AvailabilityOffline||to==AvailabilityAvailable};return false}
func validVehicleType(v string)bool{switch v{case "motorcycle","bicycle","car","van","truck":return true};return false}
func normalizeRegistration(v string)string{return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(v)," ",""))}
func publicProfile(p Profile)PublicProfile{return PublicProfile{UserID:p.UserID.Hex(),VerificationStatus:p.VerificationStatus,Availability:p.Availability,RejectionReason:p.RejectionReason,CreatedAt:p.CreatedAt,UpdatedAt:p.UpdatedAt}}
func publicVehicle(v Vehicle)PublicVehicle{return PublicVehicle{ID:v.ID.Hex(),Type:v.Type,RegistrationNumber:v.RegistrationNumber,Make:v.Make,Model:v.Model,Color:v.Color,Active:v.Active,CreatedAt:v.CreatedAt,UpdatedAt:v.UpdatedAt}}
