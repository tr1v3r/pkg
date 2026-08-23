package netool

import (
	"net"
	"time"

	"github.com/miekg/dns"
)

var dnsClient = &dns.Client{Timeout: 600 * time.Millisecond}

// LookupIP ...
func LookupIP(domain string) ([]net.IP, error) {
	return net.LookupIP(domain)
}

// LookupWithServer resolves domain via the given DNS servers.
// Each server may be a bare host (defaults to port 53), a "host:port" pair,
// or a bracketed IPv6 literal like "[::1]" or "[::1]:5353".
func LookupWithServer(domain string, servers []string, maxRetry int) (a []string, cname []string, ns []string, lastErr error) {
	for _, server := range servers {
		addr := server
		if _, _, err := net.SplitHostPort(server); err != nil {
			// bare host or bare IPv6 literal: normalize with the default port
			addr = net.JoinHostPort(server, "53")
		}
		for i := 0; i < maxRetry; i++ {
			m := dns.Msg{}
			m.SetQuestion(domain+".", dns.TypeA)
			r, _, err := dnsClient.Exchange(&m, addr)
			if err != nil {
				lastErr = err
				continue
			}

			if r.Answer == nil {
				continue
			}

			for _, ans := range r.Answer {
				switch ans := ans.(type) {
				case *dns.CNAME:
					cname = append(cname, ans.Target)
				case *dns.A:
					a = append(a, ans.A.String())
				}
			}

			for _, ans := range r.Ns {
				if ans, ok := ans.(*dns.SOA); ok {
					ns = append(ns, ans.Ns)
				}
			}

			lastErr = nil
			break
		}
		if lastErr == nil {
			break
		}
	}
	return a, cname, ns, lastErr
}
