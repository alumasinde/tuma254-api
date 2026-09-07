package services

import (
 "context"
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
 "golang.org/x/crypto/bcrypt"
)
var ErrInvalidCredentials=errors.New("invalid credentials")
var ErrInvalidToken=errors.New("invalid token")
type Service struct{repo repositories.Repository;secret []byte;accessTTL,refreshTTL time.Duration}
func New(r repositories.Repository,s string,a,b time.Duration)*Service{return &Service{repo:r,secret:[]byte(s),accessTTL:a,refreshTTL:b}}
func(s *Service)Register(ctx context.Context,in dtos.RegisterRequest,ua,ip string)(dtos.AuthResponse,error){in.Email=strings.ToLower(strings.TrimSpace(in.Email));in.Phone=strings.TrimSpace(in.Phone);in.FirstName=strings.TrimSpace(in.FirstName);in.LastName=strings.TrimSpace(in.LastName);if !strings.Contains(in.Email,"@")||len(in.Email)>254||len(in.Phone)<7||len(in.Phone)>32||len(in.FirstName)==0||len(in.FirstName)>100||len(in.LastName)==0||len(in.LastName)>100{return dtos.AuthResponse{},fmt.Errorf("invalid registration data")};if len(in.Password)<12||len(in.Password)>128{return dtos.AuthResponse{},fmt.Errorf("password must be between 12 and 128 characters")};h,err:=bcrypt.GenerateFromPassword([]byte(in.Password),bcrypt.DefaultCost);if err!=nil{return dtos.AuthResponse{},err};u,err:=s.repo.CreateUser(ctx,in.Email,in.Phone,in.FirstName,in.LastName,string(h));if err!=nil{return dtos.AuthResponse{},err};return s.issue(ctx,u,ua,ip)}
func(s *Service)Login(ctx context.Context,in dtos.LoginRequest,ua,ip string)(dtos.AuthResponse,error){u,h,err:=s.repo.FindByEmail(ctx,strings.ToLower(strings.TrimSpace(in.Email)));if err!=nil||!u.Active||bcrypt.CompareHashAndPassword([]byte(h),[]byte(in.Password))!=nil{return dtos.AuthResponse{},ErrInvalidCredentials};return s.issue(ctx,u,ua,ip)}
func(s *Service)Refresh(ctx context.Context,t,ua,ip string)(dtos.AuthResponse,error){h:=sha256.Sum256([]byte(t));u,err:=s.repo.ConsumeSession(ctx,h[:]);if err!=nil{return dtos.AuthResponse{},ErrInvalidToken};return s.issue(ctx,u,ua,ip)}
func(s *Service)Logout(ctx context.Context,t string)error{h:=sha256.Sum256([]byte(t));return s.repo.RevokeSession(ctx,h[:])}
func(s *Service)Me(ctx context.Context,id string)(models.User,error){u,err:=uuid.Parse(id);if err!=nil{return models.User{},ErrInvalidToken};return s.repo.FindByID(ctx,u)}
func(s *Service)issue(ctx context.Context,u models.User,ua,ip string)(dtos.AuthResponse,error){now:=time.Now();claims:=jwt.MapClaims{"sub":u.ID.String(),"roles":u.Roles,"iat":now.Unix(),"exp":now.Add(s.accessTTL).Unix(),"iss":"tuma254"};access,err:=jwt.NewWithClaims(jwt.SigningMethodHS256,claims).SignedString(s.secret);if err!=nil{return dtos.AuthResponse{},err};raw:=make([]byte,48);if _,err=rand.Read(raw);err!=nil{return dtos.AuthResponse{},err};refresh:=base64.RawURLEncoding.EncodeToString(raw);h:=sha256.Sum256([]byte(refresh));if err=s.repo.CreateSession(ctx,u.ID,h[:],now.Add(s.refreshTTL),ua,ip);err!=nil{return dtos.AuthResponse{},err};return dtos.AuthResponse{AccessToken:access,RefreshToken:refresh,TokenType:"Bearer",ExpiresIn:int64(s.accessTTL.Seconds())},nil}
func(s *Service)ParseAccess(raw string)(string,[]string,error){p,err:=jwt.Parse(raw,func(t *jwt.Token)(any,error){if t.Method.Alg()!=jwt.SigningMethodHS256.Alg(){return nil,ErrInvalidToken};return s.secret,nil},jwt.WithIssuer("tuma254"));if err!=nil||!p.Valid{return "",nil,ErrInvalidToken};c:=p.Claims.(jwt.MapClaims);id,_:=c["sub"].(string);if _,err:=uuid.Parse(id);err!=nil{return "",nil,ErrInvalidToken};roles:=[]string{};if xs,ok:=c["roles"].([]any);ok{for _,x:=range xs{if v,ok:=x.(string);ok{roles=append(roles,v)}}};return id,roles,nil}
