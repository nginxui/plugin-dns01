package provider

import (
	"os"
	"reflect"
	"testing"

	"github.com/nginxui/plugin-dns01/catalog"
)

func TestEnvScopeRoundTrip(t *testing.T) {
	cfg, ok := catalog.Get("cloudflare")
	if !ok {
		t.Fatal("cloudflare is missing from the catalog")
	}

	// One variable already set, one absent: both must be back as they were.
	t.Setenv("CF_API_EMAIL", "previous@example.com")
	if err := os.Unsetenv("CF_DNS_API_TOKEN"); err != nil {
		t.Fatalf("unsetenv: %v", err)
	}

	scope := newEnvScope(cfg)
	err := scope.SetEnv(map[string]string{
		"CF_API_EMAIL":                   "  user@example.com  ",
		"CF_DNS_API_TOKEN":               `"quoted-token"`,
		"CLOUDFLARE_PROPAGATION_TIMEOUT": "180",
		"NOT_DECLARED_BY_THE_PROVIDER":   "ignored",
	})
	if err != nil {
		t.Fatalf("SetEnv: %v", err)
	}

	if got := os.Getenv("CF_API_EMAIL"); got != "user@example.com" {
		t.Fatalf("CF_API_EMAIL = %q", got)
	}
	if got := os.Getenv("CF_DNS_API_TOKEN"); got != "quoted-token" {
		t.Fatalf("CF_DNS_API_TOKEN = %q", got)
	}
	if got := os.Getenv("CLOUDFLARE_PROPAGATION_TIMEOUT"); got != "180" {
		t.Fatalf("CLOUDFLARE_PROPAGATION_TIMEOUT = %q", got)
	}
	if _, ok := os.LookupEnv("NOT_DECLARED_BY_THE_PROVIDER"); ok {
		t.Fatal("an undeclared variable was exported")
	}

	scope.CleanEnv()

	if got := os.Getenv("CF_API_EMAIL"); got != "previous@example.com" {
		t.Fatalf("CF_API_EMAIL after clean = %q, want the original value", got)
	}
	if _, ok := os.LookupEnv("CF_DNS_API_TOKEN"); ok {
		t.Fatal("CF_DNS_API_TOKEN was not unset")
	}
	if _, ok := os.LookupEnv("CLOUDFLARE_PROPAGATION_TIMEOUT"); ok {
		t.Fatal("CLOUDFLARE_PROPAGATION_TIMEOUT was not unset")
	}
}

func TestEnvScopeSetRawIsRestored(t *testing.T) {
	cfg, _ := catalog.Get("exec")

	scope := newEnvScope(cfg)
	if err := scope.SetRaw(envDisableCNAME, "true"); err != nil {
		t.Fatalf("SetRaw: %v", err)
	}
	if got := os.Getenv(envDisableCNAME); got != "true" {
		t.Fatalf("%s = %q", envDisableCNAME, got)
	}

	scope.CleanEnv()
	if _, ok := os.LookupEnv(envDisableCNAME); ok {
		t.Fatalf("%s was not restored", envDisableCNAME)
	}
}

func TestNormalizeEnvValue(t *testing.T) {
	cases := map[string]string{
		"  token  ":     "token",
		`"token"`:       "token",
		"'token'":       "token",
		`"tok\"en"`:     `tok"en`,
		`"unterminated`: `"unterminated`,
		"":              "",
		`"`:             `"`,
	}
	for in, want := range cases {
		if got := normalizeEnvValue(in); got != want {
			t.Fatalf("normalizeEnvValue(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseSettings(t *testing.T) {
	s := ParseSettings(map[string]any{
		"recursive_nameservers":               "1.1.1.1:53, 8.8.8.8 ,",
		"default_propagation_timeout_seconds": float64(180),
	})

	if len(s.RecursiveNameservers) != 2 ||
		s.RecursiveNameservers[0] != "1.1.1.1:53" ||
		s.RecursiveNameservers[1] != "8.8.8.8" {
		t.Fatalf("nameservers = %v", s.RecursiveNameservers)
	}
	if s.DefaultPropagationTimeout.Seconds() != 180 {
		t.Fatalf("timeout = %v", s.DefaultPropagationTimeout)
	}

	empty := ParseSettings(nil)
	if empty.RecursiveNameservers != nil || empty.DefaultPropagationTimeout != 0 {
		t.Fatalf("empty settings = %+v", empty)
	}
}

func TestParseSettingsNameserverList(t *testing.T) {
	cases := []struct {
		name string
		raw  any
		want []string
	}{
		{"json array", []any{"1.1.1.1:53", " 8.8.8.8 ", "", 42}, []string{"1.1.1.1:53", "8.8.8.8"}},
		{"string slice", []string{"9.9.9.9:53"}, []string{"9.9.9.9:53"}},
		{"legacy string", "1.1.1.1:53,8.8.8.8", []string{"1.1.1.1:53", "8.8.8.8"}},
		{"empty array", []any{}, nil},
		{"empty string", "", nil},
		{"null", nil, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseSettings(map[string]any{"recursive_nameservers": tc.raw}).RecursiveNameservers
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("nameservers = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestBoolOption(t *testing.T) {
	cases := []struct {
		options map[string]any
		want    bool
	}{
		{nil, false},
		{map[string]any{"disable_cname": true}, true},
		{map[string]any{"disable_cname": "true"}, true},
		{map[string]any{"disable_cname": "false"}, false},
		{map[string]any{"disable_cname": float64(1)}, true},
		{map[string]any{"disable_cname": "nonsense"}, false},
	}
	for _, c := range cases {
		if got := boolOption(c.options, "disable_cname"); got != c.want {
			t.Fatalf("boolOption(%v) = %v, want %v", c.options, got, c.want)
		}
	}
}

func TestMissingFieldFromLegoError(t *testing.T) {
	cfg, _ := catalog.Get("cloudflare")

	err := errString("cloudflare: some credentials information are missing: CF_API_EMAIL,CF_API_KEY")
	if got := missingField(cfg, map[string]string{}, err); got != "CF_API_EMAIL" {
		t.Fatalf("missingField = %q, want CF_API_EMAIL", got)
	}

	// Without a recognizable message, the first empty credential wins.
	other := errString("cloudflare: something else went wrong")
	if got := missingField(cfg, map[string]string{}, other); got != "CF_API_EMAIL" {
		t.Fatalf("missingField fallback = %q", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
