package shutdown

import (
	"context"
	"os/signal"
	"syscall"
)

var ctx, stop = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

// Cancel return shutdown signal chan
func Cancel() <-chan struct{} { return ctx.Done() }

// Cancelled judge if project shutdown
func Cancelled() bool {
	select {
	case <-Cancel():
		return true
	default:
		return false
	}
}

// Stop call ctx's stop
func Stop() { stop() }
