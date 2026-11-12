package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	sdk "github.com/nginxui/plugin-sdk-go"
	"github.com/nginxui/plugin-sdk-go/protocol"
)

// fakeISPConfig answers the ISPConfig remote JSON API calls lego makes and
// records which record IDs were deleted.
type fakeISPConfig struct {
	mu      sync.Mutex
	added   int
	deleted []string
}

func (f *fakeISPConfig) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/json")

	f.mu.Lock()
	defer f.mu.Unlock()

	switch r.URL.RawQuery {
	case "login":
		_, _ = io.WriteString(w, `{"code":"ok","message":"","response":"session"}`)
	case "dns_zone_get_id":
		_, _ = io.WriteString(w, `{"code":"ok","message":"","response":123}`)
	case "dns_zone_get":
		_, _ = io.WriteString(w, `{"code":"ok","message":"","response":{"id":"456","sys_userid":"789"}}`)
	case "client_get_id":
		_, _ = io.WriteString(w, `{"code":"ok","message":"","response":321}`)
	case "dns_txt_add":
		f.added++
		_, _ = io.WriteString(w, `{"code":"ok","message":"","response":"record-1"}`)
	case "dns_txt_delete":
		id := "unknown"
		if strings.Contains(string(body), `"record-1"`) {
			id = "record-1"
		}
		f.deleted = append(f.deleted, id)
		_, _ = io.WriteString(w, `{"code":"ok","message":"","response":1}`)
	default:
		http.Error(w, "unexpected call "+r.URL.RawQuery, http.StatusBadRequest)
	}
}

func ispconfigRequest(serverURL string) sdk.DNS01Request {
	return sdk.DNS01Request{
		Provider: "ispconfig",
		Config: map[string]string{
			"ISPCONFIG_SERVER_URL": serverURL,
			"ISPCONFIG_USERNAME":   "user",
			"ISPCONFIG_PASSWORD":   "secret",
		},
		// Keeps lego from resolving the challenge record's CNAME over the
		// network.
		Options: map[string]any{protocol.DNS01OptionDisableCNAME: true},
		Domain:  "example.com",
		Token:   "token",
		KeyAuth: "token.key",
	}
}

// lego providers such as cloudflare and ispconfig keep the vendor's record ID
// in memory during Present. CleanUp must reach that same provider, or the
// record cannot be found and stays behind.
func TestCleanUpRemovesTheRecordPresentPublished(t *testing.T) {
	vendor := &fakeISPConfig{}
	server := httptest.NewServer(vendor)
	defer server.Close()

	h := New()
	req := ispconfigRequest(server.URL)

	if err := h.Present(context.Background(), req); err != nil {
		t.Fatalf("Present: %v", err)
	}
	if err := h.CleanUp(context.Background(), req); err != nil {
		t.Fatalf("CleanUp: %v", err)
	}

	vendor.mu.Lock()
	defer vendor.mu.Unlock()
	if vendor.added != 1 || len(vendor.deleted) != 1 || vendor.deleted[0] != "record-1" {
		t.Fatalf("added %d, deleted %v, want the one published record deleted", vendor.added, vendor.deleted)
	}
}

// Every record is cleaned up through its own provider: two challenges of one
// certificate, presented before either is cleaned up, both get removed.
func TestCleanUpHandlesInterleavedChallenges(t *testing.T) {
	vendor := &fakeISPConfig{}
	server := httptest.NewServer(vendor)
	defer server.Close()

	h := New()
	wildcard := ispconfigRequest(server.URL)
	apex := ispconfigRequest(server.URL)
	apex.Token, apex.KeyAuth = "token-2", "token-2.key"

	for _, req := range []sdk.DNS01Request{wildcard, apex} {
		if err := h.Present(context.Background(), req); err != nil {
			t.Fatalf("Present %s: %v", req.Token, err)
		}
	}
	for _, req := range []sdk.DNS01Request{wildcard, apex} {
		if err := h.CleanUp(context.Background(), req); err != nil {
			t.Fatalf("CleanUp %s: %v", req.Token, err)
		}
	}

	if got := len(h.presented); got != 0 {
		t.Fatalf("%d providers kept after every cleanup, want 0", got)
	}
}

// A cleanup with other credentials, or one that arrives after the plugin lost
// the provider, falls back to a fresh provider exactly as before and reports
// the provider's own error.
func TestCleanUpWithoutAPresentedProviderFallsBack(t *testing.T) {
	vendor := &fakeISPConfig{}
	server := httptest.NewServer(vendor)
	defer server.Close()

	h := New()
	req := ispconfigRequest(server.URL)
	if err := h.Present(context.Background(), req); err != nil {
		t.Fatalf("Present: %v", err)
	}

	other := ispconfigRequest(server.URL)
	other.Config["ISPCONFIG_PASSWORD"] = "another"
	err := h.CleanUp(context.Background(), other)
	if err == nil || !strings.Contains(err.Error(), "unknown record ID") {
		t.Fatalf("CleanUp with other credentials = %v, want the fresh provider's unknown record ID error", err)
	}

	// The provider built from the right credentials is still there.
	if err := h.CleanUp(context.Background(), req); err != nil {
		t.Fatalf("CleanUp: %v", err)
	}
}

// A dry run never touches the vendor and never keeps a provider.
func TestDryRunKeepsNoProvider(t *testing.T) {
	vendor := &fakeISPConfig{}
	server := httptest.NewServer(vendor)
	defer server.Close()

	h := New()
	req := ispconfigRequest(server.URL)
	req.DryRun = true
	if err := h.Present(context.Background(), req); err != nil {
		t.Fatalf("Present dry run: %v", err)
	}
	if len(h.presented) != 0 || vendor.added != 0 {
		t.Fatalf("dry run kept %d providers and added %d records, want none", len(h.presented), vendor.added)
	}
}

// An entry whose cleanup never came expires instead of leaking.
func TestPresentedProvidersExpire(t *testing.T) {
	h := New()
	key := newPresentedKey(ispconfigRequest("http://127.0.0.1:1"))
	h.keepPresented(key, nil)

	entry := h.presented[key]
	entry.expires = entry.expires.Add(-2 * presentedTTL)
	h.presented[key] = entry

	if p := h.takePresented(key); p != nil {
		t.Fatal("an expired provider was handed out")
	}
	if len(h.presented) != 0 {
		t.Fatal("the expired entry was not forgotten")
	}
}
