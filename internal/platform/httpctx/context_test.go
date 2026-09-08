package httpctx

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestWithUserIDRoundTrip(t *testing.T) {
	userID := uuid.New()
	ctx := WithUserID(context.Background(), userID)
	got, ok := UserID(ctx)
	if !ok || got != userID {
		t.Fatalf("expected stored user id")
	}
}
