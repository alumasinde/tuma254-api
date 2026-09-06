package routing

import ("context";"net/http";"net/http/httptest";"testing";"time")

func TestValhallaRoute(t *testing.T){
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){if r.URL.Path!="/route"{t.Fatalf("unexpected path %s",r.URL.Path)};w.Header().Set("Content-Type","application/json");_,_=w.Write([]byte("{\"trip\":{\"summary\":{\"length\":18.5,\"time\":2400},\"legs\":[{\"shape\":\"encoded\"}]}}"))}));defer server.Close()
 client:=NewValhalla(server.URL,time.Second);route,err:=client.Route(context.Background(),RouteRequest{Origin:Point{Latitude:-1.286,Longitude:36.817},Destination:Point{Latitude:-1.2,Longitude:36.9}})
 if err!=nil{t.Fatal(err)};if route.DistanceMeters!=18500||route.DurationSeconds!=2400||route.Geometry!="encoded"{t.Fatalf("unexpected route %#v",route)}
}
func TestValhallaMatrix(t *testing.T){
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){if r.URL.Path!="/sources_to_targets"{t.Fatalf("unexpected path %s",r.URL.Path)};w.Header().Set("Content-Type","application/json");_,_=w.Write([]byte("{\"sources_to_targets\":[[{\"distance\":1.2,\"time\":300}]]}"))}));defer server.Close()
 client:=NewValhalla(server.URL,time.Second);cells,err:=client.Matrix(context.Background(),MatrixRequest{Sources:[]Point{{Latitude:0,Longitude:0}},Targets:[]Point{{Latitude:1,Longitude:1}}})
 if err!=nil{t.Fatal(err)};if len(cells)!=1||cells[0].DistanceMeters!=1200||cells[0].DurationSeconds!=300{t.Fatalf("unexpected cells %#v",cells)}
}
