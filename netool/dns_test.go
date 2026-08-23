package netool

import (
	"testing"
)

func TestLookupIPLocalhost(t *testing.T) {
	ips, err := LookupIP("localhost")
	if err != nil {
		t.Fatalf("LookupIP(localhost) fail: %s", err)
	}
	if len(ips) == 0 {
		t.Fatal("LookupIP(localhost) returned no IPs")
	}

	found := false
	for _, ip := range ips {
		if ip.IsLoopback() {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("LookupIP(localhost) = %v, want a loopback address", ips)
	}
}

func TestLookupIPInvalidDomain(t *testing.T) {
	// invalid TLD cannot resolve; resolver availability varies, so only assert
	// that the call completes without panic.
	_, _ = LookupIP("invalid.<not-a-real-tld>")
}

func TestLookupWithServerUnreachable(t *testing.T) {
	// Nothing listens on port 53 of loopback in test environments.
	a, cname, ns, err := LookupWithServer("example.com", []string{"127.0.0.1"}, 2)
	if err == nil {
		t.Log("unexpected local DNS resolver on 127.0.0.1:53; skipping strict assertion")
		return
	}
	if len(a) != 0 || len(cname) != 0 || len(ns) != 0 {
		t.Errorf("unreachable server should not produce records: a=%v cname=%v ns=%v", a, cname, ns)
	}
}

func TestLookupWithServerEmptyServers(t *testing.T) {
	a, cname, ns, err := LookupWithServer("example.com", nil, 3)
	if err != nil {
		t.Errorf("empty server list err = %v, want nil", err)
	}
	if len(a) != 0 || len(cname) != 0 || len(ns) != 0 {
		t.Errorf("empty server list should return no records: a=%v cname=%v ns=%v", a, cname, ns)
	}
}
