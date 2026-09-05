package migrate

import "testing"

func TestRunnerConstruction(t *testing.T) {
	if New(nil, "./migrations") == nil { t.Fatal("expected runner") }
}
