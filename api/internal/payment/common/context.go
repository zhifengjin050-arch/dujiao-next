package common

import (
	"context"
	"time"
)

// DefaultTimeout ?????????
const DefaultTimeout = 12 * time.Second

// WithDefaultTimeout ??? deadline ????????
func WithDefaultTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, DefaultTimeout)
}
