package pluginsystem

import (
	"context"
	"time"
)

type workerSlotAcquireTimeoutKey struct{}

// WithWorkerSlotAcquireTimeout scopes background admission to the caller's
// existing finite job budget. Interactive calls retain the configured default.
func WithWorkerSlotAcquireTimeout(parent context.Context, timeout time.Duration) context.Context {
	if timeout <= 0 {
		return parent
	}
	return context.WithValue(parent, workerSlotAcquireTimeoutKey{}, timeout)
}
