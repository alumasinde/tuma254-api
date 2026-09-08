package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	identityrepo "github.com/alumasinde/tuma254-api/internal/identity/repositories"
	"github.com/alumasinde/tuma254-api/internal/riders/models"
	riderrepo "github.com/alumasinde/tuma254-api/internal/riders/repositories"
	"github.com/google/uuid"
)

var (
	ErrInvalidState=errors.New("invalid rider state transition")
	ErrNotApproved=errors.New("rider is not approved")
	ErrActiveVehicle=errors.New("active vehicle required")
	ErrInvalidVehicle=errors.New("invalid vehicle")
)

type RoleAssigner interface { AssignRole(context.Context, uuid.UUID, string) error }

type Service struct { repo *riderrepo.Repository; users identityrepo.UsersRepository; roles RoleAssigner }
func New(repo *riderrepo.Repository, users identityrepo.UsersRepository, roles RoleAssigner)*Service{return &Service{repo:repo,users:users,roles:roles}}

func(s *Service) Get(ctx context.Context,userID uuid.UUID)(models.Profile,error){return s.repo.FindProfile(ctx,userID)}
func(s *Service) CreateApplication(ctx context.Context,userID uuid.UUID)(models.Profile,error){if _,err:=s.users.FindByID(ctx,userID);err!=nil{return models.Profile{},err};p,err:=s.repo.FindProfile(ctx,userID);if err==nil{return p,nil};if !errors.Is(err,riderrepo.ErrNotFound){return models.Profile{},err};return s.repo.CreateProfile(ctx,userID)}
func(s *Service) SubmitApplication(ctx context.Context,userID uuid.UUID)(models.Profile,error){p,err:=s.repo.FindProfile(ctx,userID);if err!=nil{return models.Profile{},err};if p.VerificationStatus!=models.VerificationDraft&&p.VerificationStatus!=models.VerificationRejected{return models.Profile{},ErrInvalidState};ok,err:=s.repo.HasActiveVehicle(ctx,p.ID);if err!=nil{return models.Profile{},err};if !ok{return models.Profile{},ErrActiveVehicle};return s.repo.UpdateProfile(ctx,userID,models.VerificationSubmitted,models.AvailabilityOffline,"")}
func(s *Service) SetAvailability(ctx context.Context,userID uuid.UUID,to models.Availability)(models.Profile,error){p,err:=s.repo.FindProfile(ctx,userID);if err!=nil{return models.Profile{},err};if p.VerificationStatus!=models.VerificationApproved{return models.Profile{},ErrNotApproved};if to==models.AvailabilityAvailable{ok,err:=s.repo.HasActiveVehicle(ctx,p.ID);if err!=nil{return models.Profile{},err};if !ok{return models.Profile{},ErrActiveVehicle}};if !validAvailabilityTransition(p.Availability,to){return models.Profile{},ErrInvalidState};return s.repo.UpdateProfile(ctx,userID,p.VerificationStatus,to,p.RejectionReason)}
func(s *Service) AddVehicle(ctx context.Context,userID uuid.UUID,v models.Vehicle)(models.Vehicle,error){p,err:=s.repo.FindProfile(ctx,userID);if err!=nil{return models.Vehicle{},err};if p.VerificationStatus!=models.VerificationDraft&&p.VerificationStatus!=models.VerificationRejected{return models.Vehicle{},ErrInvalidState};v.RiderID=p.ID;v.Type=strings.ToLower(strings.TrimSpace(v.Type));v.RegistrationNumber=normalizeRegistration(v.RegistrationNumber);v.Make=strings.TrimSpace(v.Make);v.Model=strings.TrimSpace(v.Model);v.Color=strings.TrimSpace(v.Color);if !validVehicleType(v.Type)||v.RegistrationNumber==""{return models.Vehicle{},ErrInvalidVehicle};v.Active=false;created,err:=s.repo.CreateVehicle(ctx,v);if err!=nil{return models.Vehicle{},err};return s.repo.SetActiveVehicle(ctx,p.ID,created.ID)}
func(s *Service) ListVehicles(ctx context.Context,userID uuid.UUID)([]models.Vehicle,error){p,err:=s.repo.FindProfile(ctx,userID);if err!=nil{return nil,err};return s.repo.ListVehicles(ctx,p.ID)}
func(s *Service) SetActiveVehicle(ctx context.Context,userID,vehicleID uuid.UUID)(models.Vehicle,error){p,err:=s.repo.FindProfile(ctx,userID);if err!=nil{return models.Vehicle{},err};if p.VerificationStatus!=models.VerificationDraft&&p.VerificationStatus!=models.VerificationRejected{return models.Vehicle{},ErrInvalidState};return s.repo.SetActiveVehicle(ctx,p.ID,vehicleID)}
func(s *Service) Approve(ctx context.Context,userID uuid.UUID)(models.Profile,error){p,err:=s.repo.FindProfile(ctx,userID);if err!=nil{return models.Profile{},err};if p.VerificationStatus!=models.VerificationSubmitted&&p.VerificationStatus!=models.VerificationApproved{return models.Profile{},ErrInvalidState};if p.VerificationStatus!=models.VerificationApproved{p,err=s.repo.UpdateProfile(ctx,userID,models.VerificationApproved,models.AvailabilityOffline,"");if err!=nil{return models.Profile{},err}};if s.roles!=nil{if err:=s.roles.AssignRole(ctx,userID,"rider");err!=nil{return models.Profile{},fmt.Errorf("assign rider role: %w",err)}};return p,nil}
func(s *Service) Reject(ctx context.Context,userID uuid.UUID,reason string)(models.Profile,error){p,err:=s.repo.FindProfile(ctx,userID);if err!=nil{return models.Profile{},err};if p.VerificationStatus!=models.VerificationSubmitted{return models.Profile{},ErrInvalidState};return s.repo.UpdateProfile(ctx,userID,models.VerificationRejected,models.AvailabilityOffline,strings.TrimSpace(reason))}
func(s *Service) Suspend(ctx context.Context,userID uuid.UUID)(models.Profile,error){p,err:=s.repo.FindProfile(ctx,userID);if err!=nil{return models.Profile{},err};if p.VerificationStatus!=models.VerificationApproved{return models.Profile{},ErrInvalidState};return s.repo.UpdateProfile(ctx,userID,models.VerificationSuspended,models.AvailabilityOffline,"")}
func validAvailabilityTransition(from,to models.Availability)bool{if from==to{return true};switch from{case models.AvailabilityOffline:return to==models.AvailabilityAvailable;case models.AvailabilityAvailable:return to==models.AvailabilityOffline||to==models.AvailabilityBusy;case models.AvailabilityBusy:return to==models.AvailabilityOffline||to==models.AvailabilityAvailable};return false}
func validVehicleType(v string)bool{switch v{case"motorcycle","bicycle","car","van","truck":return true};return false}
func normalizeRegistration(v string)string{return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(v)," ",""))}
