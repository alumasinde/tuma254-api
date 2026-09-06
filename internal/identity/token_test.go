package identity

import (
	"testing"
	"time"
)

func TestAccessTokenRoundTrip(t *testing.T){
	manager:=NewTokenManager("01234567890123456789012345678901",time.Minute)
	raw,_,err:=manager.IssueAccessToken("user-1","session-1")
	if err!=nil{t.Fatalf("issue: %v",err)}
	claims,err:=manager.ParseAccessToken(raw)
	if err!=nil{t.Fatalf("parse: %v",err)}
	if claims.Subject!="user-1"||claims.SessionID!="session-1"{t.Fatal("unexpected claims")}
}

func TestOpaqueTokenHashIsStable(t *testing.T){
	if HashOpaqueToken("token")!=HashOpaqueToken("token"){t.Fatal("hash should be stable")}
	if HashOpaqueToken("token")==HashOpaqueToken("other"){t.Fatal("different tokens should not share a hash")}
}
