package identity
import ("context";"errors";"net/http";"strings";"github.com/golang-jwt/jwt/v5")
type principalKey struct{}
type Principal struct{UserID string;Roles []string}
func (s *Service) RequireAuth(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){v:=strings.TrimPrefix(r.Header.Get("Authorization"),"Bearer ");if v==""{errJSON(w,401,"missing bearer token");return};t,e:=jwt.Parse(v,func(t *jwt.Token)(interface{},error){return []byte(s.accessSecret),nil});if e!=nil||!t.Valid{errJSON(w,401,"invalid access token");return};c:=t.Claims.(jwt.MapClaims);if c["typ"]!="access"{errJSON(w,401,"invalid access token");return};sub,_:=c["sub"].(string);if sub==""{errJSON(w,401,"invalid access token");return};p:=Principal{UserID:sub};if rs,ok:=c["roles"].([]interface{});ok{for _,x:=range rs{if z,ok:=x.(string);ok{p.Roles=append(p.Roles,z)}}};next.ServeHTTP(w,r.WithContext(context.WithValue(r.Context(),principalKey{},p)))})}
func PrincipalFromContext(ctx context.Context)(Principal,error){p,ok:=ctx.Value(principalKey{}).(Principal);if !ok{return Principal{},errors.New("principal missing")};return p,nil}
