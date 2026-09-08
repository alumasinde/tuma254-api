package handlers

import "github.com/alumasinde/tuma254-api/internal/identity/services"

// Handler exposes the HTTP handlers for the Identity module.
type Handler struct {
	svc *services.Service
}

// New creates an Identity HTTP handler using the provided service.
func New(svc *services.Service) *Handler {
	return &Handler{svc: svc}
}
