package guard

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"
)

// armCustomWatcher installs a SIGUSR1-only watcher and restores the package
// defaults afterwards, so the shared package-level context never leaks a
// cancelled state into whichever test runs next.
func armCustomWatcher(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { InspectShutSignal(syscall.SIGINT, syscall.SIGTERM) })
	InspectShutSignal(syscall.SIGUSR1)
}

// TestInspectShutSignal_CustomSignalsNotBornCancelled is the H3 regression: after
// installing a custom signal set the guard must stay live until that signal
// actually arrives. Deriving the replacement context from the one stop() had just
// cancelled made it born cancelled, so Cancelled() reported a shutdown that never
// happened.
func TestInspectShutSignal_CustomSignalsNotBornCancelled(t *testing.T) {
	armCustomWatcher(t)

	if Cancelled() {
		t.Fatal("Cancelled() = true right after InspectShutSignal, before any signal")
	}
	select {
	case <-Cancel():
		t.Fatal("Cancel() already closed right after InspectShutSignal, before any signal")
	default:
	}
	if err := Context().Err(); err != nil {
		t.Fatalf("Context().Err() = %v, want nil before any signal", err)
	}
}

// TestInspectShutSignal_RepeatedReregistrationStaysLive pins the same invariant
// across repeated calls, the shape real callers use when they install a custom
// signal set at startup.
func TestInspectShutSignal_RepeatedReregistrationStaysLive(t *testing.T) {
	armCustomWatcher(t)

	for i := 1; i <= 3; i++ {
		InspectShutSignal(syscall.SIGUSR1)
		if Cancelled() {
			t.Fatalf("after re-registration #%d: Cancelled() = true, want false", i)
		}
	}
}

// TestInspectShutSignal_CustomSignalCancelsContext covers the shutdown side: once
// the configured signal is delivered the context is cancelled with
// context.Canceled, which is the observable shutdown hook callers select on.
func TestInspectShutSignal_CustomSignalCancelsContext(t *testing.T) {
	armCustomWatcher(t)

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGUSR1); err != nil {
		t.Fatalf("send SIGUSR1: %v", err)
	}

	select {
	case <-Cancel():
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown channel not closed within 3s after SIGUSR1")
	}
	if !Cancelled() {
		t.Error("Cancelled() = false after SIGUSR1")
	}
	if err := Context().Err(); !errors.Is(err, context.Canceled) {
		t.Errorf("Context().Err() = %v, want context.Canceled", err)
	}
}

// TestInspectShutSignal_NoArgsKeepsWatcher pins that the default (no-argument)
// call is a no-op that neither replaces nor disturbs the current watcher.
func TestInspectShutSignal_NoArgsKeepsWatcher(t *testing.T) {
	armCustomWatcher(t)

	before := Context()
	InspectShutSignal()
	if Context() != before {
		t.Error("InspectShutSignal() with no signals replaced the watcher context")
	}
	if Cancelled() {
		t.Error("Cancelled() = true after a no-op InspectShutSignal()")
	}
}
