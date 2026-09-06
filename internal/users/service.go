package users

import (
	"context"
	"strings"

	"github.com/alumasinde/tuma254-api/internal/identity"
	"go.mongodb.org/mongo-driver/v2/bson"
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
	if _, err := s.identity.FindUserByID(ctx, id); err != nil {
		return PublicProfile{}, err
	}
	profile, err := s.repo.FindByUserID(ctx, id)
	if err == ErrNotFound {
		return PublicProfile{UserID: id.Hex()}, nil
	}
	if err != nil {
		return PublicProfile{}, err
	}
	return public(profile), nil
}

func (s *Service) Update(ctx context.Context, userID, avatarURL string) (PublicProfile, error) {
	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return PublicProfile{}, ErrNotFound
	}
	if _, err := s.identity.FindUserByID(ctx, id); err != nil {
		return PublicProfile{}, err
	}
	profile, err := s.repo.Upsert(ctx, id, strings.TrimSpace(avatarURL))
	if err != nil {
		return PublicProfile{}, err
	}
	return public(profile), nil
}

func public(profile Profile) PublicProfile {
	return PublicProfile{
		UserID: profile.UserID.Hex(),
		AvatarURL: profile.AvatarURL,
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}
}
