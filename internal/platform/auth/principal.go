package auth

import "context"

type Principal struct {
	UserID string
	Roles  []string
}

type contextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, error) {
	principal, ok := ctx.Value(contextKey{}).(Principal)
	if !ok || principal.UserID == "" {
		return Principal{}, ErrPrincipalNotFound
	}
	principal.Roles = append([]string(nil), principal.Roles...)
	return principal, nil
}

func HasRole(principal Principal, role string) bool {
	for _, value := range principal.Roles {
		if value == role {
			return true
		}
	}
	return false
}

func HasAnyRole(principal Principal, roles ...string) bool {
	for _, role := range roles {
		if HasRole(principal, role) {
			return true
		}
	}
	return false
}
