package locations

import (
 "context"
 "errors"
 "math"
 "strings"
 "time"
 "go.mongodb.org/mongo-driver/v2/bson"
)

var (
 ErrInvalidCoordinates=errors.New("invalid coordinates")
 ErrInvalidAccuracy=errors.New("invalid accuracy")
 ErrInvalidHeading=errors.New("invalid heading")
 ErrInvalidSpeed=errors.New("invalid speed")
 ErrNotEligible=errors.New("rider not eligible for live location")
 ErrLimit=errors.New("invalid limit")
)

type RiderEligibility interface{ CanPublishLocation(context.Context,string)(bool,error) }
type Service struct{repo *Repository; eligibility RiderEligibility; freshness time.Duration; maxAccuracy float64; maxSpeed float64}
func NewService(repo *Repository,eligibility RiderEligibility,freshness time.Duration,maxAccuracy,maxSpeed float64)*Service{return &Service{repo:repo,eligibility:eligibility,freshness:freshness,maxAccuracy:maxAccuracy,maxSpeed:maxSpeed}}

func (s *Service) UpdateRiderLocation(ctx context.Context,riderID string,in UpdateRiderLocationInput)(PublicRiderLocation,error){
 id,err:=bson.ObjectIDFromHex(riderID);if err!=nil{return PublicRiderLocation{},ErrNotFound}
 if err:=validate(in,s.maxAccuracy,s.maxSpeed);err!=nil{return PublicRiderLocation{},err}
 ok,err:=s.eligibility.CanPublishLocation(ctx,riderID);if err!=nil{return PublicRiderLocation{},err};if !ok{return PublicRiderLocation{},ErrNotEligible}
 now:=time.Now().UTC();v,err:=s.repo.UpsertRiderLocation(ctx,RiderLocation{RiderID:id,Location:point(in.Longitude,in.Latitude),AccuracyMeters:in.AccuracyMeters,HeadingDegrees:in.HeadingDegrees,SpeedMPS:in.SpeedMPS,UpdatedAt:now});if err!=nil{return PublicRiderLocation{},err};return publicRider(v,0),nil
}
func (s *Service) Nearby(ctx context.Context,lat,lng,radius float64,limit int)([]PublicRiderLocation,error){
 if lat < -90||lat>90||lng < -180||lng>180||radius<=0{return nil,ErrInvalidCoordinates}
 if limit<=0{return nil,ErrLimit};if limit>100{limit=100}
 values,err:=s.repo.Nearby(ctx,point(lng,lat),radius,limit,time.Now().UTC().Add(-s.freshness));if err!=nil{return nil,err}
 out:=make([]PublicRiderLocation,0,len(values));for _,v:=range values{out=append(out,publicRider(v,haversine(lat,lng,v.Location.Coordinates[1],v.Location.Coordinates[0])))};return out,nil
}
func (s *Service) CreatePlace(ctx context.Context,userID string,in SavePlaceInput)(PublicSavedPlace,error){id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return PublicSavedPlace{},ErrNotFound};if in.Latitude < -90||in.Latitude>90||in.Longitude < -180||in.Longitude>180{return PublicSavedPlace{},ErrInvalidCoordinates};label:=strings.TrimSpace(in.Label);if label==""||len(label)>80{return PublicSavedPlace{},errors.New("invalid label")};now:=time.Now().UTC();p,err:=s.repo.CreatePlace(ctx,SavedPlace{UserID:id,Label:label,ContactName:strings.TrimSpace(in.ContactName),PhoneNumber:strings.TrimSpace(in.PhoneNumber),Address:strings.TrimSpace(in.Address),Location:point(in.Longitude,in.Latitude),CreatedAt:now,UpdatedAt:now});if err!=nil{return PublicSavedPlace{},err};return publicPlace(p),nil}
func (s *Service) ListPlaces(ctx context.Context,userID string)([]PublicSavedPlace,error){id,err:=bson.ObjectIDFromHex(userID);if err!=nil{return nil,ErrNotFound};values,err:=s.repo.ListPlaces(ctx,id);if err!=nil{return nil,err};out:=make([]PublicSavedPlace,0,len(values));for _,v:=range values{out=append(out,publicPlace(v))};return out,nil}
func (s *Service) DeletePlace(ctx context.Context,userID,placeID string)error{u,err:=bson.ObjectIDFromHex(userID);if err!=nil{return ErrNotFound};p,err:=bson.ObjectIDFromHex(placeID);if err!=nil{return ErrNotFound};return s.repo.DeletePlace(ctx,u,p)}

func validate(in UpdateRiderLocationInput,maxAccuracy,maxSpeed float64)error{if in.Latitude < -90||in.Latitude>90||in.Longitude < -180||in.Longitude>180{return ErrInvalidCoordinates};if in.AccuracyMeters<0||(maxAccuracy>0&&in.AccuracyMeters>maxAccuracy){return ErrInvalidAccuracy};if in.HeadingDegrees<0||in.HeadingDegrees>=360{return ErrInvalidHeading};if in.SpeedMPS<0||(maxSpeed>0&&in.SpeedMPS>maxSpeed){return ErrInvalidSpeed};return nil}
func point(lng,lat float64)Point{return Point{Type:"Point",Coordinates:[2]float64{lng,lat}}}
func publicRider(v RiderLocation,d float64)PublicRiderLocation{return PublicRiderLocation{RiderID:v.RiderID.Hex(),Location:v.Location,AccuracyMeters:v.AccuracyMeters,HeadingDegrees:v.HeadingDegrees,SpeedMPS:v.SpeedMPS,UpdatedAt:v.UpdatedAt,DistanceMeters:d}}
func publicPlace(v SavedPlace)PublicSavedPlace{return PublicSavedPlace{ID:v.ID.Hex(),Label:v.Label,ContactName:v.ContactName,PhoneNumber:v.PhoneNumber,Address:v.Address,Location:v.Location,CreatedAt:v.CreatedAt,UpdatedAt:v.UpdatedAt}}
func haversine(lat1,lng1,lat2,lng2 float64)float64{const R=6371000;dlat:=(lat2-lat1)*math.Pi/180;dlng:=(lng2-lng1)*math.Pi/180;a:=math.Sin(dlat/2)*math.Sin(dlat/2)+math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(dlng/2)*math.Sin(dlng/2);return 2*R*math.Atan2(math.Sqrt(a),math.Sqrt(1-a))}
