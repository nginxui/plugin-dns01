package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xJacky/nginx-ui-plugin-sdk-go/protocol"

	"github.com/0xJacky/nginx-ui-plugin-dns01/catalog"
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
