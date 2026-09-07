package services

import (
	"strings"
	"testing"
)

func TestGenerateOTP(t *testing.T) {
	code, err := generateOTP()
	if err != nil { t.Fatalf("generateOTP: %v", err) }
	if len(code) != 6 || !validOTPCode(code) {
		t.Fatalf("invalid OTP generated: %q", code)
	}
}

func TestOTPHashUsesSecret(t *testing.T) {
	s1 := &Service{otpSecret: []byte(strings.Repeat("a", 32))}
	s2 := &Service{otpSecret: []byte(strings.Repeat("b", 32))}
	h1 := s1.hashOTP("123456")
	h2 := s1.hashOTP("123456")
	h3 := s2.hashOTP("123456")
	if string(h1) != string(h2) { t.Fatal("same secret/code must hash deterministically") }
	if string(h1) == string(h3) { t.Fatal("different OTP secrets must produce different hashes") }
}

func TestPhoneValidation(t *testing.T) {
	for _, v := range []string{"+254712345678", "+12025550123"} {
		if !validPhone(v) { t.Fatalf("expected valid phone %q", v) }
	}
	for _, v := range []string{"0712345678", "+254712", "+25471234abcd", ""} {
		if validPhone(v) { t.Fatalf("expected invalid phone %q", v) }
	}
}