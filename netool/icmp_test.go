package netool

import "testing"

func TestAccessICMPDelay(t *testing.T) {
	delay, err := ICMPDelay("127.0.0.1", 6)
	if err != nil {
		t.Errorf("Failed to test ICMP: %v", err)
	}
	t.Logf("target delay: %s", delay)
}
