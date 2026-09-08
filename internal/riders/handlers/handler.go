package handlers

import (
	"errors"
	"net/http"

	"github.com/alumasinde/tuma254-api/internal/platform/httpctx"
	"github.com/alumasinde/tuma254-api/internal/platform/httpx"
	"github.com/alumasinde/tuma254-api/internal/riders/models"
	"github.com/alumasinde/tuma254-api/internal/riders/services"
	"github.com/google/uuid"
)

type Handler struct{ svc *services.Service }
func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpctx.UserID(r.Context()); if !ok { httpx.Error(w, 401, "unauthorized"); return }
	profile, err := h.svc.Get(r.Context(), userID)
	if errors.Is(err, services.ErrInvalidState) { httpx.Error(w, 409, "invalid rider state"); return }
	if err != nil { httpx.Error(w, 404, "rider profile not found"); return }
	httpx.WriteJSON(w, 200, profile)
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpctx.UserID(r.Context()); if !ok { httpx.Error(w, 401, "unauthorized"); return }
	profile, err := h.svc.CreateApplication(r.Context(), userID); if err != nil { httpx.Error(w, 400, "rider application failed"); return }
	httpx.WriteJSON(w, 201, profile)
}
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpctx.UserID(r.Context()); if !ok { httpx.Error(w, 401, "unauthorized"); return }
	profile, err := h.svc.SubmitApplication(r.Context(), userID); if err != nil { httpx.Error(w, 409, err.Error()); return }
	httpx.WriteJSON(w, 200, profile)
}
func (h *Handler) SetAvailability(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpctx.UserID(r.Context()); if !ok { httpx.Error(w, 401, "unauthorized"); return }
	var in struct{ Availability models.Availability `json:"availability"` }
	if err := httpx.DecodeJSON(w, r, &in); err != nil { httpx.Error(w, 400, "invalid request"); return }
	profile, err := h.svc.SetAvailability(r.Context(), userID, in.Availability); if err != nil { httpx.Error(w, 409, err.Error()); return }
	httpx.WriteJSON(w, 200, profile)
}
func (h *Handler) AddVehicle(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpctx.UserID(r.Context()); if !ok { httpx.Error(w, 401, "unauthorized"); return }
	var in models.Vehicle
	if err := httpx.DecodeJSON(w, r, &in); err != nil { httpx.Error(w, 400, "invalid request"); return }
	vehicle, err := h.svc.AddVehicle(r.Context(), userID, in); if err != nil { httpx.Error(w, 400, err.Error()); return }
	httpx.WriteJSON(w, 201, vehicle)
}
func (h *Handler) ListVehicles(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpctx.UserID(r.Context()); if !ok { httpx.Error(w, 401, "unauthorized"); return }
	vehicles, err := h.svc.ListVehicles(r.Context(), userID); if err != nil { httpx.Error(w, 404, "rider profile not found"); return }
	httpx.WriteJSON(w, 200, vehicles)
}
func (h *Handler) SetActiveVehicle(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpctx.UserID(r.Context()); if !ok { httpx.Error(w, 401, "unauthorized"); return }
	vehicleID, err := uuid.Parse(r.PathValue("vehicleID")); if err != nil { httpx.Error(w, 400, "invalid vehicle id"); return }
	vehicle, err := h.svc.SetActiveVehicle(r.Context(), userID, vehicleID); if err != nil { httpx.Error(w, 409, err.Error()); return }
	httpx.WriteJSON(w, 200, vehicle)
}
