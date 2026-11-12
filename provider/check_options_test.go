package provider

import (
	"errors"
	"strings"
	"testing"

	"github.com/nginxui/plugin-sdk-go/protocol"
)

// The recursive check is opt-in: a certificate that never turned it on only
// waits for the authoritative nameservers, as NGINX UI 2.x did. The false value
// of the older disable option, written by the 1.0.0 form for every
// certificate, does not turn it on.
func TestCheckOptionsRecursiveCheckIsOptIn(t *testing.T) {
	cases := []struct {
		name          string
		options       map[string]any
		skipRecursive bool
	}{
		{name: "no options", options: nil, skipRecursive: true},
		{name: "option missing", options: map[string]any{protocol.DNS01OptionDisableCNAME: true}, skipRecursive: true},
		{name: "legacy disable false", options: map[string]any{protocol.DNS01OptionDisableRecursiveNSPropagation: false}, skipRecursive: true},
		{name: "legacy disable true", options: map[string]any{protocol.DNS01OptionDisableRecursiveNSPropagation: true}, skipRecursive: true},
		{name: "turned off", options: map[string]any{optionCheckRecursiveNSPropagation: false}, skipRecursive: true},
		{name: "turned on", options: map[string]any{optionCheckRecursiveNSPropagation: true}, skipRecursive: false},
		{name: "turned on as string", options: map[string]any{optionCheckRecursiveNSPropagation: "true"}, skipRecursive: false},
		{
			name: "turned on with a legacy false",
			options: map[string]any{
				optionCheckRecursiveNSPropagation:                 true,
				protocol.DNS01OptionDisableRecursiveNSPropagation: false,
			},
			skipRecursive: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkOptions(protocol.DNS01CheckParams{FQDN: "_acme-challenge.example.com.", Value: "v", Options: tc.options}, Settings{})
			if got.DisableRecursiveNSPropagation != tc.skipRecursive {
				t.Fatalf("DisableRecursiveNSPropagation = %v, want %v", got.DisableRecursiveNSPropagation, tc.skipRecursive)
			}
		})
	}
}

// The other switches and the settings pass through unchanged.
func TestCheckOptionsKeepsTheOtherSwitches(t *testing.T) {
	params := protocol.DNS01CheckParams{
		FQDN:  "_acme-challenge.example.com.",
		Value: "v",
		Options: map[string]any{
			protocol.DNS01OptionDisableCNAME:                      true,
			protocol.DNS01OptionDisableAuthoritativeNSPropagation: true,
		},
	}
	got := checkOptions(params, Settings{RecursiveNameservers: []string{"9.9.9.9:53"}})

	if !got.DisableCNAME || !got.DisableAuthoritativeNSPropagation {
		t.Fatalf("switches = %+v, want CNAME and authoritative checks off", got)
	}
	if got.FQDN != params.FQDN || got.Value != params.Value {
		t.Fatalf("record = %q %q, want %q %q", got.FQDN, got.Value, params.FQDN, params.Value)
	}
	if len(got.RecursiveNameservers) != 1 || got.RecursiveNameservers[0] != "9.9.9.9:53" {
		t.Fatalf("RecursiveNameservers = %v, want the plugin setting", got.RecursiveNameservers)
	}
}

// Through the handler: with the option missing, a recursive resolver that
// never answers must not be what holds the check up.
func TestCheckSkipsRecursiveResolversByDefault(t *testing.T) {
	h := New()
	// Nothing listens here, every query fails at once.
	h.SetSettings(Settings{RecursiveNameservers: []string{"127.0.0.1:1"}})

	result, err := h.Check(t.Context(), protocol.DNS01CheckParams{
		FQDN:    "_acme-challenge.example.com.",
		Value:   "value",
		Options: map[string]any{protocol.DNS01OptionDisableCNAME: true},
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Ready {
		t.Fatal("Check reported ready without any nameserver answering")
	}
	if strings.HasPrefix(result.Detail, "recursive nameservers:") {
		t.Fatalf("Detail = %q, the recursive check ran although the certificate never asked for it", result.Detail)
	}
}

// With both nameserver checks off the plugin must not report the record as
// ready on the spot: it hands the wait back to the host, which waits a fixed
// time before asking for validation.
func TestCheckWithBothChecksOffDefersToTheHost(t *testing.T) {
	h := New()
	_, err := h.Check(t.Context(), protocol.DNS01CheckParams{
		FQDN:    "_acme-challenge.example.com.",
		Value:   "value",
		Options: map[string]any{protocol.DNS01OptionDisableAuthoritativeNSPropagation: true},
	})

	var rpcErr *protocol.Error
	if !errors.As(err, &rpcErr) || rpcErr.Code != protocol.CodeUnsupported {
		t.Fatalf("Check = %v, want an unsupported error so the host waits instead", err)
	}
}

// With the authoritative check off and the recursive one on, the recursive
// resolvers are still queried.
func TestCheckWithOnlyTheRecursiveCheckQueriesIt(t *testing.T) {
	h := New()
	h.SetSettings(Settings{RecursiveNameservers: []string{"127.0.0.1:1"}})

	result, err := h.Check(t.Context(), protocol.DNS01CheckParams{
		FQDN:  "_acme-challenge.example.com.",
		Value: "value",
		Options: map[string]any{
			protocol.DNS01OptionDisableCNAME:                      true,
			protocol.DNS01OptionDisableAuthoritativeNSPropagation: true,
			optionCheckRecursiveNSPropagation:                     true,
		},
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Ready || !strings.HasPrefix(result.Detail, "recursive nameservers:") {
		t.Fatalf("result = %+v, want not ready on the recursive check", result)
	}
}
