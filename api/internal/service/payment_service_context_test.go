package service

import (
	"context"
	"testing"
	"time"
)

func TestDetachOutboundRequestContextIgnoresParentCancel(t *testing.T) {
	type contextKey string

	parent, parentCancel := context.WithCancel(context.WithValue(context.Background(), contextKey("trace_id"), "trace-001"))
	parentCancel()

	ctx, cancel := detachOutboundRequestContext(parent)
	defer cancel()

	if err := ctx.Err(); err != nil {
		t.Fatalf("expected detached context to ignore parent cancellation, got %v", err)
	}
	if got := ctx.Value(contextKey("trace_id")); got != "trace-001" {
		t.Fatalf("expected detached context to preserve values, got %v", got)
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatalf("expected detached context to have a timeout deadline")
	}
	if !deadline.After(time.Now()) {
		t.Fatalf("expected detached context deadline to be in the future, got %v", deadline)
	}
}
