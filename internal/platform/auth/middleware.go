package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type Validator struct {
	accessSecret []byte
}

func NewValidator(accessSecret string) *Validator {
	return &Validator{accessSecret: []byte(accessSecret)}
}

func (v *Validator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if header == "" {
			writeError(w, http.StatusUnauthorized, ErrMissingToken)
			return
		}

		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, ErrInvalidToken)
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(
			parts[1],
			claims,
			func(t *jwt.Token) (interface{}, error) {
				return v.accessSecret, nil
			},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithExpirationRequired(),
		)
		if err != nil || token == nil || !token.Valid || claims["typ"] != "access" {
			writeError(w, http.StatusUnauthorized, ErrInvalidToken)
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok || strings.TrimSpace(userID) == "" {
			writeError(w, http.StatusUnauthorized, ErrInvalidToken)
			return
		}

		principal := Principal{UserID: userID}
		if roles, ok := claims["roles"].([]interface{}); ok {
			for _, role := range roles {
				if value, ok := role.(string); ok && value != "" {
					principal.Roles = append(principal.Roles, value)
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
	})
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
