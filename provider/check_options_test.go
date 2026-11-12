package provider

import (
	"strings"
	"testing"

	"github.com/nginxui/plugin-sdk-go/protocol"
)

// The recursive check is opt-in: a certificate that never set the option only
// waits for the authoritative nameservers, as NGINX UI 2.x did.
func TestCheckOptionsRecursiveCheckIsOptIn(t *testing.T) {
	cases := []struct {
		name          string
		options       map[string]any
		skipRecursive bool
	}{
		{name: "no options", options: nil, skipRecursive: true},
		{name: "option missing", options: map[string]any{protocol.DNS01OptionDisableCNAME: true}, skipRecursive: true},
		{name: "disabled", options: map[string]any{protocol.DNS01OptionDisableRecursiveNSPropagation: true}, skipRecursive: true},
		{name: "turned on", options: map[string]any{protocol.DNS01OptionDisableRecursiveNSPropagation: false}, skipRecursive: false},
		{name: "turned on as string", options: map[string]any{protocol.DNS01OptionDisableRecursiveNSPropagation: "false"}, skipRecursive: false},
		{name: "turned on as number", options: map[string]any{protocol.DNS01OptionDisableRecursiveNSPropagation: float64(0)}, skipRecursive: false},
		{name: "unreadable value", options: map[string]any{protocol.DNS01OptionDisableRecursiveNSPropagation: "maybe"}, skipRecursive: true},
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
