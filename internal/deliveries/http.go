package deliveries

import (
	"errors"
	"net/http"

	"github.com/alumasinde/tuma254-api/internal/identity"
	httpserver "github.com/alumasinde/tuma254-api/internal/platform/http"
)

type Authenticator interface{ RequireAuth(http.HandlerFunc) http.HandlerFunc }

type Handler struct{ s *Service; a Authenticator }

func NewHandler(s *Service, a Authenticator) *Handler { return &Handler{s: s, a: a} }

func (h *Handler) RegisterRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/v1/deliveries", h.a.RequireAuth(h.create))
	m.HandleFunc("GET /api/v1/deliveries/{deliveryID}", h.a.RequireAuth(h.get))
	m.HandleFunc("POST /api/v1/deliveries/{deliveryID}/assign", h.a.RequireAuth(h.assign))
	m.HandleFunc("POST /api/v1/deliveries/{deliveryID}/pickup-otp", h.a.RequireAuth(h.pickupOTP))
	m.HandleFunc("POST /api/v1/deliveries/{deliveryID}/pickup/verify", h.a.RequireAuth(h.pickupVerify))
	m.HandleFunc("POST /api/v1/deliveries/{deliveryID}/pickup-otp/resend", h.a.RequireAuth(h.pickupResend))
	m.HandleFunc("POST /api/v1/deliveries/{deliveryID}/transit", h.a.RequireAuth(h.transit))
	m.HandleFunc("POST /api/v1/deliveries/{deliveryID}/delivery-otp", h.a.RequireAuth(h.deliveryOTP))
	m.HandleFunc("POST /api/v1/deliveries/{deliveryID}/delivery/verify", h.a.RequireAuth(h.deliveryVerify))
	m.HandleFunc("POST /api/v1/deliveries/{deliveryID}/delivery-otp/resend", h.a.RequireAuth(h.deliveryResend))
	m.HandleFunc("POST /api/v1/deliveries/{deliveryID}/incidents", h.a.RequireAuth(h.incident))
	m.HandleFunc("POST /api/v1/deliveries/{deliveryID}/fail", h.a.RequireAuth(h.fail))
}

func uid(r *http.Request) string { c, _ := identity.ClaimsFromContext(r.Context()); return c.Subject }
func dec(w http.ResponseWriter, r *http.Request, v any) error { return httpserver.DecodeJSONStrict(w, r, v) }

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in CreateInput
	if dec(w,r,&in)!=nil { invalidRequest(w); return }
	v,e:=h.s.Create(r.Context(),uid(r),in); write(w,v,e,http.StatusCreated)
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	v,e:=h.s.GetForActor(r.Context(),r.PathValue("deliveryID"),uid(r)); write(w,v,e,http.StatusOK)
}
func (h *Handler) assign(w http.ResponseWriter, r *http.Request) {
	var x struct{RiderID string `json:"rider_id"`}
	if dec(w,r,&x)!=nil { invalidRequest(w); return }
	v,e:=h.s.Assign(r.Context(),r.PathValue("deliveryID"),x.RiderID,uid(r)); write(w,v,e,http.StatusOK)
}
func (h *Handler) pickupOTP(w http.ResponseWriter, r *http.Request) {
	_,e:=h.s.RequestPickupOTP(r.Context(),r.PathValue("deliveryID"),uid(r))
	if e!=nil { deliveryError(w,e); return }
	httpserver.WriteJSON(w,http.StatusAccepted,map[string]string{"status":"otp_requested"})
}
func (h *Handler) pickupResend(w http.ResponseWriter, r *http.Request) {
	_,e:=h.s.ResendPickupOTP(r.Context(),r.PathValue("deliveryID"),uid(r))
	if e!=nil { deliveryError(w,e); return }
	httpserver.WriteJSON(w,http.StatusAccepted,map[string]string{"status":"otp_resent"})
}
func (h *Handler) pickupVerify(w http.ResponseWriter, r *http.Request) {
	var x struct{OTP string `json:"otp"`; Custody CustodyInput `json:"custody"`}
	if dec(w,r,&x)!=nil { invalidRequest(w); return }
	v,e:=h.s.VerifyPickupWithCustody(r.Context(),r.PathValue("deliveryID"),uid(r),x.OTP,x.Custody); write(w,v,e,http.StatusOK)
}
func (h *Handler) transit(w http.ResponseWriter, r *http.Request) {
	v,e:=h.s.StartTransit(r.Context(),r.PathValue("deliveryID"),uid(r)); write(w,v,e,http.StatusOK)
}
func (h *Handler) deliveryOTP(w http.ResponseWriter, r *http.Request) {
	_,e:=h.s.RequestDeliveryOTP(r.Context(),r.PathValue("deliveryID"),uid(r))
	if e!=nil { deliveryError(w,e); return }
	httpserver.WriteJSON(w,http.StatusAccepted,map[string]string{"status":"otp_requested"})
}
func (h *Handler) deliveryResend(w http.ResponseWriter, r *http.Request) {
	_,e:=h.s.ResendDeliveryOTP(r.Context(),r.PathValue("deliveryID"),uid(r))
	if e!=nil { deliveryError(w,e); return }
	httpserver.WriteJSON(w,http.StatusAccepted,map[string]string{"status":"otp_resent"})
}
func (h *Handler) deliveryVerify(w http.ResponseWriter, r *http.Request) {
	var x struct{OTP string `json:"otp"`; Custody CustodyInput `json:"custody"`}
	if dec(w,r,&x)!=nil { invalidRequest(w); return }
	v,e:=h.s.VerifyDeliveryWithCustody(r.Context(),r.PathValue("deliveryID"),uid(r),x.OTP,x.Custody); write(w,v,e,http.StatusOK)
}
func (h *Handler) incident(w http.ResponseWriter, r *http.Request) {
	var x struct{Type string `json:"type"`; Reason string `json:"reason"`}
	if dec(w,r,&x)!=nil { invalidRequest(w); return }
	if e:=h.s.ReportIncident(r.Context(),r.PathValue("deliveryID"),uid(r),x.Type,x.Reason);e!=nil { deliveryError(w,e); return }
	httpserver.WriteJSON(w,http.StatusCreated,map[string]string{"status":"reported"})
}
func (h *Handler) fail(w http.ResponseWriter, r *http.Request) {
	var x struct{Reason string `json:"reason"`}
	if dec(w,r,&x)!=nil { invalidRequest(w); return }
	v,e:=h.s.Fail(r.Context(),r.PathValue("deliveryID"),uid(r),x.Reason); write(w,v,e,http.StatusOK)
}

func write(w http.ResponseWriter,v any,e error,status int){ if e!=nil { deliveryError(w,e); return }; httpserver.WriteJSON(w,status,v) }
func invalidRequest(w http.ResponseWriter){ httpserver.WriteJSON(w,http.StatusBadRequest,map[string]string{"error":"invalid_request"}) }
func deliveryError(w http.ResponseWriter,e error) {
	switch {
	case errors.Is(e,ErrNotFound):
		httpserver.WriteJSON(w,http.StatusNotFound,map[string]string{"error":"delivery_not_found"})
	case errors.Is(e,ErrForbidden):
		httpserver.WriteJSON(w,http.StatusForbidden,map[string]string{"error":"delivery_forbidden"})
	case errors.Is(e,ErrState):
		httpserver.WriteJSON(w,http.StatusConflict,map[string]string{"error":"invalid_delivery_state"})
	case errors.Is(e,ErrExpired):
		httpserver.WriteJSON(w,http.StatusGone,map[string]string{"error":"otp_expired"})
	case errors.Is(e,ErrOTPLocked), errors.Is(e,ErrOTPResend):
		httpserver.WriteJSON(w,http.StatusTooManyRequests,map[string]string{"error":"otp_rate_limited"})
	case errors.Is(e,ErrOTP):
		httpserver.WriteJSON(w,http.StatusBadRequest,map[string]string{"error":"invalid_otp"})
	default:
		httpserver.WriteJSON(w,http.StatusBadRequest,map[string]string{"error":"delivery_operation_failed"})
	}
}

type DispatchHandler struct{d *DispatchService;a Authenticator}
func NewDispatchHandler(d *DispatchService,a Authenticator)*DispatchHandler{return &DispatchHandler{d,a}}
func(h *DispatchHandler)RegisterRoutes(m *http.ServeMux){
	m.HandleFunc("POST /api/v1/deliveries/{deliveryID}/dispatch/nearby",h.a.RequireAuth(h.nearby))
	m.HandleFunc("POST /api/v1/delivery-offers",h.a.RequireAuth(h.offer))
	m.HandleFunc("POST /api/v1/delivery-offers/{offerID}/accept",h.a.RequireAuth(h.accept))
	m.HandleFunc("POST /api/v1/delivery-offers/{offerID}/decline",h.a.RequireAuth(h.decline))
}
func(h *DispatchHandler)nearby(w http.ResponseWriter,r *http.Request){var x NearbyDispatchInput;if dec(w,r,&x)!=nil{invalidRequest(w);return};v,e:=h.d.StartNearby(r.Context(),r.PathValue("deliveryID"),uid(r),x);write(w,v,e,http.StatusCreated)}
func(h *DispatchHandler)offer(w http.ResponseWriter,r *http.Request){var x struct{DeliveryID string `json:"delivery_id"`;RiderID string `json:"rider_id"`};if dec(w,r,&x)!=nil{invalidRequest(w);return};v,e:=h.d.Offer(r.Context(),x.DeliveryID,x.RiderID,uid(r));write(w,map[string]any{"id":v.ID.Hex(),"delivery_id":v.DeliveryID.Hex(),"expires_at":v.ExpiresAt},e,http.StatusCreated)}
func(h *DispatchHandler)accept(w http.ResponseWriter,r *http.Request){v,e:=h.d.Accept(r.Context(),r.PathValue("offerID"),uid(r));write(w,v,e,http.StatusOK)}
func(h *DispatchHandler)decline(w http.ResponseWriter,r *http.Request){e:=h.d.Decline(r.Context(),r.PathValue("offerID"),uid(r));if e!=nil{deliveryError(w,e);return};httpserver.WriteJSON(w,http.StatusOK,map[string]string{"status":"declined"})}
