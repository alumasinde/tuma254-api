package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alumasinde/tuma254-api/internal/database/migrations"
	"github.com/alumasinde/tuma254-api/internal/identity"
	"github.com/alumasinde/tuma254-api/testkit"
)

func TestIdentityRepositoryRBAC(t *testing.T) {
	uri:=os.Getenv("MONGODB_TEST_URI")
	if uri==""{t.Skip("MONGODB_TEST_URI not configured")}
	db:=testkit.MongoDatabase(t,uri,"tuma254_identity_repository_test")
	if err:=migrations.Run(context.Background(),db,migrations.All()...);err!=nil{t.Fatalf("migrate: %v",err)}

	repo:=identity.NewRepository(db)
	now:=time.Now().UTC()
	user,err:=repo.CreateUser(context.Background(),identity.User{Email:"rider@example.com",FirstName:"Test",LastName:"Rider",PasswordHash:"hash",Status:"active",CreatedAt:now,UpdatedAt:now})
	if err!=nil{t.Fatalf("create user: %v",err)}
	if err:=repo.AssignRole(context.Background(),user.ID,"rider");err!=nil{t.Fatalf("assign role: %v",err)}
	permissions,err:=repo.UserPermissions(context.Background(),user.ID)
	if err!=nil{t.Fatalf("permissions: %v",err)}
	found:=false
	for _,permission:=range permissions{if permission=="rider.availability.update"{found=true}}
	if !found{t.Fatal("expected rider permission")}
}
