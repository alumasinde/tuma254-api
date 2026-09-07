package deliveries
import("testing";"time";"github.com/alumasinde/tuma254-api/internal/locations")
func TestOTPLifecycle(t *testing.T){o,c,e:=newOTP(time.Minute);if e!=nil{t.Fatal(e)};if o.Version!=1||o.MaxAttempts!=5||o.LastSentAt.IsZero(){t.Fatalf("bad lifecycle defaults: %+v",o)};if verifyOTP(&o,c)!=nil{t.Fatal("valid otp rejected")};now:=time.Now();o.LockedAt=&now;if verifyOTP(&o,c)!=ErrOTPLocked{t.Fatal("locked otp accepted")}}
func TestOTPExpiry(t *testing.T){o,_,_:=newOTP(-time.Second);if verifyOTP(&o,"000000")!=ErrExpired{t.Fatal("expired otp accepted")}}

func TestCustodyValidation(t *testing.T){if validCustody(CustodyInput{}){t.Fatal("empty custody accepted")};if !validCustody(CustodyInput{Kind:"photo"}){t.Fatal("evidence kind rejected")}}

func TestCustodyRequiresPointWhenLocationReaderEnabled(t *testing.T){in:=CustodyInput{Kind:"photo",Location:locations.Point{Type:"Point",Coordinates:[2]float64{36.8,-1.2}}};if !validCustody(in){t.Fatal("valid point rejected")}}
