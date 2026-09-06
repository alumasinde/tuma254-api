package identity

import (
	"context"
	"errors"
	"strings"
	"time"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	ErrInvalidCredentials=errors.New("invalid credentials")
	ErrInvalidRefresh=errors.New("invalid refresh token")
	ErrSessionReuse=errors.New("refresh token reuse detected")
)

type Service struct {
	repo *Repository
	tokens *TokenManager
	refreshTokenTTL time.Duration
}

type RegisterInput struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	Password string `json:"password"`
	DeviceID string `json:"device_id"`
	DeviceName string `json:"device_name"`
}

type LoginInput struct {
	Identifier string `json:"identifier"`
	Password string `json:"password"`
	DeviceID string `json:"device_id"`
	DeviceName string `json:"device_name"`
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token"`
	DeviceID string `json:"device_id"`
	DeviceName string `json:"device_name"`
}

type AuthResult struct {
	User PublicUser `json:"user"`
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType string `json:"token_type"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewService(repo *Repository,tokens *TokenManager,refreshTokenTTL time.Duration)*Service{
	return &Service{repo:repo,tokens:tokens,refreshTokenTTL:refreshTokenTTL}
}

func (s *Service) Register(ctx context.Context,input RegisterInput,ip string)(AuthResult,error){
	email:=strings.ToLower(strings.TrimSpace(input.Email))
	phone:=strings.TrimSpace(input.Phone)
	if email==""&&phone==""{return AuthResult{},errors.New("email or phone is required")}
	if strings.TrimSpace(input.FirstName)==""||strings.TrimSpace(input.LastName)==""{return AuthResult{},errors.New("first name and last name are required")}
	if strings.TrimSpace(input.DeviceID)==""{return AuthResult{},errors.New("device id is required")}
	hash,err:=HashPassword(input.Password);if err!=nil{return AuthResult{},err}
	now:=time.Now().UTC()
	user,err:=s.repo.CreateUser(ctx,User{Email:email,Phone:phone,FirstName:strings.TrimSpace(input.FirstName),LastName:strings.TrimSpace(input.LastName),PasswordHash:hash,Status:"active",CreatedAt:now,UpdatedAt:now})
	if err!=nil{return AuthResult{},err}
	if err:=s.repo.AssignRole(ctx,user.ID,"sender");err!=nil{return AuthResult{},err}
	s.repo.RecordAuthEvent(ctx,AuthEvent{UserID:&user.ID,Type:"register",Success:true,IP:ip,CreatedAt:now})
	return s.issueSession(ctx,user,input.DeviceID,input.DeviceName)
}

func (s *Service) Login(ctx context.Context,input LoginInput,ip string)(AuthResult,error){
	user,err:=s.repo.FindUserByIdentifier(ctx,input.Identifier)
	if err!=nil||!VerifyPassword(user.PasswordHash,input.Password)||user.Status!="active"{
		if user.ID!=bson.NilObjectID{s.repo.RecordAuthEvent(ctx,AuthEvent{UserID:&user.ID,Type:"login",Success:false,IP:ip,CreatedAt:time.Now().UTC()})}else{s.repo.RecordAuthEvent(ctx,AuthEvent{Type:"login",Success:false,IP:ip,CreatedAt:time.Now().UTC()})}
		return AuthResult{},ErrInvalidCredentials
	}
	if strings.TrimSpace(input.DeviceID)==""{return AuthResult{},errors.New("device id is required")}
	s.repo.RecordAuthEvent(ctx,AuthEvent{UserID:&user.ID,Type:"login",Success:true,IP:ip,CreatedAt:time.Now().UTC()})
	return s.issueSession(ctx,user,input.DeviceID,input.DeviceName)
}

func (s *Service) Refresh(ctx context.Context,input RefreshInput,ip string)(AuthResult,error){
	raw:=strings.TrimSpace(input.RefreshToken);if raw==""{return AuthResult{},ErrInvalidRefresh}
	session,err:=s.repo.FindSessionByTokenHash(ctx,HashOpaqueToken(raw));if err!=nil{return AuthResult{},ErrInvalidRefresh}
	if session.RevokedAt!=nil{
		_=s.repo.RevokeFamily(ctx,session.FamilyID,"refresh_token_reuse")
		s.repo.RecordAuthEvent(ctx,AuthEvent{UserID:&session.UserID,Type:"refresh_reuse",Success:false,IP:ip,CreatedAt:time.Now().UTC()})
		return AuthResult{},ErrSessionReuse
	}
	if time.Now().UTC().After(session.ExpiresAt){return AuthResult{},ErrInvalidRefresh}
	user,err:=s.repo.FindUserByID(ctx,session.UserID);if err!=nil||user.Status!="active"{return AuthResult{},ErrInvalidRefresh}
	nextRaw,err:=NewOpaqueToken(32);if err!=nil{return AuthResult{},err}
	next,rotated,err:=s.repo.RotateSession(ctx,session.ID,Session{UserID:session.UserID,FamilyID:session.FamilyID,TokenHash:HashOpaqueToken(nextRaw),DeviceID:choose(input.DeviceID,session.DeviceID),DeviceName:choose(input.DeviceName,session.DeviceName),ExpiresAt:time.Now().UTC().Add(s.refreshTokenTTL)})
	if err!=nil{return AuthResult{},err}
	if !rotated{_=s.repo.RevokeFamily(ctx,session.FamilyID,"concurrent_refresh_or_reuse");return AuthResult{},ErrSessionReuse}
	access,expiresAt,err:=s.tokens.IssueAccessToken(user.ID.Hex(),next.ID.Hex());if err!=nil{return AuthResult{},err}
	s.repo.RecordAuthEvent(ctx,AuthEvent{UserID:&user.ID,Type:"refresh",Success:true,IP:ip,CreatedAt:time.Now().UTC()})
	return AuthResult{User:publicUser(user),AccessToken:access,RefreshToken:nextRaw,TokenType:"Bearer",ExpiresAt:expiresAt},nil
}

func (s *Service) Logout(ctx context.Context,sessionID,ip string)error{
	id,err:=bson.ObjectIDFromHex(sessionID);if err!=nil{return ErrInvalidToken}
	if err:=s.repo.RevokeSession(ctx,id,"logout");err!=nil{return err}
	s.repo.RecordAuthEvent(ctx,AuthEvent{Type:"logout",Success:true,IP:ip,CreatedAt:time.Now().UTC()})
	return nil
}

func (s *Service) Me(ctx context.Context,userID string)(PublicUser,error){
	id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return PublicUser{},ErrNotFound}
	user,err:=s.repo.FindUserByID(ctx,id);if err!=nil{return PublicUser{},err}
	return publicUser(user),nil
}

func (s *Service) ValidateAccessSession(ctx context.Context,userID,sessionID string)bool{
	uid,err:=bson.ObjectIDFromHex(userID);if err!=nil{return false}
	sid,err:=bson.ObjectIDFromHex(sessionID);if err!=nil{return false}
	session,err:=s.repo.FindSessionByID(ctx,sid);if err!=nil||session.UserID!=uid||session.RevokedAt!=nil||time.Now().UTC().After(session.ExpiresAt){return false}
	user,err:=s.repo.FindUserByID(ctx,uid)
	return err==nil&&user.Status=="active"
}

func (s *Service) HasPermission(ctx context.Context,userID,permission string)(bool,error){
	id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return false,ErrNotFound}
	permissions,err:=s.repo.UserPermissions(ctx,id);if err!=nil{return false,err}
	for _,item:=range permissions{if item==permission{return true,nil}}
	return false,nil
}

func (s *Service) issueSession(ctx context.Context,user User,deviceID,deviceName string)(AuthResult,error){
	raw,err:=NewOpaqueToken(32);if err!=nil{return AuthResult{},err}
	familyID,err:=NewOpaqueToken(16);if err!=nil{return AuthResult{},err}
	now:=time.Now().UTC()
	session,err:=s.repo.CreateSession(ctx,Session{UserID:user.ID,FamilyID:familyID,TokenHash:HashOpaqueToken(raw),DeviceID:deviceID,DeviceName:deviceName,CreatedAt:now,ExpiresAt:now.Add(s.refreshTokenTTL)})
	if err!=nil{return AuthResult{},err}
	access,expiresAt,err:=s.tokens.IssueAccessToken(user.ID.Hex(),session.ID.Hex());if err!=nil{return AuthResult{},err}
	return AuthResult{User:publicUser(user),AccessToken:access,RefreshToken:raw,TokenType:"Bearer",ExpiresAt:expiresAt},nil
}

func publicUser(user User)PublicUser{return PublicUser{ID:user.ID.Hex(),Email:user.Email,Phone:user.Phone,FirstName:user.FirstName,LastName:user.LastName,Status:user.Status}}
func choose(value,fallback string)string{if strings.TrimSpace(value)!=""{return strings.TrimSpace(value)};return fallback}
func IsDuplicate(err error)bool{return mongo.IsDuplicateKeyError(err)}
