package riders

import "testing"

func TestAvailabilityTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{AvailabilityOffline, AvailabilityAvailable, true},
		{AvailabilityAvailable, AvailabilityBusy, true},
		{AvailabilityBusy, AvailabilityAvailable, true},
		{AvailabilityAvailable, AvailabilityOffline, true},
		{AvailabilityOffline, AvailabilityBusy, false},
		{AvailabilityBusy, "invalid", false},
	}
	for _, tc := range cases {
		if got := validAvailabilityTransition(tc.from, tc.to); got != tc.want {
			t.Fatalf("%s -> %s: got %v want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestVehicleValidation(t *testing.T) {
	if !validVehicleType("motorcycle") {
		t.Fatal("expected motorcycle to be valid")
	}
	if validVehicleType("plane") {
		t.Fatal("expected plane to be invalid")
	}
	if got := normalizeRegistration(" kda 123a "); got != "KDA123A" {
		t.Fatalf("got %q", got)
	}
}
