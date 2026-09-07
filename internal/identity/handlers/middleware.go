package handlers

import (
 "context"
 "net/http"
 "strings"
)

type userIDKey struct{}

func(h *Handler)RequireAuth(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){v:=r.Header.Get("Authorization");parts:=strings.Fields(v);if len(parts)!=2||!strings.EqualFold(parts[0],"Bearer"){http.Error(w,"unauthorized",http.StatusUnauthorized);return};id,_,err:=h.svc.ParseAccess(parts[1]);if err!=nil{http.Error(w,"unauthorized",http.StatusUnauthorized);return};next.ServeHTTP(w,r.WithContext(context.WithValue(r.Context(),userIDKey{},id)))})}
