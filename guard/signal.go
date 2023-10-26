package guard

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

var ctx, stop = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

// InspectShutSignal inspect shutdown signal
// default signals: syscall.SIGINT, syscall.SIGTERM
func InspectShutSignal(signals ...os.Signal) {
	if len(signals) != 0 {
		stop()
		ctx, stop = signal.NotifyContext(ctx, signals...)
	}
}

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

// Context return context
func Context() context.Context { return ctx }
