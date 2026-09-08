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

type Service struct { repo repositories.Repository; sender SMSSender; secret []byte; otpSecret []byte; accessTTL time.Duration; refreshTTL time.Duration; otpPolicy OTPPolicy }

func New(r repositories.Repository, sender SMSSender, jwtSecret, otpSecret string, accessTTL, refreshTTL time.Duration, policy OTPPolicy) *Service { return &Service{repo:r,sender:sender,secret:[]byte(jwtSecret),otpSecret:[]byte(otpSecret),accessTTL:accessTTL,refreshTTL:refreshTTL,otpPolicy:policy} }

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

func (s *Service) Login(ctx context.Context,in dtos.LoginRequest,ua,ip string)(dtos.AuthResponse,error){u,hash,err:=s.repo.FindByEmail(ctx,strings.ToLower(strings.TrimSpace(in.Email)));if err!=nil||!u.Active||u.PhoneVerifiedAt==nil||bcrypt.CompareHashAndPassword([]byte(hash),[]byte(in.Password))!=nil{return dtos.AuthResponse{},ErrInvalidCredentials};return s.issue(ctx,u,ua,ip)}

func (s *Service) Refresh(ctx context.Context,token,ua,ip string)(dtos.AuthResponse,error){token=strings.TrimSpace(token);if token==""{return dtos.AuthResponse{},ErrInvalidToken};now:=time.Now();replacement,replacementHash,err:=generateRefreshToken();if err!=nil{return dtos.AuthResponse{},err};currentHash:=sha256.Sum256([]byte(token));user,err:=s.repo.RotateSession(ctx,repositories.RotateSessionParams{CurrentTokenHash:currentHash[:],ReplacementTokenHash:replacementHash,ReplacementExpiresAt:now.Add(s.refreshTTL),UserAgent:ua,IPAddress:ip});if err!=nil{return dtos.AuthResponse{},ErrInvalidToken};access,err:=s.createAccessToken(user,now);if err!=nil{return dtos.AuthResponse{},err};return dtos.AuthResponse{AccessToken:access,RefreshToken:replacement,TokenType:"Bearer",ExpiresIn:int64(s.accessTTL.Seconds())},nil}

func (s *Service) Logout(ctx context.Context,token string)error{h:=sha256.Sum256([]byte(token));return s.repo.RevokeSession(ctx,h[:])}
func (s *Service) Me(ctx context.Context,id string)(models.User,error){userID,err:=uuid.Parse(id);if err!=nil{return models.User{},ErrInvalidToken};return s.repo.FindByID(ctx,userID)}

func (s *Service) sendPhoneVerification(ctx context.Context,user models.User)error{code,err:=generateOTP();if err!=nil{return err};params:=repositories.IssueOTPParams{UserID:user.ID,Phone:user.Phone,Purpose:models.OTPPurposePhoneVerification,CodeHash:s.hashOTP(code),ExpiresAt:time.Now().Add(s.otpPolicy.TTL),MaxAttempts:s.otpPolicy.MaxAttempts,Cooldown:s.otpPolicy.ResendCooldown,ResendWindow:s.otpPolicy.ResendWindow,MaxResends:s.otpPolicy.MaxResends};if err=s.repo.IssueOTP(ctx,params);err!=nil{return err};if s.sender==nil{_ = s.repo.RevokeActiveOTP(ctx,user.ID,models.OTPPurposePhoneVerification);return errors.New("sms sender is not configured")};if err=s.sender.Send(ctx,SMSMessage{To:user.Phone,Body:fmt.Sprintf("Your Tuma254 verification code is %s. It expires in %d minutes.",code,int(s.otpPolicy.TTL.Minutes()))});err!=nil{_ = s.repo.RevokeActiveOTP(ctx,user.ID,models.OTPPurposePhoneVerification);return ErrSMSDelivery};return nil}

func (s *Service) hashOTP(code string)[]byte{h:=hmac.New(sha256.New,s.otpSecret);_,_=h.Write([]byte(code));return h.Sum(nil)}
func generateOTP()(string,error){n,err:=rand.Int(rand.Reader,big.NewInt(1000000));if err!=nil{return "",err};return fmt.Sprintf("%06d",n.Int64()),nil}
func validOTPCode(v string)bool{if len(v)!=6{return false};for _,c:=range v{if c<'0'||c>'9'{return false}};return true}
func normalizePhone(v string) string {
	v = strings.TrimSpace(v)
	v = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(v)

	switch {
	case strings.HasPrefix(v, "+254"):
		v = v[1:]
	case strings.HasPrefix(v, "254"):
		// already in Kenyan country-code format
	case strings.HasPrefix(v, "0"):
		v = "254" + v[1:]
	case strings.HasPrefix(v, "7") || strings.HasPrefix(v, "1"):
		v = "254" + v
	default:
		return ""
	}

	if len(v) != 12 || (v[3] != '7' && v[3] != '1') {
		return ""
	}
	for _, digit := range v {
		if digit < '0' || digit > '9' {
			return ""
		}
	}
	return "+" + v
}
func validPhone(v string) bool {
	if len(v) != 13 || !strings.HasPrefix(v, "+254") {
		return false
	}
	if v[4] != '7' && v[4] != '1' {
		return false
	}
	for _, digit := range v[1:] {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}
func validEmail(v string)bool{if len(v)<5||len(v)>254{return false};at:=strings.LastIndex(v,"@");return at>0&&at<len(v)-1&&strings.Contains(v[at+1:],".")}

func (s *Service) issue(ctx context.Context,user models.User,userAgent,ip string)(dtos.AuthResponse,error){now:=time.Now();access,err:=s.createAccessToken(user,now);if err!=nil{return dtos.AuthResponse{},err};refresh,refreshHash,err:=generateRefreshToken();if err!=nil{return dtos.AuthResponse{},err};if err=s.repo.CreateSession(ctx,repositories.CreateSessionParams{UserID:user.ID,TokenHash:refreshHash,ExpiresAt:now.Add(s.refreshTTL),UserAgent:userAgent,IPAddress:ip});err!=nil{return dtos.AuthResponse{},err};return dtos.AuthResponse{AccessToken:access,RefreshToken:refresh,TokenType:"Bearer",ExpiresIn:int64(s.accessTTL.Seconds())},nil}

func (s *Service) createAccessToken(user models.User,now time.Time)(string,error){claims:=jwt.MapClaims{"sub":user.ID.String(),"roles":user.Roles,"iat":now.Unix(),"exp":now.Add(s.accessTTL).Unix(),"iss":"tuma254"};return jwt.NewWithClaims(jwt.SigningMethodHS256,claims).SignedString(s.secret)}
func generateRefreshToken()(string,[]byte,error){raw:=make([]byte,48);if _,err:=rand.Read(raw);err!=nil{return "",nil,err};token:=base64.RawURLEncoding.EncodeToString(raw);hash:=sha256.Sum256([]byte(token));return token,hash[:],nil}

func (s *Service) ParseAccess(raw string)(string,[]string,error){p,err:=jwt.Parse(raw,func(t *jwt.Token)(any,error){if t.Method.Alg()!=jwt.SigningMethodHS256.Alg(){return nil,ErrInvalidToken};return s.secret,nil},jwt.WithIssuer("tuma254"));if err!=nil||!p.Valid{return "",nil,ErrInvalidToken};c,ok:=p.Claims.(jwt.MapClaims);if !ok{return "",nil,ErrInvalidToken};id,_:=c["sub"].(string);if _,err:=uuid.Parse(id);err!=nil{return "",nil,ErrInvalidToken};roles:=[]string{};if xs,ok:=c["roles"].([]any);ok{for _,x:=range xs{if v,ok:=x.(string);ok{roles=append(roles,v)}}};return id,roles,nil}
