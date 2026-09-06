package locations

import "testing"

func TestLocationValidation(t *testing.T){
 valid:=UpdateRiderLocationInput{Latitude:-1.286389,Longitude:36.817223,AccuracyMeters:10,HeadingDegrees:0,SpeedMPS:8}
 if err:=validate(valid,100,100);err!=nil{t.Fatalf("expected valid location: %v",err)}
 if err:=validate(UpdateRiderLocationInput{Latitude:100,Longitude:36},100,100);err==nil{t.Fatal("expected invalid latitude")}
 if err:=validate(UpdateRiderLocationInput{Latitude:0,Longitude:0,AccuracyMeters:101},100,100);err==nil{t.Fatal("expected invalid accuracy")}
 if err:=validate(UpdateRiderLocationInput{Latitude:0,Longitude:0,HeadingDegrees:360},100,100);err==nil{t.Fatal("expected invalid heading")}
 if err:=validate(UpdateRiderLocationInput{Latitude:0,Longitude:0,SpeedMPS:101},100,100);err==nil{t.Fatal("expected invalid speed")}
}
func TestHaversine(t *testing.T){d:=haversine(-1.286389,36.817223,-1.286389,36.817223);if d>0.001{t.Fatalf("expected zero distance, got %f",d)}}
