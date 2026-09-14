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
//
// Calling it with a non-empty signal set replaces the watcher installed by the
// previous call: the new context derives from a fresh, never-cancelled parent and
// the previous stop function is invoked so its signal handler is unregistered.
// Deriving the new context from the previous one was wrong because stop() had
// already cancelled it, which made the replacement born cancelled: Cancelled()
// reported a shutdown that never happened, so any caller selecting a custom signal
// set at startup saw an immediately "shut down" process.
func InspectShutSignal(signals ...os.Signal) {
	if len(signals) == 0 {
		return
	}
	// Install the replacement before tearing down the previous watcher so there is
	// no window in which no handler is registered.
	newCtx, newStop := signal.NotifyContext(context.Background(), signals...)
	stop()
	ctx, stop = newCtx, newStop
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
