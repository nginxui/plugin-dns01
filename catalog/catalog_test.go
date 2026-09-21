package catalog_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/0xJacky/nginx-ui-plugin-dns01/catalog"
)

// wantProviders is the number of TOML files lego ships for the pinned version.
const wantProviders = 224

func TestListParsesEveryEmbeddedFile(t *testing.T) {
	files, err := catalog.Files()
	if err != nil {
		t.Fatalf("files: %v", err)
	}
	if len(files) != wantProviders {
		t.Fatalf("embedded files = %d, want %d", len(files), wantProviders)
	}

	list, err := catalog.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != wantProviders {
		t.Fatalf("providers = %d, want %d", len(list), wantProviders)
	}

	for _, c := range list {
		if c.Name == "" {
			t.Fatalf("provider %q has no name", c.Code)
		}
		if c.Code == "" {
			t.Fatalf("provider %q has no code", c.Name)
		}
	}
}

func TestListIsSortedByName(t *testing.T) {
	list, err := catalog.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	sorted := slices.IsSortedFunc(list, func(a, b catalog.Config) int {
		left, right := strings.ToLower(a.Name), strings.ToLower(b.Name)
		if left == right {
			return strings.Compare(strings.ToLower(a.Code), strings.ToLower(b.Code))
		}
		return strings.Compare(left, right)
	})
	if !sorted {
		t.Fatal("List() is not sorted by name then code")
	}
}

func TestGetCloudflare(t *testing.T) {
	c, ok := catalog.Get("cloudflare")
	if !ok {
		t.Fatal("cloudflare is missing from the catalog")
	}
	if c.Name != "Cloudflare" {
		t.Fatalf("name = %q, want Cloudflare", c.Name)
	}
	if c.Configuration == nil {
		t.Fatal("cloudflare has no configuration block")
	}
	if _, ok := c.Configuration.Credentials["CF_DNS_API_TOKEN"]; !ok {
		t.Fatalf("credentials = %v, want CF_DNS_API_TOKEN", c.Configuration.Credentials)
	}
	if !slices.Contains(c.CredentialKeys(), "CF_DNS_API_TOKEN") {
		t.Fatalf("credential keys = %v", c.CredentialKeys())
	}
	if c.Links == nil || c.Links.API == "" {
		t.Fatalf("links = %+v, want an API link", c.Links)
	}
}

func TestGetUnknownProvider(t *testing.T) {
	if _, ok := catalog.Get("definitely-not-a-provider"); ok {
		t.Fatal("Get returned a provider for an unknown code")
	}
}

func TestListReturnsACopy(t *testing.T) {
	first, err := catalog.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	first[0].Name = "mutated"

	second, err := catalog.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if second[0].Name == "mutated" {
		t.Fatal("List() exposed its backing array")
	}
}
