package services

import (
	"context"
	"strings"

	"github.com/alumasinde/tuma254-api/internal/identity/repositories"
	"github.com/alumasinde/tuma254-api/internal/users/models"
	userrepo "github.com/alumasinde/tuma254-api/internal/users/repositories"
	"github.com/google/uuid"
)

type Service struct { profiles *userrepo.Repository; users repositories.UsersRepository }

func New(profiles *userrepo.Repository, users repositories.UsersRepository) *Service {
	return &Service{profiles: profiles, users: users}
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (models.Profile, error) {
	if _, err := s.users.FindByID(ctx, userID); err != nil { return models.Profile{}, err }
	p, err := s.profiles.FindByUserID(ctx, userID)
	if err == userrepo.ErrNotFound { return models.Profile{UserID:userID}, nil }
	return p, err
}

func (s *Service) Update(ctx context.Context, userID uuid.UUID, avatarURL string) (models.Profile, error) {
	if _, err := s.users.FindByID(ctx, userID); err != nil { return models.Profile{}, err }
	return s.profiles.Upsert(ctx, userID, strings.TrimSpace(avatarURL))
}
