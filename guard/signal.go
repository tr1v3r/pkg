package guard

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

var (
	ctx  = context.Background()
	stop = func() {}
)

// InspectShutSignal inspect shutdown signal
// default signals: syscall.SIGINT, syscall.SIGTERM
func InspectShutSignal(signals ...os.Signal) {
	if len(signals) != 0 {
		ctx, stop = signal.NotifyContext(ctx, signals...)
	} else {
		ctx, stop = signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
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
