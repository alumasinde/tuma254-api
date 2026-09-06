package identity

import (
	"errors"
	"strings"
	"unicode"
	"golang.org/x/crypto/bcrypt"
)

var ErrWeakPassword = errors.New("password must be at least 12 characters and contain a letter and a number")

func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil { return "", err }
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func ValidatePassword(password string) error {
	if len(strings.TrimSpace(password)) < 12 { return ErrWeakPassword }
	var letter, number bool
	for _, r := range password {
		letter = letter || unicode.IsLetter(r)
		number = number || unicode.IsNumber(r)
	}
	if !letter || !number { return ErrWeakPassword }
	return nil
}
