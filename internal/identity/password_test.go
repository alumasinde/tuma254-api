package identity

import "testing"

func TestPasswordHashAndVerify(t *testing.T) {
	hash,err:=HashPassword("StrongPassword123")
	if err!=nil{t.Fatalf("hash password: %v",err)}
	if !VerifyPassword(hash,"StrongPassword123"){t.Fatal("expected password to verify")}
	if VerifyPassword(hash,"wrong-password"){t.Fatal("wrong password must not verify")}
}

func TestWeakPasswordRejected(t *testing.T) {
	if _,err:=HashPassword("short1");err==nil{t.Fatal("expected weak password error")}
}
