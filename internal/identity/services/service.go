package services

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/alumasinde/tuma254-api/internal/identity/dtos"
	"github.com/alumasinde/tuma254-api/internal/identity/models"
	"github.com/alumasinde/tuma254-api/internal/identity/repositories"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type OTPPolicy struct {
	TTL            time.Duration
	ResendCooldown time.Duration
	ResendWindow   time.Duration
	MaxResends     int
	MaxAttempts    int
}

type Service struct {
	repo         repositories.Repository
	sender       SMSSender
	secret       []byte
	otpSecret    []byte
	accessTTL    time.Duration
	refreshTTL   time.Duration
	otpPolicy    OTPPolicy
	loginLimiter LoginLimiter
}

func New(r repositories.Repository, sender SMSSender, jwtSecret, otpSecret string, accessTTL, refreshTTL time.Duration, policy OTPPolicy) *Service {
	return &Service{repo: r, sender: sender, secret: []byte(jwtSecret), otpSecret: []byte(otpSecret), accessTTL: accessTTL, refreshTTL: refreshTTL, otpPolicy: policy}
}

func (s *Service) SetLoginLimiter(limiter LoginLimiter) { s.loginLimiter = limiter }

func (s *Service) Register(ctx context.Context, in dtos.RegisterRequest) (dtos.RegisterResponse, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = normalizePhone(in.Phone)
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	if !validEmail(in.Email) || !validPhone(in.Phone) || in.FirstName == "" || len(in.FirstName) > 100 || in.LastName == "" || len(in.LastName) > 100 || len(in.Password) < 12 || len(in.Password) > 128 {
		return dtos.RegisterResponse{}, ErrInvalidRegistration
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return dtos.RegisterResponse{}, err
	}
	user, err := s.repo.CreateUser(ctx, repositories.CreateUserParams{Email: in.Email, Phone: in.Phone, FirstName: in.FirstName, LastName: in.LastName, PasswordHash: string(hash)})
	if err != nil {
		return dtos.RegisterResponse{}, mapRegistrationError(err)
	}
	if err := s.sendPhoneVerification(ctx, user); err != nil {
		return dtos.RegisterResponse{}, err
	}
	return dtos.RegisterResponse{PhoneVerificationRequired: true, ResendAvailableInSeconds: int64(s.otpPolicy.ResendCooldown.Seconds())}, nil
}

func (s *Service) ResendPhoneVerification(ctx context.Context, phone string) error {
	phone = normalizePhone(phone)
	if !validPhone(phone) {
		return ErrInvalidPhone
	}
	user, err := s.repo.FindByPhone(ctx, phone)
	if errors.Is(err, pgx.ErrNoRows) || user.PhoneVerifiedAt != nil {
		return nil
	}
	if err != nil {
		return err
	}
	return s.sendPhoneVerification(ctx, user)
}

func (s *Service) VerifyPhone(ctx context.Context, in dtos.VerifyPhoneRequest, ua, ip string) (dtos.AuthResponse, error) {
	phone := normalizePhone(in.Phone)
	if !validPhone(phone) || !validOTPCode(in.Code) {
		return dtos.AuthResponse{}, ErrInvalidOTP
	}
	result, err := s.repo.VerifyOTP(ctx, phone, models.OTPPurposePhoneVerification, s.hashOTP(in.Code))
	if err != nil {
		return dtos.AuthResponse{}, err
	}
	if result.Expired {
		return dtos.AuthResponse{}, ErrExpiredOTP
	}
	if result.Exhausted {
		return dtos.AuthResponse{}, ErrOTPLocked
	}
	if !result.Verified {
		return dtos.AuthResponse{}, ErrInvalidOTP
	}
	return s.issue(ctx, result.User, ua, ip)
}

func (s *Service) Login(ctx context.Context, in dtos.LoginRequest, ua, ip string) (dtos.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	accountKey := "account:" + email
	ipKey := "ip:" + strings.TrimSpace(ip)
	if s.loginLimiter != nil && (!s.loginLimiter.Allow(ctx, accountKey) || !s.loginLimiter.Allow(ctx, ipKey)) {
		return dtos.AuthResponse{}, ErrLoginRateLimited
	}

	const dummyPasswordHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	if !validEmail(email) || strings.TrimSpace(in.Password) == "" {
		_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(in.Password))
		s.recordLoginFailure(ctx, accountKey, ipKey)
		return dtos.AuthResponse{}, ErrInvalidCredentials
	}

	user, passwordHash, err := s.repo.FindByEmail(ctx, email)
	if err != nil || !user.Active || user.PhoneVerifiedAt == nil {
		_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(in.Password))
		s.recordLoginFailure(ctx, accountKey, ipKey)
		return dtos.AuthResponse{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(in.Password)); err != nil {
		s.recordLoginFailure(ctx, accountKey, ipKey)
		return dtos.AuthResponse{}, ErrInvalidCredentials
	}
	if s.loginLimiter != nil {
		s.loginLimiter.Reset(ctx, accountKey)
		s.loginLimiter.Reset(ctx, ipKey)
	}
	return s.issue(ctx, user, ua, ip)
}

func (s *Service) recordLoginFailure(ctx context.Context, keys ...string) {
	if s.loginLimiter == nil {
		return
	}
	for _, key := range keys {
		s.loginLimiter.RecordFailure(ctx, key)
	}
}

func (s *Service) Refresh(ctx context.Context, token, ua, ip string) (dtos.AuthResponse, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return dtos.AuthResponse{}, ErrInvalidToken
	}
	currentHash := hashRefreshToken(token)
	replacement, replacementHash, err := generateRefreshToken()
	if err != nil {
		return dtos.AuthResponse{}, err
	}
	user, err := s.repo.RotateSession(ctx, repositories.RotateSessionParams{CurrentTokenHash: currentHash, ReplacementTokenHash: replacementHash, ReplacementExpiresAt: time.Now().Add(s.refreshTTL), UserAgent: ua, IPAddress: ip})
	if err != nil {
		return dtos.AuthResponse{}, ErrInvalidToken
	}
	return s.issueAccessResponse(user, replacement)
}

func (s *Service) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return s.repo.RevokeSession(ctx, hashRefreshToken(token))
}

func (s *Service) issue(ctx context.Context, user models.User, ua, ip string) (dtos.AuthResponse, error) {
	refresh, refreshHash, err := generateRefreshToken()
	if err != nil {
		return dtos.AuthResponse{}, err
	}
	if err := s.repo.CreateSession(ctx, repositories.CreateSessionParams{UserID: user.ID, TokenHash: refreshHash, ExpiresAt: time.Now().Add(s.refreshTTL), UserAgent: ua, IPAddress: ip}); err != nil {
		return dtos.AuthResponse{}, err
	}
	return s.issueAccessResponse(user, refresh)
}

func (s *Service) issueAccessResponse(user models.User, refresh string) (dtos.AuthResponse, error) {
	now := time.Now()
	claims := jwt.MapClaims{"sub": user.ID.String(), "roles": user.Roles, "iat": now.Unix(), "exp": now.Add(s.accessTTL).Unix(), "iss": "tuma254"}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	access, err := token.SignedString(s.secret)
	if err != nil {
		return dtos.AuthResponse{}, err
	}
	return dtos.AuthResponse{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer", ExpiresIn: int64(s.accessTTL.Seconds())}, nil
}

func (s *Service) ParseAccess(raw string) (string, []string, error) {
	parsed, err := jwt.Parse(raw, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	}, jwt.WithIssuer("tuma254"))
	if err != nil || !parsed.Valid {
		return "", nil, ErrInvalidToken
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return "", nil, ErrInvalidToken
	}
	sub, ok := claims["sub"].(string)
	if !ok {
		return "", nil, ErrInvalidToken
	}
	if _, err := uuid.Parse(sub); err != nil {
		return "", nil, ErrInvalidToken
	}
	roles := make([]string, 0)
	if values, ok := claims["roles"].([]any); ok {
		for _, value := range values {
			if role, ok := value.(string); ok {
				roles = append(roles, role)
			}
		}
	}
	return sub, roles, nil
}

func (s *Service) Me(ctx context.Context, userID string) (models.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return models.User{}, ErrInvalidToken
	}
	return s.repo.FindByID(ctx, id)
}

func (s *Service) hashOTP(code string) []byte {
	mac := hmac.New(sha256.New, s.otpSecret)
	_, _ = mac.Write([]byte(code))
	return mac.Sum(nil)
}

func generateRefreshToken() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, hashRefreshToken(token), nil
}

func hashRefreshToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
