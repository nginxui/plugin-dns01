package check_test

import (
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"

	"github.com/0xJacky/nginx-ui-plugin-dns01/check"
)

const (
	zone      = "example.com."
	challenge = "_acme-challenge.example.com."
	aliased   = "_acme-challenge.example.org."
	txtValue  = "MsijOYZxqyjGnFGwhjrhfg-Xgbl5r68WPda0J9EgqqI"
)

// startServer serves handler over UDP and TCP on a loopback port and returns
// "127.0.0.1:<port>".
func startServer(t *testing.T, handler dns.HandlerFunc) string {
	t.Helper()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	port := pc.LocalAddr().(*net.UDPAddr).Port

	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		_ = pc.Close()
		t.Fatalf("listen tcp: %v", err)
	}

	udp := &dns.Server{PacketConn: pc, Handler: handler}
	tcp := &dns.Server{Listener: ln, Handler: handler}

	go func() { _ = udp.ActivateAndServe() }()
	go func() { _ = tcp.ActivateAndServe() }()

	t.Cleanup(func() {
		_ = udp.Shutdown()
		_ = tcp.Shutdown()
	})

	return net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
}

// zoneHandler answers the queries a successful propagation check makes.
// followCNAME makes the aliased name point at the challenge name.
func zoneHandler(t *testing.T, value string) dns.HandlerFunc {
	t.Helper()

	return func(w dns.ResponseWriter, req *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(req)
		m.Authoritative = true

		if len(req.Question) == 0 {
			_ = w.WriteMsg(m)
			return
		}

		q := req.Question[0]

		switch {
		case q.Name == aliased && q.Qtype == dns.TypeTXT:
			m.Answer = append(m.Answer, &dns.CNAME{
				Hdr:    dns.RR_Header{Name: aliased, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: 60},
				Target: challenge,
			})

		case q.Name == challenge && q.Qtype == dns.TypeTXT:
			m.Answer = append(m.Answer, &dns.TXT{
				Hdr: dns.RR_Header{Name: challenge, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 60},
				Txt: []string{value},
			})

		case q.Name == zone && q.Qtype == dns.TypeSOA:
			m.Answer = append(m.Answer, soa())

		case q.Name == zone && q.Qtype == dns.TypeNS:
			m.Answer = append(m.Answer, &dns.NS{
				Hdr: dns.RR_Header{Name: zone, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 60},
				Ns:  "127.0.0.1.",
			})
		}

		_ = w.WriteMsg(m)
	}
}

func soa() *dns.SOA {
	return &dns.SOA{
		Hdr:     dns.RR_Header{Name: zone, Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: 60},
		Ns:      "ns1." + zone,
		Mbox:    "hostmaster." + zone,
		Serial:  1,
		Refresh: 3600,
		Retry:   600,
		Expire:  86400,
		Minttl:  60,
	}
}

func baseOptions(addr string) check.Options {
	port := addr[strings.LastIndex(addr, ":")+1:]

	return check.Options{
		FQDN:                 challenge,
		Value:                txtValue,
		RecursiveNameservers: []string{addr},
		AuthoritativeNSPort:  port,
		QueryTimeout:         2 * time.Second,
	}
}

func TestPropagatedOnRecursiveAndAuthoritativeNameservers(t *testing.T) {
	addr := startServer(t, zoneHandler(t, txtValue))

	res, err := check.Propagated(t.Context(), baseOptions(addr))
	if err != nil {
		t.Fatalf("Propagated: %v", err)
	}
	if !res.Ready {
		t.Fatalf("Ready = false, detail = %q", res.Detail)
	}
	if res.EffectiveFQDN != "_acme-challenge.example.com" {
		t.Fatalf("effective fqdn = %q", res.EffectiveFQDN)
	}
}

func TestNotPropagatedWhenTheValueDiffers(t *testing.T) {
	addr := startServer(t, zoneHandler(t, "some-other-value"))

	res, err := check.Propagated(t.Context(), baseOptions(addr))
	if err != nil {
		t.Fatalf("Propagated: %v", err)
	}
	if res.Ready {
		t.Fatal("Ready = true for a record that does not match")
	}
	if !strings.Contains(res.Detail, "did not return the expected TXT record") {
		t.Fatalf("detail = %q", res.Detail)
	}
}

func TestCNAMEIsFollowed(t *testing.T) {
	addr := startServer(t, zoneHandler(t, txtValue))

	opts := baseOptions(addr)
	opts.FQDN = aliased

	res, err := check.Propagated(t.Context(), opts)
	if err != nil {
		t.Fatalf("Propagated: %v", err)
	}
	if !res.Ready {
		t.Fatalf("Ready = false, detail = %q", res.Detail)
	}
	if res.EffectiveFQDN != "_acme-challenge.example.com" {
		t.Fatalf("effective fqdn = %q, want the CNAME target", res.EffectiveFQDN)
	}
}

func TestDisableCNAMEKeepsTheOriginalName(t *testing.T) {
	addr := startServer(t, zoneHandler(t, txtValue))

	opts := baseOptions(addr)
	opts.FQDN = aliased
	opts.DisableCNAME = true

	res, err := check.Propagated(t.Context(), opts)
	if err != nil {
		t.Fatalf("Propagated: %v", err)
	}
	if res.Ready {
		t.Fatal("Ready = true although the alias carries no TXT record")
	}
	if res.EffectiveFQDN != "_acme-challenge.example.org" {
		t.Fatalf("effective fqdn = %q, want the original name", res.EffectiveFQDN)
	}
}

func TestDisableAuthoritativeNSPropagation(t *testing.T) {
	// The server answers the recursive TXT query but knows no NS record, so
	// the check only succeeds when the authoritative step is skipped.
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(req)
		m.Authoritative = true

		if len(req.Question) > 0 && req.Question[0].Name == challenge && req.Question[0].Qtype == dns.TypeTXT {
			m.Answer = append(m.Answer, &dns.TXT{
				Hdr: dns.RR_Header{Name: challenge, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 60},
				Txt: []string{txtValue},
			})
		}
		_ = w.WriteMsg(m)
	})

	addr := startServer(t, handler)

	opts := baseOptions(addr)
	opts.DisableAuthoritativeNSPropagation = true

	res, err := check.Propagated(t.Context(), opts)
	if err != nil {
		t.Fatalf("Propagated: %v", err)
	}
	if !res.Ready {
		t.Fatalf("Ready = false, detail = %q", res.Detail)
	}
}

func TestDisableRecursiveNSPropagation(t *testing.T) {
	addr := startServer(t, zoneHandler(t, txtValue))

	opts := baseOptions(addr)
	opts.DisableRecursiveNSPropagation = true

	res, err := check.Propagated(t.Context(), opts)
	if err != nil {
		t.Fatalf("Propagated: %v", err)
	}
	if !res.Ready {
		t.Fatalf("Ready = false, detail = %q", res.Detail)
	}
}

func TestMissingZoneIsReportedAsNotReady(t *testing.T) {
	// A server that answers nothing: the zone apex cannot be found.
	addr := startServer(t, func(w dns.ResponseWriter, req *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(req)
		if len(req.Question) > 0 && req.Question[0].Qtype == dns.TypeTXT {
			m.Answer = append(m.Answer, &dns.TXT{
				Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 60},
				Txt: []string{txtValue},
			})
		}
		_ = w.WriteMsg(m)
	})

	res, err := check.Propagated(t.Context(), baseOptions(addr))
	if err != nil {
		t.Fatalf("Propagated: %v", err)
	}
	if res.Ready {
		t.Fatal("Ready = true without a zone")
	}
	if !strings.Contains(res.Detail, "authoritative nameservers") {
		t.Fatalf("detail = %q", res.Detail)
	}
}

func TestInvalidInput(t *testing.T) {
	if _, err := check.Propagated(t.Context(), check.Options{Value: txtValue}); err == nil {
		t.Fatal("an empty fqdn was accepted")
	}
	if _, err := check.Propagated(t.Context(), check.Options{FQDN: challenge}); err == nil {
		t.Fatal("an empty value was accepted")
	}
}
