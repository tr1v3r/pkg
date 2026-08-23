package circuitbreaker

import (
	"errors"
	"testing"
)

func TestErrorAndUnwrap(t *testing.T) {
	inner := errors.New("timeout")
	e := &Error{Err: inner}

	if got := e.Error(); got != "circuit breaker open: timeout" {
		t.Errorf("Error() = %q", got)
	}
	if !errors.Is(e, inner) {
		t.Error("errors.Is should unwrap to inner")
	}
}
