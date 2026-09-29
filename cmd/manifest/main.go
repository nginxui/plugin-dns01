// Command manifest regenerates plugin.json from the embedded provider catalog.
//
//	go run ./cmd/manifest
//
// The generated file is committed, so the plugin can be packaged without
// running the generator.
//
// With -platform it writes the plugin.json of one per-platform package
// instead: the committed manifest with server.executables narrowed to that
// platform, which is what build.sh puts into each archive.
//
//	go run ./cmd/manifest -platform linux-amd64 -out dist/stage/linux-amd64/plugin.json
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/nginxui/plugin-sdk-go/protocol"

	"github.com/nginxui/plugin-dns01/catalog"
)

// Plugin identity. Version is the single source of truth for the release
// artifacts; build.sh reads it back from the generated plugin.json.
const (
	PluginID          = "com.nginxui.dns01"
	PluginName        = "DNS-01 Challenge"
	PluginVersion     = "1.0.0"
	PluginDescription = "Validate domains for certificates through DNS records, with more than 200 DNS providers."
	MinNginxUIVersion = "2.7.0"
	IdleTimeout       = 300
)

// translations are the name and description in every language the plugin
// ships besides English, keyed by host locale code.
var translations = map[string]protocol.ManifestI18n{
	"zh_CN": {
		Name:        "DNS-01 验证",
		Description: "支持 200 多家 DNS 服务商，用 DNS 记录为证书验证域名。",
	},
	"zh_TW": {
		Name:        "DNS-01 驗證",
		Description: "支援 200 多家 DNS 服務商，以 DNS 記錄為憑證驗證網域。",
	},
	"ja_JP": {
		Name:        "DNS-01 チャレンジ",
		Description: "200 以上の DNS プロバイダーに対応し、DNS レコードで証明書のドメインを検証します。",
	},
}

// platforms are the targets build.sh cross compiles, in manifest order.
var platforms = []struct{ OS, Arch string }{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
	{"windows", "amd64"},
	{"windows", "arm64"},
}

// ExecutablePath returns the packaged path of one platform's binary.
func ExecutablePath(goos, goarch string) string {
	name := fmt.Sprintf("dns01-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return "server/dist/" + name
}

// Build assembles the manifest. It is deterministic: the provider list comes
// from the catalog sorted by name, and every map is encoded in key order.
func Build() (*protocol.Manifest, error) {
	providers, err := dns01Providers()
	if err != nil {
		return nil, err
	}

	executables := make(map[string]string, len(platforms))
	for _, p := range platforms {
		executables[p.OS+"-"+p.Arch] = ExecutablePath(p.OS, p.Arch)
	}

	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	shared, err := webappSharedRanges(root)
	if err != nil {
		return nil, err
	}

	return &protocol.Manifest{
		ID:                PluginID,
		Name:              PluginName,
		Version:           PluginVersion,
		Description:       PluginDescription,
		APIVersion:        protocol.APIVersion,
		MinNginxUIVersion: MinNginxUIVersion,
		I18n:              translations,
		Server: &protocol.ManifestServer{
			Executables:        executables,
			Lifecycle:          protocol.LifecycleOnDemand,
			IdleTimeoutSeconds: IdleTimeout,
		},
		IconPath: "webapp/dist/icon.svg",
		Webapp: &protocol.ManifestWebapp{
			BundlePath: "webapp/dist/main.js",
			StylePath:  "webapp/dist/style.css",
			Shared:     shared,
		},
		Capabilities: []string{protocol.CapabilityDNS01},
		Permissions:  []string{protocol.PermissionNetwork},
		DNS01:        &protocol.ManifestDNS01{Providers: providers},
		SettingsSchema: &protocol.SettingsSchema{
			Settings: []protocol.SettingsField{
				{
					Key:         "recursive_nameservers",
					Type:        "list",
					DisplayName: "Recursive DNS servers",
					HelpText:    "Used to check that DNS records are visible. Empty means the system resolvers.",
					Default:     []string{},
				},
				{
					Key:         "default_propagation_timeout_seconds",
					Type:        "number",
					DisplayName: "Default wait time (seconds)",
					HelpText:    "How long to wait at most for a record to become visible when the provider does not say.",
					Default:     120,
				},
			},
		},
	}, nil
}

// webappManifestFragment is the shape @nginxui/plugin-sdk/vite writes to
// webapp/dist/manifest.webapp.json. Only Shared is consumed here: bundle_path
// and style_path are fixed by the layout build.sh packages.
type webappManifestFragment struct {
	Shared map[string]string `json:"shared"`
}

// webappSharedRanges reads the semver ranges the webapp bundle was built
// against, if the bundle has been built. A missing file is not an error: the
// Go tests and cross compilation do not depend on the JS toolchain having run.
func webappSharedRanges(root string) (map[string]string, error) {
	path := filepath.Join(root, "webapp", "dist", "manifest.webapp.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var fragment webappManifestFragment
	if err := json.Unmarshal(data, &fragment); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return fragment.Shared, nil
}

// dns01Providers converts the catalog into manifest entries.
func dns01Providers() ([]protocol.DNS01Provider, error) {
	list, err := catalog.List()
	if err != nil {
		return nil, err
	}

	out := make([]protocol.DNS01Provider, 0, len(list))
	for _, c := range list {
		entry := protocol.DNS01Provider{Name: c.Name, Code: c.Code}

		if c.Configuration != nil {
			entry.Configuration = &protocol.DNS01ProviderConfig{
				Credentials: c.Configuration.Credentials,
				Additional:  c.Configuration.Additional,
			}
		}
		if c.Links != nil {
			entry.Links = &protocol.DNS01ProviderLinks{API: c.Links.API, GoClient: c.Links.GoClient}
		}

		out = append(out, entry)
	}
	return out, nil
}

// Render encodes the manifest the way it is committed: two space indentation
// and a trailing newline.
func Render() ([]byte, error) {
	manifest, err := Build()
	if err != nil {
		return nil, err
	}
	return encode(manifest)
}

// encode writes a manifest in the committed layout.
func encode(manifest *protocol.Manifest) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(manifest); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// FilterPlatform narrows a rendered manifest to one "<goos>-<goarch>"
// executable. A per-platform package must declare exactly the platform it
// ships, see the plugin spec PKG-12, while the catalog release keeps the full
// map in its manifest snapshot.
func FilterPlatform(data []byte, platform string) ([]byte, error) {
	var manifest protocol.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	if manifest.Server == nil {
		return nil, fmt.Errorf("the manifest has no server block")
	}
	executable, ok := manifest.Server.Executables[platform]
	if !ok {
		return nil, fmt.Errorf("the manifest declares no executable for %s", platform)
	}
	manifest.Server.Executables = map[string]string{platform: executable}
	return encode(&manifest)
}

// repoRoot resolves the module root from this file's location.
func repoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("manifest: unable to locate the generator source")
	}
	return filepath.Abs(filepath.Join(filepath.Dir(file), "..", ".."))
}

func main() {
	platform := flag.String("platform", "", `write the manifest of the "<goos>-<goarch>" package instead of regenerating plugin.json`)
	in := flag.String("in", "", "manifest to narrow with -platform (default: the committed plugin.json)")
	out := flag.String("out", "", "output file (default: the committed plugin.json, required with -platform)")
	flag.Parse()

	if err := run(*platform, *in, *out); err != nil {
		fmt.Fprintln(os.Stderr, "manifest:", err)
		os.Exit(1)
	}
}

func run(platform, in, out string) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}

	var data []byte
	switch {
	case platform == "":
		if data, err = Render(); err != nil {
			return err
		}
		if out == "" {
			out = filepath.Join(root, "plugin.json")
		}
	case out == "":
		return fmt.Errorf("-platform needs -out, the committed plugin.json keeps every platform")
	default:
		if in == "" {
			in = filepath.Join(root, "plugin.json")
		}
		source, err := os.ReadFile(in)
		if err != nil {
			return err
		}
		if data, err = FilterPlatform(source, platform); err != nil {
			return err
		}
	}

	if err = os.WriteFile(out, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d bytes)\n", out, len(data))
	return nil
}
