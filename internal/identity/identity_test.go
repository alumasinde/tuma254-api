package identity
import "testing"
func TestNormalizeEmail(t *testing.T){if got:=normEmail(" Test@Example.COM ");got!="test@example.com"{t.Fatal(got)}}
func TestNormalizePhone(t *testing.T){if got:=normPhone("0712 345678");got!="+254712345678"{t.Fatal(got)}}
