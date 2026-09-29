package catalog

import "testing"

func TestClean(t *testing.T) {
	tests := []struct {
		in   string
		want cleaned
	}{
		{"API key", cleaned{Text: "API key"}},
		{"API token with DNS:Edit permission (since v3.1.0)", cleaned{Text: "API token with DNS:Edit permission"}},
		{"Time between DNS propagation check in seconds (Default: 2)", cleaned{Text: "Time between DNS propagation check", Default: "2", Seconds: true}},
		{"API request timeout in seconds (Default: )", cleaned{Text: "API request timeout", Seconds: true}},
		{"Region ID (Default: cn-hangzhou)", cleaned{Text: "Region ID", Default: "cn-hangzhou"}},
		{"Authorization type. Possible values: 'instance_principal', 'user_principal', ''. (Default: '')", cleaned{Text: "Authorization type. Possible values: 'instance_principal', 'user_principal', ''"}},
		{"mode: 'anycast' or 'zones' (for FreeDNS) (default: 'anycast')", cleaned{Text: "mode: 'anycast' or 'zones' (for FreeDNS)", Default: "anycast"}},
		{"API endpoint URL, defaults to https://api.autodns.com/v1/", cleaned{Text: "API endpoint URL", Default: "https://api.autodns.com/v1/"}},
		{"Set to true to use private zones only (default: use public zones only)", cleaned{Text: "Set to true to use private zones only (default: use public zones only)"}},
		{"STS Security Token (optional)", cleaned{Text: "STS Security Token", Optional: true}},
		{"Your instance RAM role (https://www.alibabacloud.com/help/en/ecs)", cleaned{Text: "Your instance RAM role", Link: "https://www.alibabacloud.com/help/en/ecs"}},
		{"API key `<prefix>.<secret>` https://developer.hosting.ionos.com/docs/getstarted", cleaned{Text: "API key `<prefix>.<secret>`", Link: "https://developer.hosting.ionos.com/docs/getstarted"}},
		{"[Documentation](https://cloud.google.com/docs/authentication)", cleaned{Text: "Documentation", Link: "https://cloud.google.com/docs/authentication"}},
		{"API server URL (ex: https://panel.example.com:8080)", cleaned{Text: "API server URL", Example: "https://panel.example.com:8080"}},
		{"The base URL (e.g., https://foo.example.com:8443/API)", cleaned{Text: "The base URL", Example: "https://foo.example.com:8443/API"}},
		{"API endpoint. Ex: https://api.loopia.se/RPCSERV", cleaned{Text: "API endpoint", Example: "https://api.loopia.se/RPCSERV"}},
		{"The API token.", cleaned{Text: "The API token"}},
	}
	for _, tt := range tests {
		if got := clean(tt.in); got != tt.want {
			t.Errorf("clean(%q)\n got %+v\nwant %+v", tt.in, got, tt.want)
		}
	}
}

func TestLabel(t *testing.T) {
	tests := []struct{ in, label, help string }{
		{"API key", "API key", ""},
		{"The API user ID", "API user ID", ""},
		{"Your instance RAM role", "Instance RAM role", ""},
		{"username", "Username", ""},
		{"The customer email address. You can also use the customer id instead", "Customer email address", "You can also use the customer id instead."},
		{"Region name, used for the identity endpoint of the cloud", "Region name", "Used for the identity endpoint of the cloud."},
		{"Advanced ServiceDiscovery filter using Kusto query condition", "Advanced ServiceDiscovery filter using Kusto query condition", ""},
	}
	for _, tt := range tests {
		label, help := label(tt.in)
		if label != tt.label || help != tt.help {
			t.Errorf("label(%q) = %q, %q, want %q, %q", tt.in, label, help, tt.label, tt.help)
		}
	}
}

func TestAliasTarget(t *testing.T) {
	tests := []struct {
		in     string
		target string
		ok     bool
	}{
		{"Alias to CF_API_KEY", "CF_API_KEY", true},
		{"Alias on `OCI_REGION`", "OCI_REGION", true},
		{"Alias to CF_API_KEY.", "CF_API_KEY", true},
		{"API key", "", false},
		{"Alias to the API key", "", false},
	}
	for _, tt := range tests {
		target, ok := aliasTarget(tt.in)
		if target != tt.target || ok != tt.ok {
			t.Errorf("aliasTarget(%q) = %q, %v, want %q, %v", tt.in, target, ok, tt.target, tt.ok)
		}
	}
}

func TestIsSecret(t *testing.T) {
	tests := []struct {
		key, label string
		want       bool
	}{
		{"CF_DNS_API_TOKEN", "API token", true},
		{"CF_API_KEY", "Global API key", true},
		{"ALICLOUD_SECRET_KEY", "AccessKey secret", true},
		{"FOO_PASSWORD", "Password", true},
		{"FOO_APIKEY", "API key", true},
		{"FOO_PRIVATE_KEY", "Private key", true},
		{"DDNSS_KEY", "Update key", true},
		{"CLOUDNS_AUTH_PASSWORD", "Password for API user ID", true},
		{"DNSHOMEDE_CREDENTIALS", "Domain passwords", true},
		{"ALICLOUD_ACCESS_KEY", "Access key ID", false},
		{"GEHIRN_TOKEN_ID", "Token ID", false},
		{"FOO_USERNAME", "Username", false},
		{"FOO_EMAIL", "Account email", false},
		{"FOO_API_URL", "API endpoint URL", false},
		{"TRANSIP_PRIVATE_KEY_PATH", "Private key file", false},
		{"UCLOUD_PUBLIC_KEY", "Public key", false},
		{"NAMESURFER_API_KEY", "API key name", false},
		{"FOO_TTL", "TXT record TTL", false},
	}
	for _, tt := range tests {
		if got := isSecret(tt.key, tt.label); got != tt.want {
			t.Errorf("isSecret(%q, %q) = %v, want %v", tt.key, tt.label, got, tt.want)
		}
	}
}
