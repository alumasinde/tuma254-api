package services

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
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
	TTL time.Duration
	ResendCooldown time.Duration
	ResendWindow time.Duration
	MaxResends int
	MaxAttempts int
}

type Service struct {
	repo repositories.Repository
	sender SMSSender
	secret []byte
	otpSecret []byte
	accessTTL time.Duration
	refreshTTL time.Duration
	otpPolicy OTPPolicy
	loginLimiter LoginLimiter
}

func New(r repositories.Repository, sender SMSSender, jwtSecret, otpSecret string, accessTTL, refreshTTL time.Duration, policy OTPPolicy) *Service {
	return &Service{repo:r,sender:sender,secret:[]byte(jwtSecret),otpSecret:[]byte(otpSecret),accessTTL:accessTTL,refreshTTL:refreshTTL,otpPolicy:policy}
}
func (s *Service) SetLoginLimiter(limiter LoginLimiter) { s.loginLimiter = limiter }

func (s *Service) Register(ctx context.Context, in dtos.RegisterRequest) (dtos.RegisterResponse, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email)); in.Phone = normalizePhone(in.Phone)
	in.FirstName = strings.TrimSpace(in.FirstName); in.LastName = strings.TrimSpace(in.LastName)
	if !validEmail(in.Email)||!validPhone(in.Phone)||in.FirstName==""||len(in.FirstName)>100||in.LastName==""||len(in.LastName)>100||len(in.Password)<12||len(in.Password)>128 { return dtos.RegisterResponse{}, ErrInvalidRegistration }
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost); if err != nil { return dtos.RegisterResponse{}, err }
	user, err := s.repo.CreateUser(ctx, repositories.CreateUserParams{Email:in.Email,Phone:in.Phone,FirstName:in.FirstName,LastName:in.LastName,PasswordHash:string(hash)})
	if err != nil { return dtos.RegisterResponse{}, mapRegistrationError(err) }
	if err := s.sendPhoneVerification(ctx,user); err != nil { return dtos.RegisterResponse{},err }
	return dtos.RegisterResponse{PhoneVerificationRequired:true,ResendAvailableInSeconds:int64(s.otpPolicy.ResendCooldown.Seconds())},nil
}

func (s *Service) ResendPhoneVerification(ctx context.Context, phone string) error {
	phone=normalizePhone(phone); if !validPhone(phone) { return ErrInvalidPhone }
	user,err:=s.repo.FindByPhone(ctx,phone); if errors.Is(err,pgx.ErrNoRows)||user.PhoneVerifiedAt!=nil { return nil }; if err!=nil{return err}
	return s.sendPhoneVerification(ctx,user)
}

func (s *Service) VerifyPhone(ctx context.Context,in dtos.VerifyPhoneRequest,ua,ip string)(dtos.AuthResponse,error){
	phone:=normalizePhone(in.Phone); if !validPhone(phone)||!validOTPCode(in.Code){return dtos.AuthResponse{},ErrInvalidOTP}
	result,err:=s.repo.VerifyOTP(ctx,phone,models.OTPPurposePhoneVerification,s.hashOTP(in.Code));if err!=nil{return dtos.AuthResponse{},err}
	if result.Expired{return dtos.AuthResponse{},ErrExpiredOTP};if result.Exhausted{return dtos.AuthResponse{},ErrOTPLocked};if !result.Verified{return dtos.AuthResponse{},ErrInvalidOTP}
	return s.issue(ctx,result.User,ua,ip)
}

func (s *Service) Login(ctx context.Context,in dtos.LoginRequest,ua,ip string)(dtos.AuthResponse,error){
	email:=strings.ToLower(strings.TrimSpace(in.Email)); accountKey:="account:"+email; ipKey:="ip:"+strings.TrimSpace(ip)
	if s.loginLimiter!=nil&&(!s.loginLimiter.Allow(ctx,accountKey)||!s.loginLimiter.Allow(ctx,ipKey)){return dtos.AuthResponse{},ErrLoginRateLimited}
	const dummy="$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	if !validEmail(email)||strings.TrimSpace(in.Password)==""{_ = bcrypt.CompareHashAndPassword([]byte(dummy),[]byte(in.Password));s.recordLoginFailure(ctx,accountKey,ipKey);return dtos.AuthResponse{},ErrInvalidCredentials}
	u,passwordHash,err:=s.repo.FindByEmail(ctx,email)
	if err!=nil||!u.Active||u.PhoneVerifiedAt==nil{_ = bcrypt.CompareHashAndPassword([]byte(dummy),[]byte(in.Password));s.recordLoginFailure(ctx,accountKey,ipKey);return dtos.AuthResponse{},ErrInvalidCredentials}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash),[]byte(in.Password))!=nil{s.recordLoginFailure(ctx,accountKey,ipKey);return dtos.AuthResponse{},ErrInvalidCredentials}
	if s.loginLimiter!=nil{s.loginLimiter.Reset(ctx,accountKey);s.loginLimiter.Reset(ctx,ipKey)}
	return s.issue(ctx,u,ua,ip)
}
func (s *Service) recordLoginFailure(ctx context.Context, keys ...string){if s.loginLimiter!=nil{for _,k:=range keys{s.loginLimiter.RecordFailure(ctx,k)}}}

func (s *Service) Refresh(ctx context.Context, token, ua, ip string)(dtos.AuthResponse,error){
	token=strings.TrimSpace(token);if token==""{return dtos.AuthResponse{},ErrInvalidToken}
	current:=hashRefreshToken(token); replacement,replacementHash,err:=generateRefreshToken();if err!=nil{return dtos.AuthResponse{},err}
	u,err:=s.repo.RotateSession(ctx,repositories.RotateSessionParams{CurrentTokenHash:current,ReplacementTokenHash:replacementHash,ReplacementExpiresAt:time.Now().Add(s.refreshTTL),UserAgent:ua,IPAddress:ip});if err!=nil{return dtos.AuthResponse{},ErrInvalidToken}
	return s.issueWithRefresh(ctx,u,replacement,ua,ip,false)
}
func (s *Service) Logout(ctx context.Context, token string) error { if strings.TrimSpace(token)=="" { return nil }; return s.repo.RevokeSession(ctx,hashRefreshToken(token)) }

func (s *Service) issue(ctx context.Context,u models.User,ua,ip string)(dtos.AuthResponse,error){
	refresh,hash,err:=generateRefreshToken();if err!=nil{return dtos.AuthResponse{},err}
	if err=s.repo.CreateSession(ctx,repositories.CreateSessionParams{UserID:u.ID,TokenHash:hash,ExpiresAt:time.Now().Add(s.refreshTTL),UserAgent:ua,IPAddress:ip});err!=nil{return dtos.AuthResponse{},err}
	return s.issueWithRefresh(ctx,u,refresh,ua,ip,true)
}
func (s *Service) issueWithRefresh(_ context.Context,u models.User,refresh,_,_ string,_ bool)(dtos.AuthResponse,error){
	now:=time.Now();claims:=jwt.MapClaims{"sub":u.ID.String(),"roles":u.Roles,"iat":now.Unix(),"exp":now.Add(s.accessTTL).Unix()}
	token:=jwt.NewWithClaims(jwt.SigningMethodHS256,claims);access,err:=token.SignedString(s.secret);if err!=nil{return dtos.AuthResponse{},err}
	return dtos.AuthResponse{AccessToken:access,RefreshToken:refresh,TokenType:"Bearer",ExpiresIn:int64(s.accessTTL.Seconds())},nil
}

func (s *Service) ParseAccess(raw string)(string,[]string,error){
	claims:=jwt.MapClaims{};token,err:=jwt.ParseWithClaims(raw,claims,func(t *jwt.Token)(any,error){if _,ok:=t.Method.(*jwt.SigningMethodHMAC);!ok{return nil,ErrInvalidToken};return s.secret,nil})
	if err!=nil||!token.Valid{return "",nil,ErrInvalidToken}
	sub,ok:=claims["sub"].(string);if !ok||sub==""{return "",nil,ErrInvalidToken}
	var roles []string;if values,ok:=claims["roles"].([]any);ok{for _,v:=range values{if role,ok:=v.(string);ok{roles=append(roles,role)}}}
	return sub,roles,nil
}
func (s *Service) Me(ctx context.Context,userID string)(models.User,error){id,err:=uuid.Parse(userID);if err!=nil{return models.User{},ErrInvalidToken};return s.repo.FindByID(ctx,id)}
func (s *Service) hashOTP(code string) []byte { mac:=hmac.New(sha256.New,s.otpSecret);mac.Write([]byte(code));return mac.Sum(nil) }
func generateRefreshToken()(string,[]byte,error){raw:=make([]byte,32);if _,err:=rand.Read(raw);err!=nil{return "",nil,err};token:=base64.RawURLEncoding.EncodeToString(raw);return token,hashRefreshToken(token),nil}
func hashRefreshToken(token string) []byte { sum:=sha256.Sum256([]byte(token));return sum[:] }
