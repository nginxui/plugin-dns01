// Package provider bridges the nginx-ui dns01 capability onto lego's DNS
// providers. lego configures a provider from process environment variables, so
// every call here takes a package wide lock, exports the user's configuration,
// runs the provider and restores the previous environment.
package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	sdk "github.com/0xJacky/nginx-ui-plugin-sdk-go"
	"github.com/0xJacky/nginx-ui-plugin-sdk-go/protocol"
	"github.com/go-acme/lego/v5/challenge"
	"github.com/go-acme/lego/v5/challenge/dns01"
	dnsproviders "github.com/go-acme/lego/v5/providers/dns"

	"github.com/0xJacky/nginx-ui-plugin-dns01/catalog"
	"github.com/0xJacky/nginx-ui-plugin-dns01/check"
)

// envDisableCNAME makes lego stop following CNAMEs when deriving the
// challenge record.
const envDisableCNAME = "LEGO_DISABLE_CNAME_SUPPORT"

// envMu serializes every call that touches the process environment.
var envMu sync.Mutex

// sequentialProvider is lego's unexported interface for providers that must
// solve one challenge at a time.
type sequentialProvider interface {
	Sequential() time.Duration
}

// Settings are the plugin settings the host pushes on configure.
type Settings struct {
	// RecursiveNameservers are queried during the propagation check. Empty
	// means the system resolvers.
	RecursiveNameservers []string
	// DefaultPropagationTimeout applies to providers that do not report a
	// timeout of their own. Zero means lego's default.
	DefaultPropagationTimeout time.Duration
}

// Handler implements the dns01 capability on top of lego.
type Handler struct {
	mu         sync.RWMutex
	settings   Settings
	configured bool
}

// New builds a handler with lego's defaults.
func New() *Handler { return &Handler{} }

// SetSettings replaces the settings the handler uses for later calls.
func (h *Handler) SetSettings(s Settings) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.settings = s
	h.configured = true
}

// Settings returns the settings set through SetSettings.
func (h *Handler) Settings() Settings {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.settings
}

// settingsFor returns the settings to use for one call. It prefers what
// plugin.configure pushed and falls back to the handshake settings the host
// client carries, so a plugin that was only initialized still honours them.
func (h *Handler) settingsFor(ctx context.Context) Settings {
	h.mu.RLock()
	configured, s := h.configured, h.settings
	h.mu.RUnlock()

	if configured {
		return s
	}
	if host := sdk.HostFromContext(ctx); host != nil {
		return ParseSettings(host.Settings())
	}
	return s
}

// ParseSettings maps the raw settings map onto the typed struct.
func ParseSettings(raw map[string]any) Settings {
	var s Settings

	if v, ok := raw["recursive_nameservers"]; ok {
		s.RecursiveNameservers = splitNameservers(fmt.Sprint(v))
	}
	if v, ok := raw["default_propagation_timeout_seconds"]; ok {
		if seconds := toSeconds(v); seconds > 0 {
			s.DefaultPropagationTimeout = time.Duration(seconds) * time.Second
		}
	}
	return s
}

func splitNameservers(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func toSeconds(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(n))
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

// Present publishes the challenge record through the vendor API.
func (h *Handler) Present(ctx context.Context, req sdk.DNS01Request) error {
	return h.solve(ctx, req, true)
}

// CleanUp removes the record Present published.
func (h *Handler) CleanUp(ctx context.Context, req sdk.DNS01Request) error {
	return h.solve(ctx, req, false)
}

func (h *Handler) solve(ctx context.Context, req sdk.DNS01Request, present bool) error {
	cfg, ok := catalog.Get(req.Provider)
	if !ok {
		return sdk.InvalidConfig("provider", "unknown DNS provider: "+req.Provider)
	}

	envMu.Lock()
	defer envMu.Unlock()

	scope := newEnvScope(cfg)
	defer scope.CleanEnv()

	if err := scope.SetEnv(req.Config); err != nil {
		return sdk.Internal("could not export the provider configuration: " + err.Error())
	}
	if err := applyCNAMEOption(scope, req.Options); err != nil {
		return sdk.Internal(err.Error())
	}

	p, err := dnsproviders.NewDNSChallengeProviderByName(req.Provider)
	if err != nil {
		return configError(cfg, req.Config, err)
	}

	if req.DryRun {
		// The shapes are valid and the provider could be built; stop before
		// touching the vendor API.
		sdk.Infof("dns01: dry run for %s on %s", req.Provider, req.Domain)
		return nil
	}

	if present {
		sdk.Infof("dns01: presenting %s via %s", req.EffectiveFQDN, req.Provider)
		if err := p.Present(ctx, req.Domain, req.Token, req.KeyAuth); err != nil {
			return sdk.Internal(err.Error())
		}
		return nil
	}

	sdk.Infof("dns01: cleaning up %s via %s", req.EffectiveFQDN, req.Provider)
	if err := p.CleanUp(ctx, req.Domain, req.Token, req.KeyAuth); err != nil {
		return sdk.Internal(err.Error())
	}
	return nil
}

// Options reports the propagation timings lego would use for the provider.
func (h *Handler) Options(ctx context.Context, params protocol.DNS01OptionsParams) (protocol.DNS01OptionsResult, error) {
	cfg, ok := catalog.Get(params.Provider)
	if !ok {
		return protocol.DNS01OptionsResult{}, sdk.InvalidConfig("provider", "unknown DNS provider: "+params.Provider)
	}

	envMu.Lock()
	defer envMu.Unlock()

	scope := newEnvScope(cfg)
	defer scope.CleanEnv()

	if err := scope.SetEnv(params.Config); err != nil {
		return protocol.DNS01OptionsResult{}, sdk.Internal("could not export the provider configuration: " + err.Error())
	}

	p, err := dnsproviders.NewDNSChallengeProviderByName(params.Provider)
	if err != nil {
		return protocol.DNS01OptionsResult{}, configError(cfg, params.Config, err)
	}

	timeout, interval := h.defaultTimings(ctx)
	if pt, ok := p.(challenge.ProviderTimeout); ok {
		timeout, interval = pt.Timeout()
	}

	var sequential time.Duration
	if sp, ok := p.(sequentialProvider); ok {
		sequential = sp.Sequential()
	}

	return protocol.DNS01OptionsResult{
		PropagationTimeoutSeconds: int(timeout.Seconds()),
		PollingIntervalSeconds:    int(interval.Seconds()),
		SequentialIntervalSeconds: int(sequential.Seconds()),
	}, nil
}

// defaultTimings returns lego's defaults, with the propagation timeout
// overridden by the plugin setting when the user configured one.
func (h *Handler) defaultTimings(ctx context.Context) (timeout, interval time.Duration) {
	timeout, interval = dns01.DefaultPropagationTimeout, dns01.DefaultPollingInterval
	if d := h.settingsFor(ctx).DefaultPropagationTimeout; d > 0 {
		timeout = d
	}
	return timeout, interval
}

// Validate instantiates the provider to confirm the credentials are complete.
// It never contacts the vendor API.
func (h *Handler) Validate(_ context.Context, providerCode string, config map[string]string) error {
	cfg, ok := catalog.Get(providerCode)
	if !ok {
		return sdk.InvalidConfig("provider", "unknown DNS provider: "+providerCode)
	}

	envMu.Lock()
	defer envMu.Unlock()

	scope := newEnvScope(cfg)
	defer scope.CleanEnv()

	if err := scope.SetEnv(config); err != nil {
		return sdk.Internal("could not export the provider configuration: " + err.Error())
	}

	if _, err := dnsproviders.NewDNSChallengeProviderByName(providerCode); err != nil {
		return configError(cfg, config, err)
	}
	return nil
}

// Check runs the DNS propagation check for one challenge record.
func (h *Handler) Check(ctx context.Context, params protocol.DNS01CheckParams) (protocol.DNS01CheckResult, error) {
	settings := h.settingsFor(ctx)

	opts := check.Options{
		FQDN:                              params.FQDN,
		Value:                             params.Value,
		RecursiveNameservers:              settings.RecursiveNameservers,
		DisableCNAME:                      boolOption(params.Options, protocol.DNS01OptionDisableCNAME),
		DisableAuthoritativeNSPropagation: boolOption(params.Options, protocol.DNS01OptionDisableAuthoritativeNSPropagation),
		DisableRecursiveNSPropagation:     boolOption(params.Options, protocol.DNS01OptionDisableRecursiveNSPropagation),
	}

	return check.Propagated(ctx, opts)
}

// applyCNAMEOption exports LEGO_DISABLE_CNAME_SUPPORT for the duration of the
// call when the certificate asked for it.
func applyCNAMEOption(scope *envScope, options map[string]any) error {
	if !boolOption(options, protocol.DNS01OptionDisableCNAME) {
		return nil
	}
	if err := scope.SetRaw(envDisableCNAME, "true"); err != nil {
		return fmt.Errorf("could not set %s: %w", envDisableCNAME, err)
	}
	return nil
}

// boolOption reads a per-certificate option that may arrive as a bool or a string.
func boolOption(options map[string]any, key string) bool {
	v, ok := options[key]
	if !ok {
		return false
	}

	switch value := v.(type) {
	case bool:
		return value
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		return err == nil && parsed
	case float64:
		return value != 0
	default:
		return false
	}
}

// missingCredentialsPrefix is the message lego's env helper produces when a
// required variable is empty.
const missingCredentialsPrefix = "some credentials information are missing:"

// configError maps a provider constructor failure onto an invalid config error
// naming the offending field when it can be identified. The user's values are
// never included.
func configError(cfg catalog.Config, config map[string]string, err error) error {
	var rpcErr *protocol.Error
	if errors.As(err, &rpcErr) {
		return rpcErr
	}

	if field := missingField(cfg, config, err); field != "" {
		return sdk.InvalidConfig(field, err.Error())
	}
	return sdk.InvalidConfig("", err.Error())
}

func missingField(cfg catalog.Config, config map[string]string, err error) string {
	msg := err.Error()

	if idx := strings.Index(msg, missingCredentialsPrefix); idx >= 0 {
		rest := strings.TrimSpace(msg[idx+len(missingCredentialsPrefix):])
		// lego joins the names with a comma and may append its own context.
		rest = strings.SplitN(rest, ":", 2)[0]
		for _, name := range strings.Split(rest, ",") {
			if trimmed := strings.TrimSpace(name); trimmed != "" {
				return trimmed
			}
		}
	}

	// Fall back to the first declared credential the user left empty.
	for _, key := range cfg.CredentialKeys() {
		if strings.TrimSpace(config[key]) == "" {
			return key
		}
	}
	return ""
}
