package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alumasinde/tuma254-api/internal/riders/models"
	"github.com/alumasinde/tuma254-api/internal/riders/services"
	"github.com/google/uuid"
)

type Handler struct{ svc *services.Service }
func New(svc *services.Service)*Handler{return &Handler{svc:svc}}
func (h *Handler) GetMe(w http.ResponseWriter,r *http.Request){id,ok:=currentUser(r);if !ok{http.Error(w,"unauthorized",401);return};p,err:=h.svc.Get(r.Context(),id);if errors.Is(err,services.ErrInvalidState){http.Error(w,"invalid rider state",409);return};if err!=nil{http.Error(w,"rider profile not found",404);return};write(w,200,p)}
func(h *Handler) Create(w http.ResponseWriter,r *http.Request){id,ok:=currentUser(r);if !ok{http.Error(w,"unauthorized",401);return};p,err:=h.svc.CreateApplication(r.Context(),id);if err!=nil{http.Error(w,"rider application failed",400);return};write(w,201,p)}
func(h *Handler) Submit(w http.ResponseWriter,r *http.Request){id,ok:=currentUser(r);if !ok{http.Error(w,"unauthorized",401);return};p,err:=h.svc.SubmitApplication(r.Context(),id);if err!=nil{http.Error(w,err.Error(),409);return};write(w,200,p)}
func(h *Handler) SetAvailability(w http.ResponseWriter,r *http.Request){id,ok:=currentUser(r);if !ok{http.Error(w,"unauthorized",401);return};var in struct{Availability models.Availability `json:"availability"`};if !decode(w,r,&in){return};p,err:=h.svc.SetAvailability(r.Context(),id,in.Availability);if err!=nil{http.Error(w,err.Error(),409);return};write(w,200,p)}
func(h *Handler) AddVehicle(w http.ResponseWriter,r *http.Request){id,ok:=currentUser(r);if !ok{http.Error(w,"unauthorized",401);return};var in models.Vehicle;if !decode(w,r,&in){return};v,err:=h.svc.AddVehicle(r.Context(),id,in);if err!=nil{http.Error(w,err.Error(),400);return};write(w,201,v)}
func(h *Handler) ListVehicles(w http.ResponseWriter,r *http.Request){id,ok:=currentUser(r);if !ok{http.Error(w,"unauthorized",401);return};v,err:=h.svc.ListVehicles(r.Context(),id);if err!=nil{http.Error(w,"rider profile not found",404);return};write(w,200,v)}
func(h *Handler) SetActiveVehicle(w http.ResponseWriter,r *http.Request){id,ok:=currentUser(r);if !ok{http.Error(w,"unauthorized",401);return};vid,err:=uuid.Parse(r.PathValue("vehicleID"));if err!=nil{http.Error(w,"invalid vehicle id",400);return};v,err:=h.svc.SetActiveVehicle(r.Context(),id,vid);if err!=nil{http.Error(w,err.Error(),409);return};write(w,200,v)}
func currentUser(r *http.Request)(uuid.UUID,bool){v:=r.Context().Value("tuma254.user_id");s,ok:=v.(string);if !ok{return uuid.Nil,false};id,err:=uuid.Parse(s);return id,err==nil}
func decode(w http.ResponseWriter,r *http.Request,dst any)bool{d:=json.NewDecoder(http.MaxBytesReader(w,r.Body,1<<20));d.DisallowUnknownFields();if err:=d.Decode(dst);err!=nil{http.Error(w,"invalid request",400);return false};return true}
func write(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json; charset=utf-8");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)}
