package services

import "testing"

func TestNormalizePhone(t *testing.T) {
	tests := map[string]string{
		"0712345678":      "+254712345678",
		"712345678":       "+254712345678",
		"254712345678":    "+254712345678",
		"+254712345678":   "+254712345678",
		" 0712 345 678 ":  "+254712345678",
		"0112345678":      "+254112345678",
	}

	for input, expected := range tests {
		if actual := normalizePhone(input); actual != expected {
			t.Fatalf("normalizePhone(%q) = %q, expected %q", input, actual, expected)
		}
	}
}

func TestNormalizePhoneRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"", "abc", "254612345678", "+255712345678", "071234567"} {
		if actual := normalizePhone(input); actual != "" {
			t.Fatalf("normalizePhone(%q) = %q, expected empty value", input, actual)
		}
	}
}

func TestOTPValidation(t *testing.T) {
	if !validOTPCode("012345") {
		t.Fatal("expected six-digit OTP to be valid")
	}
	for _, code := range []string{"12345", "1234567", "12a456", ""} {
		if validOTPCode(code) {
			t.Fatalf("expected %q to be invalid", code)
		}
	}
}

func TestRefreshTokenGeneration(t *testing.T) {
	token, hash, err := generateRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || len(hash) == 0 {
		t.Fatal("expected token and hash")
	}
	other, _, err := generateRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == other {
		t.Fatal("expected unique refresh tokens")
	}
}
