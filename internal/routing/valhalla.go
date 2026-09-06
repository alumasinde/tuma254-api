package routing

import (
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "net/http"
 "strings"
 "time"
)

type Valhalla struct{baseURL string;client *http.Client}
func NewValhalla(baseURL string,timeout time.Duration)*Valhalla{if timeout<=0{timeout=5*time.Second};return &Valhalla{baseURL:strings.TrimRight(strings.TrimSpace(baseURL),"/"),client:&http.Client{Timeout:timeout}}}
func(v *Valhalla)Route(ctx context.Context,r RouteRequest)(Route,error){if err:=validRoute(r);err!=nil{return Route{},err};payload:=map[string]any{"locations":[]map[string]float64{{"lat":r.Origin.Latitude,"lon":r.Origin.Longitude},{"lat":r.Destination.Latitude,"lon":r.Destination.Longitude}},"costing":costing(r.Costing),"units":"kilometers"};var raw struct{Trip struct{Summary struct{Length float64 `json:"length"`; Time float64 `json:"time"`}`json:"summary"`; Legs []struct{Shape string `json:"shape"`}`json:"legs"`}`json:"trip"`};if err:=v.call(ctx,"/route",payload,&raw);err!=nil{return Route{},err};geometry:="";if len(raw.Trip.Legs)>0{geometry=raw.Trip.Legs[0].Shape};return Route{DistanceMeters:raw.Trip.Summary.Length*1000,DurationSeconds:raw.Trip.Summary.Time,Geometry:geometry},nil}
func(v *Valhalla)Matrix(ctx context.Context,r MatrixRequest)([]MatrixCell,error){if len(r.Sources)==0||len(r.Targets)==0{return nil,errors.New("sources and targets required")};payload:=map[string]any{"sources":points(r.Sources),"targets":points(r.Targets),"costing":costing(r.Costing),"units":"kilometers"};var raw struct{SourcesToTargets [][]struct{Distance float64 `json:"distance"`; Time float64 `json:"time"`}`json:"sources_to_targets"`}`json:"sources_to_targets"`};if err:=v.call(ctx,"/sources_to_targets",payload,&raw);err!=nil{return nil,err};out:=make([]MatrixCell,0,len(r.Sources)*len(r.Targets));for i,row:=range raw.SourcesToTargets{for j,c:=range row{out=append(out,MatrixCell{FromIndex:i,ToIndex:j,DistanceMeters:c.Distance*1000,DurationSeconds:c.Time})}};return out,nil}
func(v *Valhalla)call(ctx,path string,payload any,dst any)error{if v.baseURL==""{return errors.New("valhalla base URL is required")};body,err:=json.Marshal(payload);if err!=nil{return err};req,err:=http.NewRequestWithContext(ctx,http.MethodPost,v.baseURL+path,bytes.NewReader(body));if err!=nil{return err};req.Header.Set("Content-Type","application/json");res,err:=v.client.Do(req);if err!=nil{return err};defer res.Body.Close();if res.StatusCode<200||res.StatusCode>=300{return fmt.Errorf("valhalla status %d",res.StatusCode)};return json.NewDecoder(res.Body).Decode(dst)}
func points(values []Point)[]map[string]float64{out:=make([]map[string]float64,0,len(values));for _,p:=range values{out=append(out,map[string]float64{"lat":p.Latitude,"lon":p.Longitude})};return out}
func validRoute(r RouteRequest)error{for _,p:=range []Point{r.Origin,r.Destination}{if p.Latitude < -90||p.Latitude>90||p.Longitude < -180||p.Longitude>180{return errors.New("invalid route coordinates")}};return nil}
func costing(v string)string{v=strings.TrimSpace(v);if v==""{return "auto"};return v}
