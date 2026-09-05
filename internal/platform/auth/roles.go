package auth

import "net/http"

func RequireRole(role string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := PrincipalFromContext(r.Context())
		if err != nil {
			writeError(w, http.StatusUnauthorized, ErrMissingToken)
			return
		}
		if !HasRole(principal, role) {
			writeError(w, http.StatusForbidden, ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := PrincipalFromContext(r.Context())
			if err != nil {
				writeError(w, http.StatusUnauthorized, ErrMissingToken)
				return
			}
			if !HasAnyRole(principal, roles...) {
				writeError(w, http.StatusForbidden, ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
