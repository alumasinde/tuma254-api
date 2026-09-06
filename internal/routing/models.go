package routing

type Point struct{ Latitude float64 `json:"latitude"`; Longitude float64 `json:"longitude"` }
type RouteRequest struct{ Origin Point `json:"origin"`; Destination Point `json:"destination"`; Costing string `json:"costing"` }
type Route struct{ DistanceMeters float64 `json:"distance_meters"`; DurationSeconds float64 `json:"duration_seconds"`; Geometry string `json:"geometry,omitempty"` }
type MatrixRequest struct{ Sources []Point `json:"sources"`; Targets []Point `json:"targets"`; Costing string `json:"costing"` }
type MatrixCell struct{ FromIndex int `json:"from_index"`; ToIndex int `json:"to_index"`; DistanceMeters float64 `json:"distance_meters"`; DurationSeconds float64 `json:"duration_seconds"` }
type Router interface{Route(ctx context.Context,request RouteRequest)(Route,error);Matrix(ctx context.Context,request MatrixRequest)([]MatrixCell,error)}
