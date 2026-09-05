package identity

import (
 "context"
 "crypto/sha256"
 "encoding/hex"
 "errors"
 "strings"
 "time"

 "github.com/golang-jwt/jwt/v5"
 "github.com/jackc/pgx/v5/pgxpool"
 "golang.org/x/crypto/bcrypt"
)

type Service struct { db *pgxpool.Pool; accessSecret, refreshSecret string; accessTTL, refreshTTL time.Duration }
func New(db *pgxpool.Pool, a,r string, at,rt time.Duration)*Service{return &Service{db:db,accessSecret:a,refreshSecret:r,accessTTL:at,refreshTTL:rt}}
type User struct { ID,FirstName,LastName string; Email *string; Phone *string; Roles []string }
type Tokens struct { AccessToken string `json:"access_token"`; RefreshToken string `json:"refresh_token"`; TokenType string `json:"token_type"`; ExpiresIn int64 `json:"expires_in"` }

func normEmail(v string) string{return strings.ToLower(strings.TrimSpace(v))}
func normPhone(v string) string { v=strings.TrimSpace(strings.ReplaceAll(v," ","")); if strings.HasPrefix(v,"07") {return "+254"+v[1:]}; if strings.HasPrefix(v,"254") {return "+"+v}; return v }
func (s *Service) Register(ctx context.Context, first,last,email,phone,password string)(User,Tokens,error){
 first,last=strings.TrimSpace(first),strings.TrimSpace(last); email=normEmail(email); phone=normPhone(phone)
 if first==""||last==""||len(password)<8||(email==""&&phone==""){return User{},Tokens{},errors.New("invalid registration data")}
 hash,err:=bcrypt.GenerateFromPassword([]byte(password),bcrypt.DefaultCost);if err!=nil{return User{},Tokens{},err}
 var u User; err=s.db.QueryRow(ctx,`INSERT INTO users(first_name,last_name,email,phone,password_hash) VALUES($1,$2,NULLIF($3,''),NULLIF($4,''),$5) RETURNING id,first_name,last_name,email,phone`,first,last,email,phone,string(hash)).Scan(&u.ID,&u.FirstName,&u.LastName,&u.Email,&u.Phone);if err!=nil{return User{},Tokens{},err}
 _,err=s.db.Exec(ctx,`INSERT INTO user_roles(user_id,role_id) SELECT $1,id FROM roles WHERE code='customer'`,u.ID);if err!=nil{return User{},Tokens{},err}; u.Roles=[]string{"customer"}
 t,err:=s.issue(ctx,u);return u,t,err
}
func (s *Service) Login(ctx context.Context, identifier,password string)(User,Tokens,error){
 id:=normEmail(identifier); phone:=normPhone(identifier); var u User; var hash string
 err:=s.db.QueryRow(ctx,`SELECT id,first_name,last_name,email,phone,password_hash FROM users WHERE lower(email)=lower($1) OR phone=$2 AND status='active'`,id,phone).Scan(&u.ID,&u.FirstName,&u.LastName,&u.Email,&u.Phone,&hash);if err!=nil{return User{},Tokens{},errors.New("invalid credentials")}
 if bcrypt.CompareHashAndPassword([]byte(hash),[]byte(password))!=nil{return User{},Tokens{},errors.New("invalid credentials")}
 rows,err:=s.db.Query(ctx,`SELECT r.code FROM roles r JOIN user_roles ur ON ur.role_id=r.id WHERE ur.user_id=$1 ORDER BY r.code`,u.ID);if err!=nil{return User{},Tokens{},err};defer rows.Close();for rows.Next(){var role string;rows.Scan(&role);u.Roles=append(u.Roles,role)}
 t,err:=s.issue(ctx,u);return u,t,err
}
func (s *Service) issue(ctx context.Context,u User)(Tokens,error){
 now:=time.Now(); access,err:=jwt.NewWithClaims(jwt.SigningMethodHS256,jwt.MapClaims{"sub":u.ID,"roles":u.Roles,"typ":"access","iat":now.Unix(),"exp":now.Add(s.accessTTL).Unix()}).SignedString([]byte(s.accessSecret));if err!=nil{return Tokens{},err}
 refresh,err:=jwt.NewWithClaims(jwt.SigningMethodHS256,jwt.MapClaims{"sub":u.ID,"typ":"refresh","iat":now.Unix(),"exp":now.Add(s.refreshTTL).Unix()}).SignedString([]byte(s.refreshSecret));if err!=nil{return Tokens{},err}
 sum:=sha256.Sum256([]byte(refresh));_,err=s.db.Exec(ctx,`INSERT INTO refresh_tokens(user_id,token_hash,expires_at) VALUES($1,$2,$3)`,u.ID,hex.EncodeToString(sum[:]),now.Add(s.refreshTTL));if err!=nil{return Tokens{},err}
 return Tokens{AccessToken:access,RefreshToken:refresh,TokenType:"Bearer",ExpiresIn:int64(s.accessTTL.Seconds())},nil
}
func (s *Service) Refresh(ctx context.Context,raw string)(Tokens,error){
 token,err:=jwt.Parse(raw,func(t *jwt.Token)(interface{},error){if _,ok:=t.Method.(*jwt.SigningMethodHMAC);!ok{return nil,errors.New("invalid signing method")};return []byte(s.refreshSecret),nil});if err!=nil||!token.Valid{return Tokens{},errors.New("invalid refresh token")}
 c,ok:=token.Claims.(jwt.MapClaims);if !ok||c["typ"]!="refresh"{return Tokens{},errors.New("invalid refresh token")}; uid,_:=c["sub"].(string)
 sum:=sha256.Sum256([]byte(raw)); var userID string; err=s.db.QueryRow(ctx,`SELECT user_id FROM refresh_tokens WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>now()`,hex.EncodeToString(sum[:])).Scan(&userID);if err!=nil||userID!=uid{return Tokens{},errors.New("invalid refresh token")}
 _,err=s.db.Exec(ctx,`UPDATE refresh_tokens SET revoked_at=now() WHERE token_hash=$1`,hex.EncodeToString(sum[:]));if err!=nil{return Tokens{},err}
 u:=User{ID:uid}; return s.issue(ctx,u)
}
func (s *Service) Logout(ctx context.Context,raw string) error {sum:=sha256.Sum256([]byte(raw));_,err:=s.db.Exec(ctx,`UPDATE refresh_tokens SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`,hex.EncodeToString(sum[:]));return err}
