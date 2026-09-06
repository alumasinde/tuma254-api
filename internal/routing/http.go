package routing

import (
 "encoding/json"
 "errors"
 "io"
 "net/http"
 httpserver "github.com/alumasinde/tuma254-api/internal/platform/http"
)

type Authenticator interface{RequireAuth(http.HandlerFunc)http.HandlerFunc}
type Handler struct{service *Service;auth Authenticator}
func NewHandler(service *Service,auth Authenticator)*Handler{return &Handler{service:service,auth:auth}}
func(h *Handler)RegisterRoutes(mux *http.ServeMux){mux.HandleFunc("POST /api/v1/routing/route",h.auth.RequireAuth(h.route));mux.HandleFunc("POST /api/v1/routing/matrix",h.auth.RequireAuth(h.matrix))}
func(h *Handler)route(w http.ResponseWriter,r *http.Request){var in RouteRequest;if err:=decode(r,&in);err!=nil{write(w,http.StatusBadRequest,"invalid_request");return};v,err:=h.service.Route(r.Context(),in);if err!=nil{write(w,http.StatusBadGateway,"routing_unavailable");return};httpserver.WriteJSON(w,http.StatusOK,v)}
func(h *Handler)matrix(w http.ResponseWriter,r *http.Request){var in MatrixRequest;if err:=decode(r,&in);err!=nil{write(w,http.StatusBadRequest,"invalid_request");return};v,err:=h.service.Matrix(r.Context(),in);if err!=nil{write(w,http.StatusBadGateway,"routing_unavailable");return};httpserver.WriteJSON(w,http.StatusOK,v)}
func decode(r *http.Request,dst any)error{d:=json.NewDecoder(io.LimitReader(r.Body,1<<20));d.DisallowUnknownFields();if err:=d.Decode(dst);err!=nil{return err};if err:=d.Decode(&struct{}{});err!=io.EOF{return errors.New("multiple JSON values")};return nil}
func write(w http.ResponseWriter,status int,msg string){httpserver.WriteJSON(w,status,map[string]string{"error":msg})}
