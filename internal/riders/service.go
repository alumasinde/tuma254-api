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

type Service struct {
	repo     *Repository
	identity *identity.Repository
}

func NewService(repo *Repository, identityRepo *identity.Repository) *Service {
	return &Service{repo: repo, identity: identityRepo}
}

func (s *Service) Get(ctx context.Context, userID string) (PublicProfile, error) {
	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return PublicProfile{}, ErrNotFound
	}
	profile, err := s.repo.FindProfile(ctx, id)
	if err != nil {
		return PublicProfile{}, err
	}
	return publicProfile(profile), nil
}

func (s *Service) CreateApplication(ctx context.Context, userID string) (PublicProfile, error) {
	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return PublicProfile{}, ErrNotFound
	}
	if _, err := s.identity.FindUserByID(ctx, id); err != nil {
		return PublicProfile{}, err
	}
	existing, err := s.repo.FindProfile(ctx, id)
	if err == nil {
		return publicProfile(existing), nil
	}
	if err != ErrNotFound {
		return PublicProfile{}, err
	}
	now := time.Now().UTC()
	profile, err := s.repo.CreateProfile(ctx, Profile{
		UserID: id, VerificationStatus: VerificationDraft, Availability: AvailabilityOffline,
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return PublicProfile{}, err
	}
	return publicProfile(profile), nil
}

func (s *Service) SubmitApplication(ctx context.Context, userID string) (PublicProfile, error) {
	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return PublicProfile{}, ErrNotFound
	}
	profile, err := s.repo.FindProfile(ctx, id)
	if err != nil {
		return PublicProfile{}, err
	}
	if profile.VerificationStatus != VerificationDraft && profile.VerificationStatus != VerificationRejected {
		return PublicProfile{}, ErrInvalidState
	}
	hasVehicle, err := s.repo.HasActiveVehicle(ctx, id)
	if err != nil {
		return PublicProfile{}, err
	}
	if !hasVehicle {
		return PublicProfile{}, ErrActiveVehicle
	}
	profile, err = s.repo.UpdateProfile(ctx, id, bson.M{
		"verificationStatus": VerificationSubmitted,
		"availability": AvailabilityOffline,
		"rejectionReason": "",
	})
	if err != nil {
		return PublicProfile{}, err
	}
	return publicProfile(profile), nil
}

func (s *Service) SetAvailability(ctx context.Context, userID, availability string) (PublicProfile, error) {
	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return PublicProfile{}, ErrNotFound
	}
	profile, err := s.repo.FindProfile(ctx, id)
	if err != nil {
		return PublicProfile{}, err
	}
	availability = strings.ToLower(strings.TrimSpace(availability))
	if availability != AvailabilityOffline && availability != AvailabilityAvailable && availability != AvailabilityBusy {
		return PublicProfile{}, ErrInvalidState
	}
	if profile.VerificationStatus != VerificationApproved {
		return PublicProfile{}, ErrNotApproved
	}
	if availability == AvailabilityAvailable {
		hasVehicle, err := s.repo.HasActiveVehicle(ctx, id)
		if err != nil {
			return PublicProfile{}, err
		}
		if !hasVehicle {
			return PublicProfile{}, ErrActiveVehicle
		}
	}
	if !validAvailabilityTransition(profile.Availability, availability) {
		return PublicProfile{}, ErrInvalidState
	}
	profile, err = s.repo.UpdateProfile(ctx, id, bson.M{"availability": availability})
	if err != nil {
		return PublicProfile{}, err
	}
	return publicProfile(profile), nil
}

func (s *Service) AddVehicle(ctx context.Context, userID string, vehicle Vehicle) (PublicVehicle, error) {
	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return PublicVehicle{}, ErrNotFound
	}
	profile, err := s.repo.FindProfile(ctx, id)
	if err != nil {
		return PublicVehicle{}, err
	}
	if profile.VerificationStatus != VerificationDraft && profile.VerificationStatus != VerificationRejected {
		return PublicVehicle{}, ErrInvalidState
	}
	vehicle.RiderID = id
	vehicle.Type = strings.ToLower(strings.TrimSpace(vehicle.Type))
	vehicle.RegistrationNumber = normalizeRegistration(vehicle.RegistrationNumber)
	vehicle.Make = strings.TrimSpace(vehicle.Make)
	vehicle.Model = strings.TrimSpace(vehicle.Model)
	vehicle.Color = strings.TrimSpace(vehicle.Color)
	if !validVehicleType(vehicle.Type) || vehicle.RegistrationNumber == "" {
		return PublicVehicle{}, ErrInvalidVehicle
	}
	now := time.Now().UTC()
	vehicle.CreatedAt = now
	vehicle.UpdatedAt = now
	vehicle.Active = true
	created, err := s.repo.CreateVehicle(ctx, vehicle)
	if err != nil {
		return PublicVehicle{}, err
	}
	created, err = s.repo.SetActiveVehicle(ctx, id, created.ID)
	if err != nil {
		return PublicVehicle{}, err
	}
	return publicVehicle(created), nil
}

func (s *Service) ListVehicles(ctx context.Context, userID string) ([]PublicVehicle, error) {
	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, ErrNotFound
	}
	vehicles, err := s.repo.ListVehicles(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]PublicVehicle, 0, len(vehicles))
	for _, vehicle := range vehicles {
		out = append(out, publicVehicle(vehicle))
	}
	return out, nil
}

func (s *Service) SetActiveVehicle(ctx context.Context, userID, vehicleID string) (PublicVehicle, error) {
	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return PublicVehicle{}, ErrNotFound
	}
	profile, err := s.repo.FindProfile(ctx, id)
	if err != nil {
		return PublicVehicle{}, err
	}
	if profile.VerificationStatus != VerificationDraft && profile.VerificationStatus != VerificationRejected {
		return PublicVehicle{}, ErrInvalidState
	}
	vid, err := bson.ObjectIDFromHex(vehicleID)
	if err != nil {
		return PublicVehicle{}, ErrNotFound
	}
	vehicle, err := s.repo.SetActiveVehicle(ctx, id, vid)
	if err != nil {
		return PublicVehicle{}, err
	}
	return publicVehicle(vehicle), nil
}

func (s *Service) Reject(ctx context.Context, userID, reason string) (PublicProfile, error) {
	return s.transitionByOperations(ctx, userID, VerificationRejected, reason)
}

func (s *Service) Approve(ctx context.Context, userID string) (PublicProfile, error) {
	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return PublicProfile{}, ErrNotFound
	}
	profile, err := s.repo.FindProfile(ctx, id)
	if err != nil {
		return PublicProfile{}, err
	}
	if profile.VerificationStatus != VerificationSubmitted && profile.VerificationStatus != VerificationApproved {
		return PublicProfile{}, ErrInvalidState
	}
	if profile.VerificationStatus != VerificationApproved {
		profile, err = s.repo.UpdateProfile(ctx, id, bson.M{
			"verificationStatus": VerificationApproved,
			"availability": AvailabilityOffline,
			"rejectionReason": "",
		})
		if err != nil {
			return PublicProfile{}, err
		}
	}
	if err := s.identity.AssignRole(ctx, id, "rider"); err != nil {
		return PublicProfile{}, fmt.Errorf("assign rider role: %w", err)
	}
	return publicProfile(profile), nil
}

func (s *Service) Suspend(ctx context.Context, userID string) (PublicProfile, error) {
	return s.transitionByOperations(ctx, userID, VerificationSuspended, "")
}

func (s *Service) transitionByOperations(ctx context.Context, userID, target, reason string) (PublicProfile, error) {
	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return PublicProfile{}, ErrNotFound
	}
	profile, err := s.repo.FindProfile(ctx, id)
	if err != nil {
		return PublicProfile{}, err
	}
	if target == VerificationRejected && profile.VerificationStatus != VerificationSubmitted {
		return PublicProfile{}, ErrInvalidState
	}
	if target == VerificationSuspended && profile.VerificationStatus != VerificationApproved {
		return PublicProfile{}, ErrInvalidState
	}
	profile, err = s.repo.UpdateProfile(ctx, id, bson.M{
		"verificationStatus": target,
		"availability": AvailabilityOffline,
		"rejectionReason": strings.TrimSpace(reason),
	})
	if err != nil {
		return PublicProfile{}, err
	}
	return publicProfile(profile), nil
}

func validAvailabilityTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case AvailabilityOffline:
		return to == AvailabilityAvailable
	case AvailabilityAvailable:
		return to == AvailabilityOffline || to == AvailabilityBusy
	case AvailabilityBusy:
		return to == AvailabilityOffline || to == AvailabilityAvailable
	default:
		return false
	}
}

func validVehicleType(value string) bool {
	switch value {
	case "motorcycle", "bicycle", "car", "van", "truck":
		return true
	default:
		return false
	}
}

func normalizeRegistration(value string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
}

func publicProfile(profile Profile) PublicProfile {
	return PublicProfile{
		UserID: profile.UserID.Hex(),
		VerificationStatus: profile.VerificationStatus,
		Availability: profile.Availability,
		RejectionReason: profile.RejectionReason,
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}
}

func publicVehicle(vehicle Vehicle) PublicVehicle {
	return PublicVehicle{
		ID: vehicle.ID.Hex(),
		Type: vehicle.Type,
		RegistrationNumber: vehicle.RegistrationNumber,
		Make: vehicle.Make,
		Model: vehicle.Model,
		Color: vehicle.Color,
		Active: vehicle.Active,
		CreatedAt: vehicle.CreatedAt,
		UpdatedAt: vehicle.UpdatedAt,
	}
}
