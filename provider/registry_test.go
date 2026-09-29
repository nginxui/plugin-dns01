package provider

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/nginxui/plugin-dns01/catalog"
)

// reRegistryCase matches a `case "a", "b":` line of lego's provider switch.
var reRegistryCase = regexp.MustCompile(`^\s*case\s+(.+):\s*$`)

// legoProviderNames reads the provider names lego's registry switch accepts
// from the source of the pinned module.
func legoProviderNames(t *testing.T) map[string]bool {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/go-acme/lego/v5").Output()
	if err != nil {
		t.Skipf("go list: %v", err)
	}
	source, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(out)), "providers", "dns", "zz_gen_dns_providers.go"))
	if err != nil {
		t.Fatal(err)
	}

	names := make(map[string]bool)
	for _, line := range strings.Split(string(source), "\n") {
		m := reRegistryCase.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		for _, name := range strings.Split(m[1], ",") {
			names[strings.Trim(strings.TrimSpace(name), `"`)] = true
		}
	}
	if len(names) < 100 {
		t.Fatalf("read only %d provider names from lego", len(names))
	}
	return names
}

// TestCatalogMatchesLegoRegistry keeps catalog/data and the lego module in
// step: every offered provider must be one lego can build.
func TestCatalogMatchesLegoRegistry(t *testing.T) {
	names := legoProviderNames(t)
	list, err := catalog.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range list {
		if !names[c.Code] {
			t.Errorf("catalog provider %s is not in lego's registry", c.Code)
		}
	}
}
