// Package check answers dns01.check: it verifies that the challenge TXT record
// is visible before the host tells the ACME server to validate.
//
// It is a standalone port of lego's unexported checkDNSPropagation
// (challenge/dns01/dns_challenge_precheck.go). lego keeps its DNS client in
// challenge/internal, which cannot be imported from outside the lego module,
// so the queries are reimplemented on miekg/dns. The zone apex lookup still
// reuses lego's exported dns01.Client.
package check

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	sdk "github.com/0xJacky/nginx-ui-plugin-sdk-go"
	"github.com/0xJacky/nginx-ui-plugin-sdk-go/protocol"
	"github.com/go-acme/lego/v5/challenge/dns01"
	"github.com/miekg/dns"
)

// DefaultQueryTimeout bounds a single DNS exchange.
const DefaultQueryTimeout = 10 * time.Second

// defaultResolvConf is the resolver configuration read when the user did not
// configure recursive nameservers.
const defaultResolvConf = "/etc/resolv.conf"

// fallbackNameservers are used when the system resolvers cannot be read.
var fallbackNameservers = []string{
	"1.1.1.1:53",
	"1.0.0.1:53",
	"[2606:4700:4700::1111]:53",
	"[2606:4700:4700::1001]:53",
}

// maxCNAMEDepth stops a CNAME loop from spinning out of control.
const maxCNAMEDepth = 50

// Options describes one propagation check.
type Options struct {
	// FQDN is the challenge record name, with or without a trailing dot.
	FQDN string
	// Value is the expected TXT payload.
	Value string
	// RecursiveNameservers are queried for the recursive check and for the
	// zone apex lookup. Empty means the system resolvers.
	RecursiveNameservers []string
	// DisableCNAME stops the check from following the challenge record's CNAME.
	DisableCNAME bool
	// DisableAuthoritativeNSPropagation skips the authoritative nameservers.
	DisableAuthoritativeNSPropagation bool
	// DisableRecursiveNSPropagation skips the recursive nameservers.
	DisableRecursiveNSPropagation bool
	// QueryTimeout bounds one exchange. Zero means DefaultQueryTimeout.
	QueryTimeout time.Duration
	// AuthoritativeNSPort is the port used to reach the authoritative
	// nameservers. Empty means 53. For testing purposes only.
	AuthoritativeNSPort string
}

// Propagated reports whether the expected TXT record is visible. A record that
// has not propagated yet is not an error: the result carries Ready false and a
// Detail the host can show while it keeps polling.
func Propagated(ctx context.Context, opts Options) (protocol.DNS01CheckResult, error) {
	if strings.TrimSpace(opts.FQDN) == "" {
		return protocol.DNS01CheckResult{}, sdk.InvalidConfig("fqdn", "the challenge fqdn is required")
	}
	if opts.Value == "" {
		return protocol.DNS01CheckResult{}, sdk.InvalidConfig("value", "the challenge value is required")
	}

	c := newClient(opts)
	fqdn := dns.Fqdn(strings.TrimSpace(opts.FQDN))
	effective := fqdn

	if !opts.DisableCNAME {
		resolved, err := c.resolveCNAME(ctx, fqdn)
		if err != nil {
			return notReady(effective, fmt.Sprintf("initial recursive nameserver: %v", err)), nil
		}
		effective = resolved
	}

	if !opts.DisableRecursiveNSPropagation {
		if err := c.checkNameservers(ctx, effective, c.recursive, false, true); err != nil {
			return notReady(effective, fmt.Sprintf("recursive nameservers: %v", err)), nil
		}
	}

	if opts.DisableAuthoritativeNSPropagation {
		return protocol.DNS01CheckResult{
			Ready:         true,
			EffectiveFQDN: dns01.UnFqdn(effective),
			Detail:        "the record is visible on the recursive nameservers",
		}, nil
	}

	nameservers, err := c.authoritativeNameservers(ctx, effective)
	if err != nil {
		return notReady(effective, fmt.Sprintf("authoritative nameservers: %v", err)), nil
	}
	if err := c.checkNameservers(ctx, effective, nameservers, true, false); err != nil {
		return notReady(effective, fmt.Sprintf("authoritative nameservers: %v", err)), nil
	}

	return protocol.DNS01CheckResult{
		Ready:         true,
		EffectiveFQDN: dns01.UnFqdn(effective),
		Detail:        fmt.Sprintf("the record is visible on %d authoritative nameserver(s)", len(nameservers)),
	}, nil
}

func notReady(fqdn, detail string) protocol.DNS01CheckResult {
	return protocol.DNS01CheckResult{
		Ready:         false,
		EffectiveFQDN: dns01.UnFqdn(fqdn),
		Detail:        detail,
	}
}

// client holds the resolvers and the miekg/dns clients used for one check.
type client struct {
	recursive []string
	value     string
	nsPort    string

	udp *dns.Client
	tcp *dns.Client

	// zone resolves the zone apex through lego's exported client.
	zone *dns01.Client
}

func newClient(opts Options) *client {
	timeout := opts.QueryTimeout
	if timeout <= 0 {
		timeout = DefaultQueryTimeout
	}

	nsPort := opts.AuthoritativeNSPort
	if nsPort == "" {
		nsPort = "53"
	}

	recursive := normalizeNameservers(opts.RecursiveNameservers)
	if len(recursive) == 0 {
		recursive = systemNameservers()
	}

	legoOpts := dns01.NewOptions()
	legoOpts.RecursiveNameservers = recursive
	legoOpts.Timeout = timeout

	return &client{
		recursive: recursive,
		value:     opts.Value,
		nsPort:    nsPort,
		udp:       &dns.Client{Net: "udp", Timeout: timeout},
		tcp:       &dns.Client{Net: "tcp", Timeout: timeout},
		zone:      dns01.NewClient(legoOpts),
	}
}

// normalizeNameservers makes sure every entry carries a port.
func normalizeNameservers(servers []string) []string {
	var out []string
	for _, s := range servers {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, _, err := net.SplitHostPort(s); err != nil {
			s = net.JoinHostPort(s, "53")
		}
		out = append(out, s)
	}
	return out
}

// systemNameservers reads the resolver configuration, falling back to public
// resolvers the way lego does.
func systemNameservers() []string {
	config, err := dns.ClientConfigFromFile(defaultResolvConf)
	if err == nil && len(config.Servers) > 0 {
		return normalizeNameservers(config.Servers)
	}
	return fallbackNameservers
}

// exchange sends one query, retrying over TCP when the answer is truncated.
func (c *client) exchange(ctx context.Context, msg *dns.Msg, ns string) (*dns.Msg, error) {
	r, _, err := c.udp.ExchangeContext(ctx, msg, ns)
	if r != nil && r.Truncated {
		r, _, err = c.tcp.ExchangeContext(ctx, msg, ns)
	}
	if err != nil {
		return r, fmt.Errorf("DNS call to %s failed: %w", ns, err)
	}
	return r, nil
}

// query asks the given nameservers in order and returns the first answer that
// carries records.
func (c *client) query(ctx context.Context, fqdn string, rtype uint16, nameservers []string, recursive bool) (*dns.Msg, error) {
	if len(nameservers) == 0 {
		return nil, errors.New("empty list of nameservers")
	}

	msg := new(dns.Msg)
	msg.SetQuestion(fqdn, rtype)
	msg.SetEdns0(4096, false)
	msg.RecursionDesired = recursive

	var (
		last    *dns.Msg
		lastErr error
		all     error
	)

	for _, ns := range nameservers {
		r, err := c.exchange(ctx, msg.Copy(), ns)
		last, lastErr = r, err
		if err == nil && r != nil && len(r.Answer) > 0 {
			return r, nil
		}
		all = errors.Join(all, err)
	}

	if lastErr != nil {
		return last, all
	}
	return last, nil
}

// resolveCNAME follows the challenge record's CNAME chain on the recursive
// nameservers, mirroring lego's resolveCNAME.
func (c *client) resolveCNAME(ctx context.Context, fqdn string) (string, error) {
	current := fqdn

	for range maxCNAMEDepth {
		r, err := c.query(ctx, current, dns.TypeTXT, c.recursive, true)
		if err != nil {
			return "", err
		}
		if r == nil || r.Rcode != dns.RcodeSuccess {
			return current, nil
		}

		target := extractCNAME(r, current)
		if target == "" || target == current {
			return current, nil
		}
		current = target
	}

	return current, nil
}

func extractCNAME(msg *dns.Msg, name string) string {
	for _, rr := range msg.Answer {
		cn, ok := rr.(*dns.CNAME)
		if !ok {
			continue
		}
		if strings.EqualFold(cn.Hdr.Name, name) {
			return cn.Target
		}
	}
	return ""
}

// authoritativeNameservers returns the nameservers of the zone holding fqdn.
func (c *client) authoritativeNameservers(ctx context.Context, fqdn string) ([]string, error) {
	zone, err := c.zone.FindZoneByFqdn(ctx, fqdn)
	if err != nil {
		return nil, fmt.Errorf("could not find zone: %w", err)
	}

	r, err := c.query(ctx, dns.Fqdn(zone), dns.TypeNS, c.recursive, true)
	if err != nil {
		return nil, fmt.Errorf("NS call failed: %w", err)
	}

	var out []string
	if r != nil {
		for _, rr := range r.Answer {
			if ns, ok := rr.(*dns.NS); ok {
				out = append(out, strings.ToLower(ns.Ns))
			}
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("[zone=%s] could not determine authoritative nameservers", zone)
	}
	return out, nil
}

// checkNameservers requires every given nameserver to answer with the expected
// TXT record.
func (c *client) checkNameservers(ctx context.Context, fqdn string, nameservers []string, addPort, recursive bool) error {
	if len(nameservers) == 0 {
		return errors.New("empty list of nameservers")
	}

	for _, ns := range nameservers {
		target := ns
		if addPort {
			target = net.JoinHostPort(strings.TrimSuffix(ns, "."), c.nsPort)
		}

		r, err := c.query(ctx, fqdn, dns.TypeTXT, []string{target}, recursive)
		if err != nil {
			return err
		}
		if r == nil {
			return fmt.Errorf("NS %s returned no answer for %s", target, fqdn)
		}
		if r.Rcode != dns.RcodeSuccess {
			return fmt.Errorf("NS %s returned %s for %s", target, dns.RcodeToString[r.Rcode], fqdn)
		}

		if !containsTXT(r, c.value) {
			return fmt.Errorf("NS %s did not return the expected TXT record for %s", target, fqdn)
		}
	}

	return nil
}

func containsTXT(msg *dns.Msg, value string) bool {
	for _, rr := range msg.Answer {
		txt, ok := rr.(*dns.TXT)
		if !ok {
			continue
		}
		if strings.Join(txt.Txt, "") == value {
			return true
		}
	}
	return false
}
