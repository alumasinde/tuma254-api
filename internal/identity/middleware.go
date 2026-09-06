package identity

import(
	"context"
	"net/http"
	"strings"
)

type claimsKey struct{}

func(h *Handler)requireAuth(next http.HandlerFunc)http.HandlerFunc{
	return func(w http.ResponseWriter,r *http.Request){
		header:=strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header,"Bearer "){writeError(w,http.StatusUnauthorized,"missing_access_token");return}
		claims,err:=h.service.tokens.ParseAccessToken(strings.TrimSpace(strings.TrimPrefix(header,"Bearer ")))
		if err!=nil{writeError(w,http.StatusUnauthorized,"invalid_access_token");return}
		if !h.service.ValidateAccessSession(r.Context(),claims.Subject,claims.SessionID){writeError(w,http.StatusUnauthorized,"invalid_session");return}
		next(w,r.WithContext(context.WithValue(r.Context(),claimsKey{},claims)))
	}
}

func ClaimsFromContext(ctx context.Context)(Claims,bool){claims,ok:=ctx.Value(claimsKey{}).(Claims);return claims,ok}

func(h *Handler)RequirePermission(permission string,next http.HandlerFunc)http.HandlerFunc{
	return h.requireAuth(func(w http.ResponseWriter,r *http.Request){
		claims,_:=ClaimsFromContext(r.Context())
		allowed,err:=h.service.HasPermission(r.Context(),claims.Subject,permission)
		if err!=nil||!allowed{writeError(w,http.StatusForbidden,"permission_denied");return}
		next(w,r)
	})
}
