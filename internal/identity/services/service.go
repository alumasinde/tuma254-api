package services

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/alumasinde/tuma254-api/internal/identity/dtos"
	"github.com/alumasinde/tuma254-api/internal/identity/models"
	"github.com/alumasinde/tuma254-api/internal/identity/repositories"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type OTPPolicy struct { TTL time.Duration; ResendCooldown time.Duration; ResendWindow time.Duration; MaxResends int; MaxAttempts int }

type Service struct { repo repositories.Repository; sender SMSSender; secret []byte; otpSecret []byte; accessTTL time.Duration; refreshTTL time.Duration; otpPolicy OTPPolicy; loginLimiter LoginLimiter }

func New(r repositories.Repository, sender SMSSender, jwtSecret, otpSecret string, accessTTL, refreshTTL time.Duration, policy OTPPolicy) *Service { return &Service{repo:r,sender:sender,secret:[]byte(jwtSecret),otpSecret:[]byte(otpSecret),accessTTL:accessTTL,refreshTTL:refreshTTL,otpPolicy:policy} }

func (s *Service) SetLoginLimiter(limiter LoginLimiter) { s.loginLimiter = limiter }

func (s *Service) Register(ctx context.Context, in dtos.RegisterRequest)(dtos.RegisterResponse,error){
	in.Email=strings.ToLower(strings.TrimSpace(in.Email));in.Phone=normalizePhone(in.Phone);in.FirstName=strings.TrimSpace(in.FirstName);in.LastName=strings.TrimSpace(in.LastName)
	if !validEmail(in.Email)||!validPhone(in.Phone)||len(in.FirstName)==0||len(in.FirstName)>100||len(in.LastName)==0||len(in.LastName)>100{return dtos.RegisterResponse{},ErrInvalidRegistration}
	if len(in.Password)<12||len(in.Password)>128{return dtos.RegisterResponse{},ErrInvalidRegistration}
	hash,err:=bcrypt.GenerateFromPassword([]byte(in.Password),bcrypt.DefaultCost);if err!=nil{return dtos.RegisterResponse{},err}
	user,err:=s.repo.CreateUser(ctx,repositories.CreateUserParams{Email:in.Email,Phone:in.Phone,FirstName:in.FirstName,LastName:in.LastName,PasswordHash:string(hash)});if err!=nil{return dtos.RegisterResponse{},mapRegistrationError(err)}
	if err:=s.sendPhoneVerification(ctx,user);err!=nil{return dtos.RegisterResponse{},err}
	return dtos.RegisterResponse{PhoneVerificationRequired:true,ResendAvailableInSeconds:int64(s.otpPolicy.ResendCooldown.Seconds())},nil
}

func (s *Service) ResendPhoneVerification(ctx context.Context,phone string)error{phone=normalizePhone(phone);if !validPhone(phone){return ErrInvalidPhone};user,err:=s.repo.FindByPhone(ctx,phone);if err!=nil||user.PhoneVerifiedAt!=nil{return nil};return s.sendPhoneVerification(ctx,user)}

func (s *Service) VerifyPhone(ctx context.Context,in dtos.VerifyPhoneRequest,ua,ip string)(dtos.AuthResponse,error){phone:=normalizePhone(in.Phone);if !validPhone(phone)||!validOTPCode(in.Code){return dtos.AuthResponse{},ErrInvalidOTP};result,err:=s.repo.VerifyOTP(ctx,phone,models.OTPPurposePhoneVerification,s.hashOTP(in.Code));if err!=nil{return dtos.AuthResponse{},err};if result.Expired{return dtos.AuthResponse{},ErrExpiredOTP};if result.Exhausted{return dtos.AuthResponse{},ErrOTPLocked};if !result.Verified{return dtos.AuthResponse{},ErrInvalidOTP};return s.issue(ctx,result.User,ua,ip)}

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
		if s.loginLimiter != nil { s.loginLimiter.RecordFailure(ctx, accountKey); s.loginLimiter.RecordFailure(ctx, ipKey) }
		return dtos.AuthResponse{}, ErrInvalidCredentials
	}

	u, passwordHash, err := s.repo.FindByEmail(ctx, email)
	if err != nil || !u.Active || u.PhoneVerifiedAt == nil {
		_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(in.Password))
		if s.loginLimiter != nil { s.loginLimiter.RecordFailure(ctx, accountKey); s.loginLimiter.RecordFailure(ctx, ipKey) }
		return dtos.AuthResponse{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(in.Password)); err != nil {
		if s.loginLimiter != nil { s.loginLimiter.RecordFailure(ctx, accountKey); s.loginLimiter.RecordFailure(ctx, ipKey) }
		return dtos.AuthResponse{}, ErrInvalidCredentials
	}
	if s.loginLimiter != nil { s.loginLimiter.Reset(ctx, accountKey); s.loginLimiter.Reset(ctx, ipKey) }
	return s.issue(ctx, u, ua, ip)
}
