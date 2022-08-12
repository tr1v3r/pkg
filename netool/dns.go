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

// LookupWithServer ...
func LookupWithServer(domain string, servers []string, maxRetry int) (a []string, cname []string, ns []string, lastErr error) {
	for _, server := range servers {
		for i := 0; i < maxRetry; i++ {
			m := dns.Msg{}
			m.SetQuestion(domain+".", dns.TypeA)
			r, _, err := dnsClient.Exchange(&m, server+":53")
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
				switch ans := ans.(type) {
				case *dns.SOA:
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
