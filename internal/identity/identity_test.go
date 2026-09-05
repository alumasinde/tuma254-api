package identity

import "testing"

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
