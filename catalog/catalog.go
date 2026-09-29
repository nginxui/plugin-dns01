// Package catalog exposes the DNS provider descriptions lego ships as TOML
// files. The files live in catalog/data and are embedded at build time, so the
// plugin needs no network access to list the providers it supports.
//
// The parsing mirrors internal/cert/dns/config_env.go of nginx-ui so that the
// host and the plugin describe a provider the same way.
package catalog

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

//go:embed data/*.toml
var data embed.FS

// Configuration lists the environment variables a provider reads.
type Configuration struct {
	// Credentials are the secrets the user must supply.
	Credentials map[string]string `json:"credentials,omitempty" toml:"Credentials"`
	// Additional are optional tuning variables such as timeouts.
	Additional map[string]string `json:"additional,omitempty" toml:"Additional"`
}

// Links points at the vendor documentation.
type Links struct {
	API string `json:"api,omitempty" toml:"API"`
}

// Config describes one DNS provider.
type Config struct {
	Name          string         `json:"name" toml:"Name"`
	Code          string         `json:"code" toml:"Code"`
	Configuration *Configuration `json:"configuration,omitempty" toml:"Configuration"`
	Links         *Links         `json:"links,omitempty" toml:"Links"`
	// Example is the upstream usage sample, read to find the ways to sign in.
	Example string `json:"-" toml:"Example"`
	// Doc is the upstream markdown documentation. A provider without a
	// Configuration table lists its variables there.
	Doc string `json:"-" toml:"Additional"`

	// credentialOrder and additionalOrder keep the keys in file order.
	credentialOrder []string
	additionalOrder []string
}

var (
	once      sync.Once
	loadErr   error
	ordered   []Config
	byCode    map[string]Config
	loadedAll []string
	// everyConfig also holds the providers overrides.json hides.
	everyConfig []Config
)

func load() {
	byCode = make(map[string]Config)

	entries, err := fs.ReadDir(data, "data")
	if err != nil {
		loadErr = fmt.Errorf("catalog: read data dir: %w", err)
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".toml") {
			continue
		}

		raw, err := data.ReadFile(path.Join("data", name))
		if err != nil {
			loadErr = fmt.Errorf("catalog: read %s: %w", name, err)
			return
		}

		var c Config
		meta, err := toml.Decode(string(raw), &c)
		if err != nil {
			loadErr = fmt.Errorf("catalog: parse %s: %w", name, err)
			return
		}
		c.credentialOrder, c.additionalOrder = keyOrder(meta)
		if c.Configuration == nil {
			c.Configuration, c.credentialOrder, c.additionalOrder = docTables(c.Doc)
		}
		if c.Code == "" {
			loadErr = fmt.Errorf("catalog: %s has no Code", name)
			return
		}
		loadedAll = append(loadedAll, name)
		everyConfig = append(everyConfig, c)
		if hiddenCode(c.Code) {
			continue
		}

		byCode[c.Code] = c
		ordered = append(ordered, c)
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		left := strings.ToLower(ordered[i].Name)
		right := strings.ToLower(ordered[j].Name)
		if left == right {
			return strings.ToLower(ordered[i].Code) < strings.ToLower(ordered[j].Code)
		}
		return left < right
	})
}

// List returns every known provider, sorted by name then code. The returned
// slice is a copy and safe to modify.
func List() ([]Config, error) {
	once.Do(load)
	if loadErr != nil {
		return nil, loadErr
	}
	return append([]Config(nil), ordered...), nil
}

// Get returns the provider with the given lego code.
func Get(code string) (Config, bool) {
	once.Do(load)
	if loadErr != nil {
		return Config{}, false
	}
	c, ok := byCode[code]
	return c, ok
}

// Files returns the names of the embedded TOML files, mainly for diagnostics.
func Files() ([]string, error) {
	once.Do(load)
	if loadErr != nil {
		return nil, loadErr
	}
	return append([]string(nil), loadedAll...), nil
}

// CredentialKeys returns the credential variable names of a provider in a
// stable order.
func (c Config) CredentialKeys() []string {
	if c.Configuration == nil {
		return nil
	}
	return sortedKeys(c.Configuration.Credentials)
}

// AdditionalKeys returns the optional variable names of a provider in a stable
// order.
func (c Config) AdditionalKeys() []string {
	if c.Configuration == nil {
		return nil
	}
	return sortedKeys(c.Configuration.Additional)
}

func sortedKeys(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// keyOrder returns the credential and additional keys in file order.
func keyOrder(meta toml.MetaData) (credentials, additional []string) {
	for _, key := range meta.Keys() {
		if len(key) != 3 || key[0] != "Configuration" {
			continue
		}
		switch key[1] {
		case "Credentials":
			credentials = append(credentials, key[2])
		case "Additional":
			additional = append(additional, key[2])
		}
	}
	return credentials, additional
}

// reDocRow matches a "| `KEY` | description |" row of a markdown table.
var reDocRow = regexp.MustCompile("^\\|\\s*`([A-Z][A-Z0-9_]*)`\\s*\\|\\s*(.*?)\\s*\\|\\s*$")

// docTables reads the variables of the "Base Configuration" and
// "Additional Configuration" tables of the markdown documentation, for the
// few providers that have no Configuration table.
func docTables(doc string) (*Configuration, []string, []string) {
	cfg := &Configuration{Credentials: map[string]string{}, Additional: map[string]string{}}
	var credentials, additional []string
	var target map[string]string
	var order *[]string
	for _, line := range strings.Split(doc, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			switch strings.ToLower(strings.TrimSpace(strings.TrimLeft(trimmed, "#"))) {
			case "base configuration":
				target, order = cfg.Credentials, &credentials
			case "additional configuration":
				target, order = cfg.Additional, &additional
			default:
				target, order = nil, nil
			}
			continue
		}
		if target == nil {
			continue
		}
		if m := reDocRow.FindStringSubmatch(trimmed); m != nil {
			target[m[1]] = m[2]
			*order = append(*order, m[1])
		}
	}
	if len(credentials) == 0 && len(additional) == 0 {
		return nil, nil, nil
	}
	return cfg, credentials, additional
}
