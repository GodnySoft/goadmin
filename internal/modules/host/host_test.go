package host

import (
	"context"
	"testing"
)

func TestUnknownCommand(t *testing.T) {
	m := &Module{}
	ctx := context.Background()
	_, err := m.Execute(ctx, "unknown", nil)
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
}
