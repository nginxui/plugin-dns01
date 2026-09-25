package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nginxui/plugin-sdk-go/protocol"

	"github.com/nginxui/plugin-dns01/catalog"
)

func TestRenderIsStable(t *testing.T) {
	first, err := Render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	second, err := Render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Fatal("Render() is not deterministic")
	}
	if !bytes.HasSuffix(first, []byte("\n")) {
		t.Fatal("the rendered manifest has no trailing newline")
	}
}

func TestCommittedManifestIsUpToDate(t *testing.T) {
	want, err := Render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	root, err := repoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "plugin.json"))
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}

	if !bytes.Equal(want, got) {
		t.Fatal("plugin.json is stale, run: go run ./cmd/manifest")
	}
}

func TestManifestShape(t *testing.T) {
	data, err := Render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	var m protocol.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if m.ID != PluginID || m.Version != PluginVersion {
		t.Fatalf("id = %q, version = %q", m.ID, m.Version)
	}
	if m.APIVersion != protocol.APIVersion {
		t.Fatalf("api_version = %d", m.APIVersion)
	}
	if m.MinNginxUIVersion != MinNginxUIVersion {
		t.Fatalf("min_nginx_ui_version = %q", m.MinNginxUIVersion)
	}
	for _, locale := range []string{"zh_CN", "zh_TW", "ja_JP"} {
		if tr := m.I18n[locale]; tr.Name == "" || tr.Description == "" {
			t.Fatalf("i18n[%s] = %+v", locale, tr)
		}
	}
	if m.Server == nil || m.Server.Lifecycle != protocol.LifecycleOnDemand || m.Server.IdleTimeoutSeconds != IdleTimeout {
		t.Fatalf("server = %+v", m.Server)
	}
	if len(m.Server.Executables) != len(platforms) {
		t.Fatalf("executables = %v", m.Server.Executables)
	}
	for _, p := range platforms {
		key := p.OS + "-" + p.Arch
		if m.Server.Executables[key] != ExecutablePath(p.OS, p.Arch) {
			t.Fatalf("executable %s = %q", key, m.Server.Executables[key])
		}
	}
	if m.Webapp == nil || m.Webapp.BundlePath != "webapp/dist/main.js" || m.Webapp.StylePath != "webapp/dist/style.css" {
		t.Fatalf("webapp = %+v", m.Webapp)
	}
	if len(m.Capabilities) != 1 || m.Capabilities[0] != protocol.CapabilityDNS01 {
		t.Fatalf("capabilities = %v", m.Capabilities)
	}
	if len(m.Permissions) != 1 || m.Permissions[0] != protocol.PermissionNetwork {
		t.Fatalf("permissions = %v", m.Permissions)
	}
	if m.SettingsSchema == nil || len(m.SettingsSchema.Settings) != 2 {
		t.Fatalf("settings schema = %+v", m.SettingsSchema)
	}

	list, err := catalog.List()
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if m.DNS01 == nil || len(m.DNS01.Providers) != len(list) {
		t.Fatalf("providers = %d, want %d", len(m.DNS01.Providers), len(list))
	}
	for i, p := range m.DNS01.Providers {
		if p.Name != list[i].Name || p.Code != list[i].Code {
			t.Fatalf("provider %d = %s/%s, want %s/%s", i, p.Name, p.Code, list[i].Name, list[i].Code)
		}
	}
}

func TestFilterPlatformKeepsOnlyOneExecutable(t *testing.T) {
	full, err := Render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	for _, p := range platforms {
		key := p.OS + "-" + p.Arch
		narrowed, err := FilterPlatform(full, key)
		if err != nil {
			t.Fatalf("filter %s: %v", key, err)
		}

		var m protocol.Manifest
		if err := json.Unmarshal(narrowed, &m); err != nil {
			t.Fatalf("unmarshal %s: %v", key, err)
		}
		if len(m.Server.Executables) != 1 || m.Server.Executables[key] != ExecutablePath(p.OS, p.Arch) {
			t.Fatalf("executables for %s = %v", key, m.Server.Executables)
		}

		// Everything but the executables map survives byte for byte.
		restored, err := restoreExecutables(narrowed)
		if err != nil {
			t.Fatalf("restore %s: %v", key, err)
		}
		if !bytes.Equal(restored, full) {
			t.Fatalf("the %s manifest differs from plugin.json beyond server.executables", key)
		}
	}

	if _, err := FilterPlatform(full, "plan9-386"); err == nil {
		t.Fatal("an undeclared platform must be refused")
	}
}

// restoreExecutables puts the full executables map back into a narrowed
// manifest, so the rest of the document can be compared.
func restoreExecutables(data []byte) ([]byte, error) {
	var m protocol.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	m.Server.Executables = make(map[string]string, len(platforms))
	for _, p := range platforms {
		m.Server.Executables[p.OS+"-"+p.Arch] = ExecutablePath(p.OS, p.Arch)
	}
	return encode(&m)
}
