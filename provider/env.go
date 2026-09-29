package provider

import (
	"os"
	"strconv"
	"strings"

	"github.com/nginxui/plugin-dns01/catalog"
)

// envScope sets the environment variables a lego provider reads and restores
// the previous process environment afterwards. The environment is process
// global, so a scope is only safe while the package mutex is held.
//
// Ported from internal/cert/dns/config_env.go of nginx-ui.
type envScope struct {
	cfg    catalog.Config
	backup map[string]*string
}

func newEnvScope(cfg catalog.Config) *envScope {
	return &envScope{cfg: cfg, backup: make(map[string]*string)}
}

// SetEnv exports the values of the keys the provider form declares: its
// fields and the fixed values of its sign-in methods. Other keys are
// ignored. On error the scope is rolled back before returning.
func (s *envScope) SetEnv(config map[string]string) error {
	for _, k := range s.cfg.Keys() {
		if value, ok := config[k]; ok {
			if err := s.set(k, value); err != nil {
				s.CleanEnv()
				return err
			}
		}
	}
	return nil
}

// SetRaw exports one variable that is not part of the provider description,
// such as LEGO_DISABLE_CNAME_SUPPORT.
func (s *envScope) SetRaw(key, value string) error {
	s.backupEnv(key)
	return os.Setenv(key, value)
}

// CleanEnv restores every variable the scope touched.
func (s *envScope) CleanEnv() {
	for key := range s.backup {
		s.restore(key)
	}
	s.backup = make(map[string]*string)
}

func (s *envScope) set(key, value string) error {
	s.backupEnv(key)
	return os.Setenv(key, normalizeEnvValue(value))
}

func (s *envScope) backupEnv(key string) {
	if _, ok := s.backup[key]; ok {
		return
	}

	value, exists := os.LookupEnv(key)
	if !exists {
		s.backup[key] = nil
		return
	}

	copied := value
	s.backup[key] = &copied
}

func (s *envScope) restore(key string) {
	value, exists := s.backup[key]
	if !exists || value == nil {
		_ = os.Unsetenv(key)
		return
	}
	_ = os.Setenv(key, *value)
}

// normalizeEnvValue trims the value and strips one matching pair of surrounding
// quotes, so a pasted `"token"` behaves like a bare token.
func normalizeEnvValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 2 {
		return trimmed
	}

	first := trimmed[0]
	last := trimmed[len(trimmed)-1]
	if first != last || first != '"' && first != '\'' {
		return trimmed
	}
	if first == '\'' {
		return trimmed[1 : len(trimmed)-1]
	}

	unquoted, err := strconv.Unquote(trimmed)
	if err != nil {
		return trimmed[1 : len(trimmed)-1]
	}

	return unquoted
}
