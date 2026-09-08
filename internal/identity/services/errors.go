package services

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken = errors.New("invalid token")
	ErrInvalidOTP = errors.New("invalid verification code")
	ErrExpiredOTP = errors.New("verification code expired")
	ErrOTPLocked = errors.New("verification code locked")
	ErrInvalidRegistration = errors.New("invalid registration data")
	ErrInvalidPhone = errors.New("invalid phone")
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	ErrPhoneAlreadyRegistered = errors.New("phone already registered")
	ErrAccountInactive = errors.New("account inactive")
	ErrPhoneNotVerified = errors.New("phone not verified")
	ErrSMSDelivery = errors.New("sms delivery failed")
	ErrLoginRateLimited = errors.New("too many login attempts")
}
