package guard

import (
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCatchStack(t *testing.T) {
	stack := CatchStack()
	if stack == "" {
		t.Fatal("CatchStack returned empty string")
	}
	if !strings.Contains(stack, "goroutine") {
		t.Errorf("stack = %q, want it to mention goroutine", stack)
	}
}

// TestGuardLifecycle walks the whole shutdown flow in order, because the
// package keeps a single shared context whose cancelled state cannot be
// reset (re-registering derives from the previous context).
func TestGuardLifecycle(t *testing.T) {
	// 1. initially live
	if Cancelled() {
		t.Fatal("context cancelled before any signal")
	}
	select {
	case <-Cancel():
		t.Fatal("context should not be done initially")
	default:
	}

	// 2. no-arg re-register keeps the current signals registered.
	InspectShutSignal()

	// 3. re-register with SIGUSR1 only, then deliver SIGUSR1 to ourselves.
	InspectShutSignal(syscall.SIGUSR1)

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGUSR1); err != nil {
		t.Fatalf("send SIGUSR1 fail: %s", err)
	}

	select {
	case <-Cancel():
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown channel not closed after SIGUSR1")
	}

	if !Cancelled() {
		t.Error("Cancelled() should be true after signal")
	}

	// 4. Stop() cancels the (already cancelled) context; calling it must be
	// safe and keep the context done.
	Stop()

	select {
	case <-Cancel():
	default:
		t.Fatal("Stop() should leave the context done")
	}
}
