package catalog

import (
	"reflect"
	"testing"

	"github.com/nginxui/plugin-sdk-go/protocol"
)

func TestParseExample(t *testing.T) {
	tests := []struct {
		name    string
		example string
		want    []exampleBlock
	}{
		{
			name:    "single block",
			example: "FOO_TOKEN=x \\\nlego run --dns foo -d example.com\n",
			want:    nil,
		},
		{
			name: "bare or",
			example: `CLOUDFLARE_EMAIL=you@example.com \
CLOUDFLARE_API_KEY=b98 \
lego run --dns cloudflare -d example.com

# or

CLOUDFLARE_DNS_API_TOKEN=123 \
lego run --dns cloudflare -d example.com
`,
			want: []exampleBlock{
				{Keys: []string{"CLOUDFLARE_EMAIL", "CLOUDFLARE_API_KEY"}},
				{Keys: []string{"CLOUDFLARE_DNS_API_TOKEN"}},
			},
		},
		{
			name: "named blocks",
			example: `# Setup using instance RAM role
ALICLOUD_RAM_ROLE=lego \
lego run --dns alidns -d example.com

# Or, using credentials
ALICLOUD_ACCESS_KEY=abc \
ALICLOUD_SECRET_KEY=your-secret-key \
lego run --dns alidns -d example.com
`,
			want: []exampleBlock{
				{Name: "Instance RAM role", Keys: []string{"ALICLOUD_RAM_ROLE"}},
				{Name: "Credentials", Keys: []string{"ALICLOUD_ACCESS_KEY", "ALICLOUD_SECRET_KEY"}},
			},
		},
		{
			name: "names with colons",
			example: `# Application Key authentication:

OVH_APPLICATION_KEY=1 \
OVH_ENDPOINT=ovh-eu \
lego run

# Or Access Token:

OVH_ACCESS_TOKEN=xxx \
OVH_ENDPOINT=ovh-eu \
lego run

# or with API key:

OVH_API_KEY=zzz \
lego run
`,
			want: []exampleBlock{
				{Name: "Application Key authentication", Keys: []string{"OVH_APPLICATION_KEY", "OVH_ENDPOINT"}},
				{Name: "Access Token", Keys: []string{"OVH_ACCESS_TOKEN", "OVH_ENDPOINT"}},
				{Name: "API key", Keys: []string{"OVH_API_KEY"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseExample(tt.example); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseExample\n got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestNormalizeMethods(t *testing.T) {
	labels := map[string]string{"USER": "Username", "PASS": "Password", "KEY": "API key", "URL": "Server URL"}
	m := func(name string, keys ...string) protocol.DNS01ProviderMethod {
		return protocol.DNS01ProviderMethod{Name: name, Fields: keys}
	}
	tests := []struct {
		name string
		in   []protocol.DNS01ProviderMethod
		want []protocol.DNS01ProviderMethod
	}{
		{
			name: "shared keys move out and unnamed methods get labels",
			in:   []protocol.DNS01ProviderMethod{m("", "URL", "USER", "PASS"), m("Key", "URL", "KEY")},
			want: []protocol.DNS01ProviderMethod{m("Username and password", "USER", "PASS"), m("Key", "KEY")},
		},
		{
			name: "empty methods are dropped before sharing",
			in:   []protocol.DNS01ProviderMethod{m("File"), m("", "URL", "USER"), m("", "URL", "KEY")},
			want: []protocol.DNS01ProviderMethod{m("Username", "USER"), m("API key", "KEY")},
		},
		{
			name: "duplicates leave no choice",
			in:   []protocol.DNS01ProviderMethod{m("A", "KEY"), m("B", "KEY")},
			want: nil,
		},
		{
			name: "one method is no choice",
			in:   []protocol.DNS01ProviderMethod{m("A", "KEY")},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeMethods(tt.in, labels); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeMethods\n got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestJoinLabels(t *testing.T) {
	labels := map[string]string{"A": "Access key ID", "B": "Secret", "C": "STS token"}
	if got := joinLabels([]string{"A", "B", "C"}, labels); got != "Access key ID, secret and STS token" {
		t.Fatalf("joinLabels = %q", got)
	}
}
