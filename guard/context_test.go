package guard

import "testing"

func TestContext(t *testing.T) {
	if Context() == nil {
		t.Fatal("Context() returned nil")
	}
}
