package deliveries
import("testing";"time")
func TestOTP(t *testing.T){o,c,e:=newOTP(time.Minute);if e!=nil||c==""{t.Fatal("otp generation failed")};if verifyOTP(&o,c)!=nil{t.Fatal("valid otp rejected")};if verifyOTP(&o,"000000")==nil{t.Fatal("wrong otp accepted")}}
func TestStops(t *testing.T){if validStop(Stop{}){t.Fatal("empty stop accepted")}}
