package identity

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	httpserver "github.com/alumasinde/tuma254-api/internal/platform/http"
	"github.com/alumasinde/tuma254-api/internal/platform/ratelimit"
)

type Handler struct{service *Service;limiter *ratelimit.Limiter}
func NewHandler(service *Service)*Handler{return &Handler{service:service,limiter:ratelimit.New()}}

func(h *Handler)RegisterRoutes(mux *http.ServeMux){
	mux.HandleFunc("POST /api/v1/auth/register",h.register)
	mux.HandleFunc("POST /api/v1/auth/login",h.login)
	mux.HandleFunc("POST /api/v1/auth/refresh",h.refresh)
	mux.HandleFunc("POST /api/v1/auth/logout",h.requireAuth(h.logout))
	mux.HandleFunc("GET /api/v1/auth/me",h.requireAuth(h.me))
}

func(h *Handler)register(w http.ResponseWriter,r *http.Request){
	if !h.allow(w,r,10,time.Minute){return}
	var input RegisterInput
	if err:=decode(r,&input);err!=nil{writeError(w,http.StatusBadRequest,"invalid_request");return}
	result,err:=h.service.Register(r.Context(),input,clientIP(r))
	if err!=nil{
		if IsDuplicate(err){writeError(w,http.StatusConflict,"account_already_exists");return}
		if errors.Is(err,ErrWeakPassword){writeError(w,http.StatusBadRequest,"weak_password");return}
		writeError(w,http.StatusBadRequest,"registration_failed");return
	}
	httpserver.WriteJSON(w,http.StatusCreated,result)
}

func(h *Handler)login(w http.ResponseWriter,r *http.Request){
	if !h.allow(w,r,8,time.Minute){return}
	var input LoginInput
	if err:=decode(r,&input);err!=nil{writeError(w,http.StatusBadRequest,"invalid_request");return}
	result,err:=h.service.Login(r.Context(),input,clientIP(r))
	if err!=nil{writeError(w,http.StatusUnauthorized,"invalid_credentials");return}
	httpserver.WriteJSON(w,http.StatusOK,result)
}

func(h *Handler)refresh(w http.ResponseWriter,r *http.Request){
	if !h.allow(w,r,20,time.Minute){return}
	var input RefreshInput
	if err:=decode(r,&input);err!=nil{writeError(w,http.StatusBadRequest,"invalid_request");return}
	result,err:=h.service.Refresh(r.Context(),input,clientIP(r))
	if errors.Is(err,ErrSessionReuse){writeError(w,http.StatusUnauthorized,"session_reuse_detected");return}
	if err!=nil{writeError(w,http.StatusUnauthorized,"invalid_refresh_token");return}
	httpserver.WriteJSON(w,http.StatusOK,result)
}

func(h *Handler)logout(w http.ResponseWriter,r *http.Request){
	claims,_:=ClaimsFromContext(r.Context())
	if err:=h.service.Logout(r.Context(),claims.SessionID,clientIP(r));err!=nil{writeError(w,http.StatusUnauthorized,"invalid_session");return}
	w.WriteHeader(http.StatusNoContent)
}

func(h *Handler)me(w http.ResponseWriter,r *http.Request){
	claims,_:=ClaimsFromContext(r.Context())
	user,err:=h.service.Me(r.Context(),claims.Subject)
	if err!=nil{writeError(w,http.StatusUnauthorized,"invalid_session");return}
	httpserver.WriteJSON(w,http.StatusOK,user)
}

func(h *Handler)allow(w http.ResponseWriter,r *http.Request,limit int,window time.Duration)bool{
	key:=r.Method+":"+r.URL.Path+":"+clientIP(r)
	if h.limiter.Allow(key,limit,window){return true}
	w.Header().Set("Retry-After","60")
	writeError(w,http.StatusTooManyRequests,"rate_limited")
	return false
}

func decode(r *http.Request,dst any)error{
	decoder:=json.NewDecoder(io.LimitReader(r.Body,1<<20))
	decoder.DisallowUnknownFields()
	if err:=decoder.Decode(dst);err!=nil{return err}
	if err:=decoder.Decode(&struct{}{});err!=io.EOF{return errors.New("multiple JSON values")}
	return nil
}

func writeError(w http.ResponseWriter,status int,code string){httpserver.WriteJSON(w,status,map[string]string{"error":code})}
func clientIP(r *http.Request)string{host,_,err:=net.SplitHostPort(strings.TrimSpace(r.RemoteAddr));if err==nil{return host};return r.RemoteAddr}
