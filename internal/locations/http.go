package locations

import (
 "encoding/json"
 "errors"
 "io"
 "net/http"
 "strconv"
 "github.com/alumasinde/tuma254-api/internal/identity"
 httpserver "github.com/alumasinde/tuma254-api/internal/platform/http"
)

type Authenticator interface{RequireAuth(http.HandlerFunc)http.HandlerFunc}
type Handler struct{service *Service;auth Authenticator}
func NewHandler(service *Service,auth Authenticator)*Handler{return &Handler{service:service,auth:auth}}
func(h *Handler)RegisterRoutes(mux *http.ServeMux){
 mux.HandleFunc("PUT /api/v1/rider/location",h.auth.RequireAuth(h.updateRider))
 mux.HandleFunc("GET /api/v1/riders/nearby",h.auth.RequireAuth(h.nearby))
 mux.HandleFunc("GET /api/v1/places",h.auth.RequireAuth(h.listPlaces))
 mux.HandleFunc("POST /api/v1/places",h.auth.RequireAuth(h.createPlace))
 mux.HandleFunc("DELETE /api/v1/places/{placeID}",h.auth.RequireAuth(h.deletePlace))
}
func(h *Handler)updateRider(w http.ResponseWriter,r *http.Request){id,ok:=userID(r);if !ok{unauthorized(w);return};var in UpdateRiderLocationInput;if err:=decode(r,&in);err!=nil{bad(w);return};v,err:=h.service.UpdateRiderLocation(r.Context(),id,in);if err!=nil{locationError(w,err);return};httpserver.WriteJSON(w,http.StatusOK,v)}
func(h *Handler)nearby(w http.ResponseWriter,r *http.Request){lat,err:=number(r,"latitude");if err!=nil{bad(w);return};lng,err:=number(r,"longitude");if err!=nil{bad(w);return};radius,err:=number(r,"radius_meters");if err!=nil{bad(w);return};limit:=20;if raw:=r.URL.Query().Get("limit");raw!=""{limit,err=strconv.Atoi(raw);if err!=nil{bad(w);return}};v,err:=h.service.Nearby(r.Context(),lat,lng,radius,limit);if err!=nil{locationError(w,err);return};httpserver.WriteJSON(w,http.StatusOK,v)}
func(h *Handler)listPlaces(w http.ResponseWriter,r *http.Request){id,ok:=userID(r);if !ok{unauthorized(w);return};v,err:=h.service.ListPlaces(r.Context(),id);if err!=nil{locationError(w,err);return};httpserver.WriteJSON(w,http.StatusOK,v)}
func(h *Handler)createPlace(w http.ResponseWriter,r *http.Request){id,ok:=userID(r);if !ok{unauthorized(w);return};var in SavePlaceInput;if err:=decode(r,&in);err!=nil{bad(w);return};v,err:=h.service.CreatePlace(r.Context(),id,in);if err!=nil{locationError(w,err);return};httpserver.WriteJSON(w,http.StatusCreated,v)}
func(h *Handler)deletePlace(w http.ResponseWriter,r *http.Request){id,ok:=userID(r);if !ok{unauthorized(w);return};err:=h.service.DeletePlace(r.Context(),id,r.PathValue("placeID"));if err!=nil{locationError(w,err);return};w.WriteHeader(http.StatusNoContent)}
func userID(r *http.Request)(string,bool){c,ok:=identity.ClaimsFromContext(r.Context());return c.Subject,ok}
func number(r *http.Request,key string)(float64,error){return strconv.ParseFloat(r.URL.Query().Get(key),64)}
func decode(r *http.Request,dst any)error{d:=json.NewDecoder(io.LimitReader(r.Body,1<<20));d.DisallowUnknownFields();if err:=d.Decode(dst);err!=nil{return err};if err:=d.Decode(&struct{}{});err!=io.EOF{return errors.New("multiple values")};return nil}
func locationError(w http.ResponseWriter,err error){code:=http.StatusBadRequest;msg:="location_operation_failed";if errors.Is(err,ErrNotEligible){code=http.StatusForbidden;msg="rider_not_location_eligible"};if errors.Is(err,ErrNotFound){code=http.StatusNotFound;msg="not_found"};httpserver.WriteJSON(w,code,map[string]string{"error":msg})}
func bad(w http.ResponseWriter){httpserver.WriteJSON(w,http.StatusBadRequest,map[string]string{"error":"invalid_request"})}
func unauthorized(w http.ResponseWriter){httpserver.WriteJSON(w,http.StatusUnauthorized,map[string]string{"error":"invalid_session"})}
