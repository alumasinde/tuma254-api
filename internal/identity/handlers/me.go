package handlers

import (
 "net/http"
 "github.com/alumasinde/tuma254-api/internal/identity/dtos"
)

func(h *Handler)Me(w http.ResponseWriter,r *http.Request){id,ok:=r.Context().Value(userIDKey{}).(string);if !ok{http.Error(w,"unauthorized",http.StatusUnauthorized);return};u,err:=h.svc.Me(r.Context(),id);if err!=nil||!u.Active{http.Error(w,"unauthorized",http.StatusUnauthorized);return};write(w,http.StatusOK,dtos.MeResponse{ID:u.ID.String(),Email:u.Email,Phone:u.Phone,FirstName:u.FirstName,LastName:u.LastName,Roles:u.Roles})}
