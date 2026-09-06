package riders

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/alumasinde/tuma254-api/internal/identity"
	httpserver "github.com/alumasinde/tuma254-api/internal/platform/http"
)

type Authenticator interface {
	RequireAuth(http.HandlerFunc) http.HandlerFunc
	RequirePermission(string, http.HandlerFunc) http.HandlerFunc
}

type Handler struct {
	service *Service
	auth    Authenticator
}

func NewHandler(service *Service, auth Authenticator) *Handler {
	return &Handler{service: service, auth: auth}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/rider/profile", h.auth.RequireAuth(h.get))
	mux.HandleFunc("POST /api/v1/rider/application", h.auth.RequireAuth(h.createApplication))
	mux.HandleFunc("POST /api/v1/rider/application/submit", h.auth.RequireAuth(h.submitApplication))
	mux.HandleFunc("PUT /api/v1/rider/availability", h.auth.RequireAuth(h.setAvailability))
	mux.HandleFunc("GET /api/v1/rider/vehicles", h.auth.RequireAuth(h.listVehicles))
	mux.HandleFunc("POST /api/v1/rider/vehicles", h.auth.RequireAuth(h.addVehicle))
	mux.HandleFunc("PUT /api/v1/rider/vehicles/{vehicleID}/active", h.auth.RequireAuth(h.setActiveVehicle))
	mux.HandleFunc("POST /api/v1/operations/riders/{userID}/approve", h.auth.RequirePermission("operations.users.manage", h.approve))
	mux.HandleFunc("POST /api/v1/operations/riders/{userID}/reject", h.auth.RequirePermission("operations.users.manage", h.reject))
	mux.HandleFunc("POST /api/v1/operations/riders/{userID}/suspend", h.auth.RequirePermission("operations.users.manage", h.suspend))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok { unauthorized(w); return }
	profile, err := h.service.Get(r.Context(), userID)
	if errors.Is(err, ErrNotFound) { httpserver.WriteJSON(w, http.StatusNotFound, map[string]string{"error":"rider_profile_not_found"}); return }
	if err != nil { httpserver.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error":"rider_profile_failed"}); return }
	httpserver.WriteJSON(w, http.StatusOK, profile)
}

func (h *Handler) createApplication(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok { unauthorized(w); return }
	profile, err := h.service.CreateApplication(r.Context(), userID)
	if err != nil { httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"error":"rider_application_failed"}); return }
	httpserver.WriteJSON(w, http.StatusCreated, profile)
}

func (h *Handler) submitApplication(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok { unauthorized(w); return }
	profile, err := h.service.SubmitApplication(r.Context(), userID)
	switch {
	case errors.Is(err, ErrActiveVehicle):
		httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"error":"active_vehicle_required"})
	case errors.Is(err, ErrInvalidState):
		httpserver.WriteJSON(w, http.StatusConflict, map[string]string{"error":"invalid_rider_state"})
	case errors.Is(err, ErrNotFound):
		httpserver.WriteJSON(w, http.StatusNotFound, map[string]string{"error":"rider_profile_not_found"})
	case err != nil:
		httpserver.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error":"rider_application_failed"})
	default:
		httpserver.WriteJSON(w, http.StatusOK, profile)
	}
}

func (h *Handler) setAvailability(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok { unauthorized(w); return }
	var input struct { Availability string `json:"availability"` }
	if err := decode(r, &input); err != nil { badRequest(w); return }
	profile, err := h.service.SetAvailability(r.Context(), userID, input.Availability)
	switch {
	case errors.Is(err, ErrNotApproved):
		httpserver.WriteJSON(w, http.StatusForbidden, map[string]string{"error":"rider_not_approved"})
	case errors.Is(err, ErrActiveVehicle):
		httpserver.WriteJSON(w, http.StatusConflict, map[string]string{"error":"active_vehicle_required"})
	case errors.Is(err, ErrInvalidState):
		httpserver.WriteJSON(w, http.StatusConflict, map[string]string{"error":"invalid_availability_transition"})
	case err != nil:
		httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"error":"availability_update_failed"})
	default:
		httpserver.WriteJSON(w, http.StatusOK, profile)
	}
}

func (h *Handler) listVehicles(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok { unauthorized(w); return }
	vehicles, err := h.service.ListVehicles(r.Context(), userID)
	if err != nil { httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"error":"vehicle_list_failed"}); return }
	httpserver.WriteJSON(w, http.StatusOK, vehicles)
}

func (h *Handler) addVehicle(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok { unauthorized(w); return }
	var input struct {
		Type string `json:"type"`
		RegistrationNumber string `json:"registration_number"`
		Make string `json:"make"`
		Model string `json:"model"`
		Color string `json:"color"`
	}
	if err := decode(r, &input); err != nil { badRequest(w); return }
	vehicle, err := h.service.AddVehicle(r.Context(), userID, Vehicle{
		Type: input.Type, RegistrationNumber: input.RegistrationNumber,
		Make: input.Make, Model: input.Model, Color: input.Color,
	})
	switch {
	case errors.Is(err, ErrInvalidVehicle):
		httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"error":"invalid_vehicle"})
	case errors.Is(err, ErrInvalidState):
		httpserver.WriteJSON(w, http.StatusConflict, map[string]string{"error":"invalid_rider_state"})
	case err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate"):
		httpserver.WriteJSON(w, http.StatusConflict, map[string]string{"error":"vehicle_already_exists"})
	case err != nil:
		httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"error":"vehicle_create_failed"})
	default:
		httpserver.WriteJSON(w, http.StatusCreated, vehicle)
	}
}

func (h *Handler) setActiveVehicle(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(r)
	if !ok { unauthorized(w); return }
	vehicle, err := h.service.SetActiveVehicle(r.Context(), userID, r.PathValue("vehicleID"))
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.WriteJSON(w, http.StatusNotFound, map[string]string{"error":"vehicle_not_found"})
	case errors.Is(err, ErrInvalidState):
		httpserver.WriteJSON(w, http.StatusConflict, map[string]string{"error":"invalid_rider_state"})
	case err != nil:
		httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"error":"vehicle_update_failed"})
	default:
		httpserver.WriteJSON(w, http.StatusOK, vehicle)
	}
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	profile, err := h.service.Approve(r.Context(), r.PathValue("userID"))
	if err != nil { operationError(w, err); return }
	httpserver.WriteJSON(w, http.StatusOK, profile)
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request) {
	var input struct { Reason string `json:"reason"` }
	if err := decode(r, &input); err != nil { badRequest(w); return }
	profile, err := h.service.Reject(r.Context(), r.PathValue("userID"), input.Reason)
	if err != nil { operationError(w, err); return }
	httpserver.WriteJSON(w, http.StatusOK, profile)
}

func (h *Handler) suspend(w http.ResponseWriter, r *http.Request) {
	profile, err := h.service.Suspend(r.Context(), r.PathValue("userID"))
	if err != nil { operationError(w, err); return }
	httpserver.WriteJSON(w, http.StatusOK, profile)
}

func currentUserID(r *http.Request) (string, bool) {
	claims, ok := identity.ClaimsFromContext(r.Context())
	return claims.Subject, ok
}

func decode(r *http.Request, dst any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil { return err }
	if err := decoder.Decode(&struct{}{}); err != io.EOF { return errors.New("multiple JSON values") }
	return nil
}

func operationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.WriteJSON(w, http.StatusNotFound, map[string]string{"error":"rider_not_found"})
	case errors.Is(err, ErrInvalidState):
		httpserver.WriteJSON(w, http.StatusConflict, map[string]string{"error":"invalid_rider_state"})
	default:
		httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"error":"rider_operation_failed"})
	}
}

func badRequest(w http.ResponseWriter) { httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"error":"invalid_request"}) }
func unauthorized(w http.ResponseWriter) { httpserver.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error":"invalid_session"}) }
