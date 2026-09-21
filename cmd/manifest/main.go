// Command manifest regenerates plugin.json from the embedded provider catalog.
//
//	go run ./cmd/manifest
//
// The generated file is committed, so the plugin can be packaged without
// running the generator.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/0xJacky/nginx-ui-plugin-sdk-go/protocol"

	"github.com/0xJacky/nginx-ui-plugin-dns01/catalog"
)

// Plugin identity. Version is the single source of truth for the release
// artifacts; build.sh reads it back from the generated plugin.json.
const (
	PluginID          = "com.nginxui.dns01"
	PluginName        = "DNS-01 Challenge"
	PluginVersion     = "1.0.0"
	PluginDescription = "Solve the ACME DNS-01 challenge with any of the DNS providers supported by lego."
	MinNginxUIVersion = "2.7.0"
	IdleTimeout       = 300
)

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
		Server: &protocol.ManifestServer{
			Executables:        executables,
			Lifecycle:          protocol.LifecycleOnDemand,
			IdleTimeoutSeconds: IdleTimeout,
		},
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
					Type:        "text",
					DisplayName: "Recursive nameservers",
					HelpText:    "Comma separated host:port list used for the DNS propagation check. Empty means the system resolvers.",
				},
				{
					Key:         "default_propagation_timeout_seconds",
					Type:        "number",
					DisplayName: "Default propagation timeout (seconds)",
					HelpText:    "Applied to providers that do not report a propagation timeout of their own.",
					Default:     120,
				},
			},
		},
	}, nil
}

// webappManifestFragment is the shape @nginx-ui/plugin-sdk/vite writes to
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

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(manifest); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
	data, err := Render()
	if err != nil {
		fmt.Fprintln(os.Stderr, "manifest:", err)
		os.Exit(1)
	}

	root, err := repoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "manifest:", err)
		os.Exit(1)
	}

	out := filepath.Join(root, "plugin.json")
	if err := os.WriteFile(out, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "manifest:", err)
		os.Exit(1)
	}

	fmt.Printf("wrote %s (%d bytes)\n", out, len(data))
}
