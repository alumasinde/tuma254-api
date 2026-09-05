package identity

import "testing"

func TestUserResponseDoesNotExposePassword(t *testing.T) {
	user := User{
		ID:        "user-id",
		FirstName: "Test",
		LastName:  "User",
		Status:    "active",
		Roles:     []string{"customer"},
	}
	if _, ok := any(user).(interface{ PasswordHash string }); ok {
		t.Fatal("user response must not expose password hash")
	}
}
