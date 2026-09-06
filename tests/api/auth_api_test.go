package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alumasinde/tuma254-api/internal/database/migrations"
	"github.com/alumasinde/tuma254-api/internal/identity"
	httpserver "github.com/alumasinde/tuma254-api/internal/platform/http"
	"github.com/alumasinde/tuma254-api/testkit"
)

func TestAuthenticatedMeAndLogout(t *testing.T) {
	uri:=os.Getenv("MONGODB_TEST_URI")
	if uri==""{t.Skip("MONGODB_TEST_URI not configured")}
	db:=testkit.MongoDatabase(t,uri,"tuma254_auth_api_test")
	if err:=migrations.Run(context.Background(),db,migrations.All()...);err!=nil{t.Fatalf("migrate: %v",err)}

	repo:=identity.NewRepository(db)
	now:=time.Now().UTC()
	user,err:=repo.CreateUser(context.Background(),identity.User{Email:"api@example.com",FirstName:"API",LastName:"User",PasswordHash:"hash",Status:"active",CreatedAt:now,UpdatedAt:now})
	if err!=nil{t.Fatalf("create user: %v",err)}
	session,err:=repo.CreateSession(context.Background(),identity.Session{UserID:user.ID,FamilyID:"family",TokenHash:identity.HashOpaqueToken("refresh"),DeviceID:"device-1",CreatedAt:now,ExpiresAt:now.Add(time.Hour)})
	if err!=nil{t.Fatalf("create session: %v",err)}

	tokens:=identity.NewTokenManager("01234567890123456789012345678901",time.Minute)
	access,_,err:=tokens.IssueAccessToken(user.ID.Hex(),session.ID.Hex())
	if err!=nil{t.Fatalf("issue access: %v",err)}
	auth:=identity.NewHandler(identity.NewService(repo,tokens,time.Hour))
	handler:=httpserver.NewHandler(func()error{return nil},auth)

	me:=httptest.NewRequest(http.MethodGet,"/api/v1/auth/me",nil)
	me.Header.Set("Authorization","Bearer "+access)
	meResponse:=httptest.NewRecorder()
	handler.ServeHTTP(meResponse,me)
	if meResponse.Code!=http.StatusOK{t.Fatalf("expected me 200, got %d: %s",meResponse.Code,meResponse.Body.String())}

	logout:=httptest.NewRequest(http.MethodPost,"/api/v1/auth/logout",nil)
	logout.Header.Set("Authorization","Bearer "+access)
	logoutResponse:=httptest.NewRecorder()
	handler.ServeHTTP(logoutResponse,logout)
	if logoutResponse.Code!=http.StatusNoContent{t.Fatalf("expected logout 204, got %d",logoutResponse.Code)}

	again:=httptest.NewRequest(http.MethodGet,"/api/v1/auth/me",nil)
	again.Header.Set("Authorization","Bearer "+access)
	againResponse:=httptest.NewRecorder()
	handler.ServeHTTP(againResponse,again)
	if againResponse.Code!=http.StatusUnauthorized{t.Fatalf("expected revoked session 401, got %d",againResponse.Code)}
}
