package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/alumasinde/tuma254-api/internal/identity/models"
	"github.com/alumasinde/tuma254-api/internal/identity/repositories"
)

var (
	emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	phonePattern = regexp.MustCompile(`^\+254(?:1|7)\d{8}$`)
	otpPattern   = regexp.MustCompile(`^\d{6}$`)
)

func normalizePhone(phone string) string {
	phone = strings.ReplaceAll(strings.TrimSpace(phone), " ", "")
	switch {
	case strings.HasPrefix(phone, "07") || strings.HasPrefix(phone, "01"):
		return "+254" + phone[1:]
	case strings.HasPrefix(phone, "254"):
		return "+" + phone
	default:
		return phone
	}
}

func validEmail(email string) bool { return emailPattern.MatchString(strings.TrimSpace(email)) }
func validPhone(phone string) bool { return phonePattern.MatchString(phone) }
func validOTPCode(code string) bool { return otpPattern.MatchString(strings.TrimSpace(code)) }

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil { return "", err }
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s *Service) sendPhoneVerification(ctx context.Context, user models.User) error {
	code, err := generateOTP()
	if err != nil { return err }
	if err := s.repo.IssueOTP(ctx, repositories.IssueOTPParams{
		UserID: user.ID,
		Phone: user.Phone,
		Purpose: models.OTPPurposePhoneVerification,
		CodeHash: s.hashOTP(code),
		ExpiresAt: time.Now().Add(s.otpPolicy.TTL),
		MaxAttempts: s.otpPolicy.MaxAttempts,
		Cooldown: s.otpPolicy.ResendCooldown,
		ResendWindow: s.otpPolicy.ResendWindow,
		MaxResends: s.otpPolicy.MaxResends,
	}); err != nil { return err }
	if err := s.sender.Send(ctx, SMSMessage{To: user.Phone, Body: "Your Tuma254 verification code is " + code}); err != nil { return ErrSMSDelivery }
	return nil
}
