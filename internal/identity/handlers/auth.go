package handlers

import (
 "encoding/json"
 "errors"
 "io"
 "net"
 "net/http"
 "strings"
 "github.com/alumasinde/tuma254-api/internal/identity/dtos"
 "github.com/alumasinde/tuma254-api/internal/identity/services"
)

type Handler struct{svc *services.Service}
func New(s *services.Service)*Handler{return &Handler{svc:s}}
func(h *Handler)Register(w http.ResponseWriter,r *http.Request){var in dtos.RegisterRequest;if !decode(w,r,&in){return};out,err:=h.svc.Register(r.Context(),in,r.UserAgent(),clientIP(r));if err!=nil{http.Error(w,"registration failed",http.StatusBadRequest);return};write(w,http.StatusCreated,out)}
func(h *Handler)Login(w http.ResponseWriter,r *http.Request){var in dtos.LoginRequest;if !decode(w,r,&in){return};out,err:=h.svc.Login(r.Context(),in,r.UserAgent(),clientIP(r));if err!=nil{http.Error(w,"invalid email or password",http.StatusUnauthorized);return};write(w,http.StatusOK,out)}
func(h *Handler)Refresh(w http.ResponseWriter,r *http.Request){var in dtos.RefreshRequest;if !decode(w,r,&in)||strings.TrimSpace(in.RefreshToken)==""{if strings.TrimSpace(in.RefreshToken)==""{http.Error(w,"invalid refresh token",http.StatusUnauthorized)};return};out,err:=h.svc.Refresh(r.Context(),in.RefreshToken,r.UserAgent(),clientIP(r));if err!=nil{http.Error(w,"invalid refresh token",http.StatusUnauthorized);return};write(w,http.StatusOK,out)}
func(h *Handler)Logout(w http.ResponseWriter,r *http.Request){var in dtos.RefreshRequest;if !decode(w,r,&in){return};_ = h.svc.Logout(r.Context(),in.RefreshToken);w.WriteHeader(http.StatusNoContent)}
func decode(w http.ResponseWriter,r *http.Request,v any)bool{r.Body=http.MaxBytesReader(w,r.Body,1<<20);d:=json.NewDecoder(r.Body);d.DisallowUnknownFields();if err:=d.Decode(v);err!=nil{http.Error(w,"invalid request",http.StatusBadRequest);return false};var extra any;if err:=d.Decode(&extra);!errors.Is(err,ioEOF()){http.Error(w,"invalid request",http.StatusBadRequest);return false};return true}
func ioEOF()error{return io.EOF}
func write(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json; charset=utf-8");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)}
func clientIP(r *http.Request)string{host,_,err:=net.SplitHostPort(r.RemoteAddr);if err==nil{return host};return r.RemoteAddr}
