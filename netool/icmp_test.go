package netool

import (
	"errors"
	"strings"
	"testing"
)

// TestAccessICMPDelay pings localhost; raw ICMP sockets are not available in
// every environment (containers, sandboxes), so permission failures skip.
func TestAccessICMPDelay(t *testing.T) {
	delay, err := ICMPDelay("127.0.0.1", 2)
	if err != nil {
		if isPermissionErr(err) {
			t.Skipf("ICMP not permitted in this environment: %v", err)
		}
		t.Errorf("Failed to test ICMP: %v", err)
		return
	}
	t.Logf("target delay: %s", delay)
}

func TestSingleICMPDelay(t *testing.T) {
	delay, err := SingleICMPDelay("127.0.0.1")
	if err != nil {
		if isPermissionErr(err) {
			t.Skipf("ICMP not permitted in this environment: %v", err)
		}
		t.Errorf("Failed to test single ICMP: %v", err)
		return
	}
	t.Logf("single delay: %s", delay)
}

func TestICMPDelayInvalidIP(t *testing.T) {
	_, err := ICMPDelay("999.999.999.999", 1)
	if err == nil {
		t.Error("invalid IP should return error")
	}
}

func TestSingleICMPDelayInvalidIP(t *testing.T) {
	_, err := SingleICMPDelay("999.999.999.999")
	if err == nil {
		t.Error("invalid IP should return error")
	}
	if !strings.Contains(err.Error(), "send icmp fail") {
		t.Errorf("error should wrap dial failure, got: %v", err)
	}
}

func TestICMPDelayUnresolvableHost(t *testing.T) {
	// dial error path with a hostname that cannot resolve
	_, err := ICMPDelay("no-such-host.invalid.example", 1)
	if err == nil {
		t.Error("unresolvable host should return error")
	}
}

func TestCheckSum(t *testing.T) {
	// RFC 1071 example-ish: checksum of known bytes
	data := []byte{0x00, 0x01, 0xf2, 0x03, 0xf4, 0xf5, 0xf6, 0xf7}
	got := checkSum(data)
	if got == 0 {
		t.Error("checksum should not be zero for non-zero input")
	}

	// checksum of the checksummed packet should be 0 (property of internet checksum)
	data2 := append([]byte{}, data...)
	var cs [2]byte
	cs[0] = byte(got >> 8)
	cs[1] = byte(got)
	withSum := append(data2, cs[:]...)
	if s := checkSum(withSum); s != 0 {
		t.Errorf("checksum over packet+checksum = %#x, want 0", s)
	}
}

func TestGetICMP(t *testing.T) {
	icmp := getICMP(7)
	if icmp.Type != 8 {
		t.Errorf("echo request type = %d, want 8", icmp.Type)
	}
	if icmp.SequenceNum != 7 {
		t.Errorf("sequence = %d, want 7", icmp.SequenceNum)
	}
	if icmp.CheckSum == 0 {
		t.Error("checksum should be computed")
	}
}

func isPermissionErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return errors.Is(err, errors.New("permission denied")) ||
		strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "operation not permitted")
}
