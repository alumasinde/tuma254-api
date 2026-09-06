package ratelimit

import("testing";"time")
func TestLimiter(t *testing.T){
	l:=New()
	if !l.Allow("login:127.0.0.1",2,time.Minute){t.Fatal("first request should pass")}
	if !l.Allow("login:127.0.0.1",2,time.Minute){t.Fatal("second request should pass")}
	if l.Allow("login:127.0.0.1",2,time.Minute){t.Fatal("third request should be blocked")}
}
