package identity

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type principalKey struct{}

type Principal struct {
	UserID string
	Roles  []string
}

func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			errJSON(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if raw == "" {
			errJSON(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(
			raw,
			claims,
			func(token *jwt.Token) (interface{}, error) {
				return []byte(s.accessSecret), nil
			},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithExpirationRequired(),
		)
		if err != nil || !token.Valid || claims["typ"] != "access" {
			errJSON(w, http.StatusUnauthorized, "invalid access token")
			return
		}

		userID, _ := claims["sub"].(string)
		if userID == "" {
			errJSON(w, http.StatusUnauthorized, "invalid access token")
			return
		}

		principal := Principal{UserID: userID}
		if rawRoles, ok := claims["roles"].([]interface{}); ok {
			for _, value := range rawRoles {
				if role, ok := value.(string); ok && role != "" {
					principal.Roles = append(principal.Roles, role)
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, principal)))
	})
}

func PrincipalFromContext(ctx context.Context) (Principal, error) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	if !ok {
		return Principal{}, errors.New("principal missing")
	}
	return principal, nil
}
