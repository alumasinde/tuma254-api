package deliveries
import("testing";"go.mongodb.org/mongo-driver/v2/bson")
func TestRankCandidate(t *testing.T){a:=rankCandidate(bson.NewObjectID(),1000,0);b:=rankCandidate(bson.NewObjectID(),500,1);if !(a.Score<b.Score){t.Fatalf("expected closer idle rider to rank ahead: %v >= %v",a.Score,b.Score)}}
func TestRankCandidateLoadPenalty(t *testing.T){a:=rankCandidate(bson.NewObjectID(),1000,0);b:=rankCandidate(bson.NewObjectID(),1000,2);if b.Score<=a.Score{t.Fatal("expected active load to increase score")}}
