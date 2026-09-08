package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alumasinde/tuma254-api/internal/users/services"
	"github.com/google/uuid"
)

type Handler struct{ svc *services.Service }
func New(svc *services.Service)*Handler{return &Handler{svc:svc}}
func (h *Handler) Get(w http.ResponseWriter,r *http.Request){id,ok:=uuidFromContext(r);if !ok{http.Error(w,"unauthorized",401);return};p,err:=h.svc.Get(r.Context(),id);if err!=nil{http.Error(w,"profile not found",404);return};write(w,200,p)}
func (h *Handler) Update(w http.ResponseWriter,r *http.Request){id,ok:=uuidFromContext(r);if !ok{http.Error(w,"unauthorized",401);return};var in struct{AvatarURL string `json:"avatar_url"`};if !decode(w,r,&in){return};p,err:=h.svc.Update(r.Context(),id,strings.TrimSpace(in.AvatarURL));if err!=nil{http.Error(w,"profile update failed",400);return};write(w,200,p)}
type userIDKey struct{}
func WithUserID(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){v:=r.Context().Value("tuma254.user_id");id,ok:=v.(string);if !ok{http.Error(w,"unauthorized",401);return};next.ServeHTTP(w,r.WithContext(r.Context()))})}
func uuidFromContext(r *http.Request)(uuid.UUID,bool){v:=r.Context().Value("tuma254.user_id");s,ok:=v.(string);if !ok{return uuid.Nil,false};id,err:=uuid.Parse(s);return id,err==nil}
func decode(w http.ResponseWriter,r *http.Request,dst any)bool{d:=json.NewDecoder(http.MaxBytesReader(w,r.Body,1<<20));d.DisallowUnknownFields();if err:=d.Decode(dst);err!=nil{http.Error(w,"invalid request",400);return false};return true}
func write(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json; charset=utf-8");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)}
