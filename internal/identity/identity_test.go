package identity

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNormalizeEmail(t *testing.T) {
	if got := normEmail(" Test@Example.COM "); got != "test@example.com" {
		t.Fatal(got)
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := map[string]string{
		"0712 345678":    "+254712345678",
		"254712345678":   "+254712345678",
		"+254712345678":  "+254712345678",
	}
	for input, want := range tests {
		if got := normPhone(input); got != want {
			t.Fatalf("%q: got %q want %q", input, got, want)
		}
	}
}

func TestValidation(t *testing.T) {
	if !validEmail("user@example.com") || validEmail("not-an-email") {
		t.Fatal("email validation failed")
	}
	if !validPhone("+254712345678") || validPhone("0712345678") {
		t.Fatal("phone validation failed")
	}
}

func TestUserJSONFields(t *testing.T) {
	user := User{
		ID:        "id",
		FirstName: "Test",
		LastName:  "User",
		Status:    "active",
		Roles:     []string{"customer"},
	}
	if user.ID == "" || user.Status != "active" || len(user.Roles) != 1 {
		t.Fatal("user response model is incomplete")
	}
}


func TestParseRefreshToken(t *testing.T) {
	svc := New(nil, "access-secret", "refresh-secret", 15*time.Minute, 24*time.Hour)

	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-id",
		"typ": "refresh",
		"iat": time.Now().Add(-time.Minute).Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("refresh-secret"))
	if err != nil {
		t.Fatal(err)
	}

	claims, err := svc.parseRefreshToken(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims["sub"] != "user-id" || claims["typ"] != "refresh" {
		t.Fatal("refresh claims were not preserved")
	}
}

func TestParseRefreshTokenRejectsAccessToken(t *testing.T) {
	svc := New(nil, "access-secret", "refresh-secret", 15*time.Minute, 24*time.Hour)

	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-id",
		"typ": "access",
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("refresh-secret"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.parseRefreshToken(raw); err == nil {
		t.Fatal("access token must not be accepted as refresh token")
	}
}
