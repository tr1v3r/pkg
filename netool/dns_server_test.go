package netool

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

// startTestDNSServer runs an in-process DNS server and returns its
// "127.0.0.1:port" address. t.Cleanup shuts it down.
func startTestDNSServer(t *testing.T) string {
	t.Helper()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen packet fail: %s", err)
	}

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, q *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(q)

		name := ""
		if len(q.Question) > 0 {
			name = q.Question[0].Name
		}

		switch name {
		case "ok.example.com.":
			m.Answer = append(m.Answer,
				&dns.CNAME{
					Hdr:    dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: 60},
					Target: "www.example.com.",
				},
				&dns.A{
					Hdr: dns.RR_Header{Name: "www.example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
					A:   net.ParseIP("93.184.216.34"),
				},
			)
			m.Ns = append(m.Ns, &dns.SOA{
				Hdr:     dns.RR_Header{Name: name, Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: 60},
				Ns:      "ns1.example.com.",
				Mbox:    "hostmaster.example.com.",
				Serial:  2024010101,
				Refresh: 7200,
				Retry:   3600,
				Expire:  1209600,
				Minttl:  3600,
			})
		default:
			// no answer records for anything else
		}

		_ = w.WriteMsg(m)
	})

	server := &dns.Server{PacketConn: pc, Handler: handler}
	go func() { _ = server.ActivateAndServe() }()
	t.Cleanup(func() { _ = server.Shutdown() })

	return pc.LocalAddr().String()
}

func TestLookupWithServer_LocalServer(t *testing.T) {
	addr := startTestDNSServer(t)

	a, cname, ns, err := LookupWithServer("ok.example.com", []string{addr}, 1)
	if err != nil {
		t.Fatalf("lookup fail: %s", err)
	}

	if len(a) != 1 || a[0] != "93.184.216.34" {
		t.Errorf("a records = %v, want [93.184.216.34]", a)
	}
	if len(cname) != 1 || cname[0] != "www.example.com." {
		t.Errorf("cname records = %v, want [www.example.com.]", cname)
	}
	if len(ns) != 1 || ns[0] != "ns1.example.com." {
		t.Errorf("ns records = %v, want [ns1.example.com.]", ns)
	}
}

func TestLookupWithServer_EmptyAnswer(t *testing.T) {
	addr := startTestDNSServer(t)

	// Server answers authoritatively but with no records: the retry loop
	// exhausts without transport errors and returns empty result sets.
	a, cname, ns, err := LookupWithServer("empty.example.com", []string{addr}, 2)
	if err != nil {
		t.Fatalf("lookup fail: %s", err)
	}
	if len(a) != 0 || len(cname) != 0 || len(ns) != 0 {
		t.Errorf("empty answer should return no records: a=%v cname=%v ns=%v", a, cname, ns)
	}
}

func TestLookupWithServer_ServerFallback(t *testing.T) {
	live := startTestDNSServer(t)

	// Reserve a dead port by opening and closing a listener.
	deadPC, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen fail: %s", err)
	}
	deadAddr := deadPC.LocalAddr().String()
	_ = deadPC.Close()

	a, _, _, err := LookupWithServer("ok.example.com", []string{deadAddr, live}, 1)
	if err != nil {
		t.Fatalf("lookup with fallback fail: %s", err)
	}
	if len(a) != 1 || a[0] != "93.184.216.34" {
		t.Errorf("a records = %v, want result from live server", a)
	}
}

func TestLookupWithServer_AllServersDead(t *testing.T) {
	deadPC, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen fail: %s", err)
	}
	deadAddr := deadPC.LocalAddr().String()
	_ = deadPC.Close()

	_, _, _, lastErr := LookupWithServer("ok.example.com", []string{deadAddr}, 1)
	if lastErr == nil {
		t.Error("all servers dead should return the last transport error")
	}
}
