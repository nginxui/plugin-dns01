package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/go-acme/lego/v5/challenge/dns01"
	"github.com/miekg/dns"
	sdk "github.com/nginxui/plugin-sdk-go"
)

// fakeResolver is a loopback recursive resolver that holds the SOA of one zone
// apex and records every name it is asked about.
type fakeResolver struct {
	addr string
	zone string

	mu      sync.Mutex
	queries []string
}

// startResolver serves the SOA of zone over UDP and TCP on a loopback port.
// Every other name gets an empty NOERROR answer, which makes lego's zone lookup
// move on to the parent name.
func startResolver(t *testing.T, zone string) *fakeResolver {
	t.Helper()

	r := &fakeResolver{zone: dns.Fqdn(zone)}

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

	handler := dns.HandlerFunc(r.serveDNS)
	udp := &dns.Server{PacketConn: pc, Handler: handler}
	tcp := &dns.Server{Listener: ln, Handler: handler}
	go func() { _ = udp.ActivateAndServe() }()
	go func() { _ = tcp.ActivateAndServe() }()
	t.Cleanup(func() {
		_ = udp.Shutdown()
		_ = tcp.Shutdown()
	})

	r.addr = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	return r
}

func (r *fakeResolver) serveDNS(w dns.ResponseWriter, req *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(req)
	m.RecursionAvailable = true

	if len(req.Question) > 0 {
		q := req.Question[0]
		name := strings.ToLower(q.Name)

		r.mu.Lock()
		r.queries = append(r.queries, name)
		r.mu.Unlock()

		if q.Qtype == dns.TypeSOA && name == r.zone {
			m.Answer = append(m.Answer, &dns.SOA{
				Hdr:     dns.RR_Header{Name: r.zone, Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: 300},
				Ns:      "ns1." + r.zone,
				Mbox:    "hostmaster." + r.zone,
				Serial:  1,
				Refresh: 3600,
				Retry:   600,
				Expire:  86400,
				Minttl:  60,
			})
		}
	}

	_ = w.WriteMsg(m)
}

func (r *fakeResolver) names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.queries...)
}

// useSystemResolver stands in for the resolvers of /etc/resolv.conf: it makes
// lego's default DNS client query r for the rest of the test, so no test ever
// reaches the host's real resolvers.
func useSystemResolver(t *testing.T, r *fakeResolver) *dns01.Client {
	t.Helper()

	opts := dns01.NewOptions()
	opts.RecursiveNameservers = []string{r.addr}
	system := dns01.NewClient(opts)

	previous := dns01.DefaultClient()
	dns01.SetDefaultClient(system)
	t.Cleanup(func() { dns01.SetDefaultClient(previous) })

	return system
}

// fakeCloudflare answers the Cloudflare API calls lego makes for the zones it
// knows, and records the zone names asked for, the records created and
// deleted, and the default DNS client in place while each call was served.
type fakeCloudflare struct {
	zones map[string]bool

	mu       sync.Mutex
	asked    []string
	created  []string
	deleted  []string
	clients  []*dns01.Client
	sequence int
}

func newFakeCloudflare(zones ...string) *fakeCloudflare {
	f := &fakeCloudflare{zones: make(map[string]bool)}
	for _, z := range zones {
		f.zones[z] = true
	}
	return f
}

func (f *fakeCloudflare) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.clients = append(f.clients, dns01.DefaultClient())
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	switch {
	case r.Method == http.MethodGet && len(parts) == 1 && parts[0] == "zones":
		name := r.URL.Query().Get("name")
		f.asked = append(f.asked, name)
		result := []map[string]string{}
		if f.zones[name] {
			result = append(result, map[string]string{"id": "zone-" + name, "name": name})
		}
		writeCloudflare(w, result)

	case r.Method == http.MethodPost && len(parts) == 3 && parts[0] == "zones" && parts[2] == "dns_records":
		f.sequence++
		id := fmt.Sprintf("record-%d", f.sequence)
		f.created = append(f.created, parts[1]+"/"+id)
		writeCloudflare(w, map[string]string{"id": id})

	case r.Method == http.MethodDelete && len(parts) == 4 && parts[0] == "zones" && parts[2] == "dns_records":
		f.deleted = append(f.deleted, parts[1]+"/"+parts[3])
		writeCloudflare(w, map[string]string{"id": parts[3]})

	default:
		http.Error(w, "unexpected call "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
	}
}

func writeCloudflare(w http.ResponseWriter, result any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": result})
}

func cloudflareRequest(serverURL, domain, token string) sdk.DNS01Request {
	return sdk.DNS01Request{
		Provider: "cloudflare",
		Config: map[string]string{
			"CF_DNS_API_TOKEN":    "token",
			"CLOUDFLARE_BASE_URL": serverURL,
		},
		Domain:        domain,
		EffectiveFQDN: "_acme-challenge." + domain,
		Token:         token,
		KeyAuth:       token + ".key",
	}
}

// With split-horizon DNS the system resolver serves an internal zone the DNS
// vendor does not know about. The zone lookup the provider does in Present and
// CleanUp must go to the recursive nameservers set in the plugin settings, as
// NGINX UI 2.x did, or Cloudflare is asked for the internal zone and fails
// with "zone could not be found" (nginxui/plugin-dns01#4).
func TestZoneLookupUsesTheRecursiveNameservers(t *testing.T) {
	splitHorizon := startResolver(t, "www.example.test")
	system := useSystemResolver(t, splitHorizon)
	public := startResolver(t, "example.test")

	vendor := newFakeCloudflare("example.test")
	server := httptest.NewServer(vendor)
	defer server.Close()

	h := New()
	h.SetSettings(Settings{RecursiveNameservers: []string{public.addr}})
	req := cloudflareRequest(server.URL, "www.example.test", "token-1")

	if err := h.Present(context.Background(), req); err != nil {
		t.Fatalf("Present: %v", err)
	}
	if got := dns01.DefaultClient(); got != system {
		t.Fatal("Present left another default DNS client in place")
	}
	if err := h.CleanUp(context.Background(), req); err != nil {
		t.Fatalf("CleanUp: %v", err)
	}
	if got := dns01.DefaultClient(); got != system {
		t.Fatal("CleanUp left another default DNS client in place")
	}

	vendor.mu.Lock()
	defer vendor.mu.Unlock()
	for _, zone := range vendor.asked {
		if zone != "example.test" {
			t.Errorf("Cloudflare was asked for zone %q, want only example.test", zone)
		}
	}
	// The cleanup went through the provider Present built, which holds the
	// record ID.
	if len(vendor.created) != 1 || len(vendor.deleted) != 1 || vendor.created[0] != vendor.deleted[0] {
		t.Errorf("created %v, deleted %v, want the one published record deleted", vendor.created, vendor.deleted)
	}
	for i, c := range vendor.clients {
		if c == system {
			t.Errorf("vendor call %d ran with the system DNS client in place", i)
		}
	}

	if names := splitHorizon.names(); len(names) != 0 {
		t.Errorf("the system resolver was asked %v, want no query", names)
	}
	if len(public.names()) == 0 {
		t.Error("the configured recursive nameserver was never asked")
	}
}

// Without recursive nameservers in the settings nothing changes: the providers
// use lego's default DNS client, which reads the system resolvers.
func TestZoneLookupWithoutRecursiveNameserversUsesTheSystemResolver(t *testing.T) {
	resolver := startResolver(t, "example.test")
	system := useSystemResolver(t, resolver)

	vendor := newFakeCloudflare("example.test")
	server := httptest.NewServer(vendor)
	defer server.Close()

	h := New()
	h.SetSettings(Settings{})
	req := cloudflareRequest(server.URL, "www.example.test", "token-1")

	if err := h.Present(context.Background(), req); err != nil {
		t.Fatalf("Present: %v", err)
	}
	if err := h.CleanUp(context.Background(), req); err != nil {
		t.Fatalf("CleanUp: %v", err)
	}

	if got := dns01.DefaultClient(); got != system {
		t.Fatal("the default DNS client was replaced")
	}

	vendor.mu.Lock()
	defer vendor.mu.Unlock()
	if len(vendor.created) != 1 || len(vendor.deleted) != 1 {
		t.Errorf("created %v, deleted %v, want one record published and deleted", vendor.created, vendor.deleted)
	}
	for i, c := range vendor.clients {
		if c != system {
			t.Errorf("vendor call %d ran with another default DNS client in place", i)
		}
	}
	if len(resolver.names()) == 0 {
		t.Error("the system resolver was never asked")
	}
}

// The default DNS client is process wide. Challenges served at the same time
// for handlers with different recursive nameservers must each find their zone
// through their own nameservers and never through the other's.
func TestZoneLookupKeepsConcurrentChallengesApart(t *testing.T) {
	system := startResolver(t, "unused.test")
	useSystemResolver(t, system)
	resolverA := startResolver(t, "a.test")
	resolverB := startResolver(t, "b.test")

	vendor := newFakeCloudflare("a.test", "b.test")
	server := httptest.NewServer(vendor)
	defer server.Close()

	handlerA := New()
	handlerA.SetSettings(Settings{RecursiveNameservers: []string{resolverA.addr}})
	handlerB := New()
	handlerB.SetSettings(Settings{RecursiveNameservers: []string{resolverB.addr}})

	const perHandler = 8

	var wg sync.WaitGroup
	errs := make(chan error, 2*perHandler)
	run := func(h *Handler, domain, token string) {
		defer wg.Done()
		req := cloudflareRequest(server.URL, domain, token)
		if err := h.Present(context.Background(), req); err != nil {
			errs <- fmt.Errorf("Present %s %s: %w", domain, token, err)
			return
		}
		if err := h.CleanUp(context.Background(), req); err != nil {
			errs <- fmt.Errorf("CleanUp %s %s: %w", domain, token, err)
		}
	}
	for i := range perHandler {
		wg.Add(2)
		go run(handlerA, "www.a.test", fmt.Sprintf("token-a-%d", i))
		go run(handlerB, "www.b.test", fmt.Sprintf("token-b-%d", i))
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	for _, name := range resolverA.names() {
		if !strings.HasSuffix(name, ".a.test.") && name != "a.test." {
			t.Errorf("resolver A was asked about %s", name)
		}
	}
	for _, name := range resolverB.names() {
		if !strings.HasSuffix(name, ".b.test.") && name != "b.test." {
			t.Errorf("resolver B was asked about %s", name)
		}
	}
	if names := system.names(); len(names) != 0 {
		t.Errorf("the system resolver was asked %v, want no query", names)
	}

	vendor.mu.Lock()
	defer vendor.mu.Unlock()
	if len(vendor.created) != 2*perHandler || len(vendor.deleted) != 2*perHandler {
		t.Errorf("created %d and deleted %d records, want %d of each", len(vendor.created), len(vendor.deleted), 2*perHandler)
	}
}
