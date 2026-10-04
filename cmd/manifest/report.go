package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/nginxui/plugin-sdk-go/protocol"

	"github.com/nginxui/plugin-dns01/catalog"
)

// Locales are the languages the provider phrases are translated into.
var Locales = []string{"zh_CN", "zh_TW", "ja_JP", "ko_KR", "de_DE", "fr_FR", "es", "it_IT", "pt_PT", "ru_RU", "uk_UA", "tr_TR", "vi_VN", "ar"}

// PhraseKind tells where a form phrase is used.
type PhraseKind int

const (
	PhraseLabel PhraseKind = iota
	PhraseMethod
	PhraseHelp
)

// Phrases collects the distinct form strings by kind, sorted.
func Phrases(providers []protocol.DNS01Provider) map[PhraseKind][]string {
	sets := map[PhraseKind]map[string]bool{PhraseLabel: {}, PhraseMethod: {}, PhraseHelp: {}}
	for _, p := range providers {
		for _, f := range p.Form.Fields {
			sets[PhraseLabel][f.Label] = true
			if f.Help != "" {
				sets[PhraseHelp][f.Help] = true
			}
		}
		for _, m := range p.Form.Methods {
			sets[PhraseMethod][m.Name] = true
		}
	}
	out := make(map[PhraseKind][]string, len(sets))
	for kind, set := range sets {
		list := make([]string, 0, len(set))
		for s := range set {
			list = append(list, s)
		}
		sort.Strings(list)
		out[kind] = list
	}
	return out
}

// LoadTranslations reads catalog/i18n/<locale>.json for every locale.
func LoadTranslations(root string) (map[string]map[string]string, error) {
	out := make(map[string]map[string]string, len(Locales))
	for _, locale := range Locales {
		path := filepath.Join(root, "catalog", "i18n", locale+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var messages map[string]string
		if err := json.Unmarshal(data, &messages); err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		out[locale] = messages
	}
	return out, nil
}

// Missing lists the phrases a locale has no translation for.
func Missing(phrases []string, messages map[string]string) []string {
	var out []string
	for _, p := range phrases {
		if strings.TrimSpace(messages[p]) == "" {
			out = append(out, p)
		}
	}
	return out
}

// Unused lists the translated strings that are neither a form phrase nor a
// provider name.
func Unused(providers []protocol.DNS01Provider, phrases map[PhraseKind][]string, messages map[string]string) []string {
	known := make(map[string]bool)
	for _, list := range phrases {
		for _, p := range list {
			known[p] = true
		}
	}
	for _, p := range providers {
		known[p.Name] = true
	}
	var out []string
	for msgid := range messages {
		if !known[msgid] {
			out = append(out, msgid)
		}
	}
	sort.Strings(out)
	return out
}

// reInternals matches text that names a mechanism instead of a meaning.
var reInternals = regexp.MustCompile("`|[A-Z0-9]{2,}_[A-Z0-9_]+|(?i)\\blego\\b|(?i)environment variable|://|[()]")

// Issue is one field text that needs a look.
type Issue struct {
	Provider string
	Key      string
	Problem  string
	Text     string
}

// Issues lists labels over the length limit and texts that still carry
// upstream details.
func Issues(providers []protocol.DNS01Provider) []Issue {
	var out []Issue
	for _, p := range providers {
		for _, f := range p.Form.Fields {
			if utf8.RuneCountInString(f.Label) > catalog.MaxLabelLength {
				out = append(out, Issue{p.Code, f.Key, "long label", f.Label})
			}
			if reInternals.MatchString(f.Label) {
				out = append(out, Issue{p.Code, f.Key, "uncleaned label", f.Label})
			}
			if f.Help != "" && reInternals.MatchString(stripExample(f.Help)) {
				out = append(out, Issue{p.Code, f.Key, "uncleaned help", f.Help})
			}
			if strings.ContainsAny(f.Default, " \t") {
				out = append(out, Issue{p.Code, f.Key, "odd default", f.Default})
			}
		}
		for _, m := range p.Form.Methods {
			if reInternals.MatchString(m.Name) || utf8.RuneCountInString(m.Name) > catalog.MaxLabelLength {
				out = append(out, Issue{p.Code, "", "uncleaned method", m.Name})
			}
		}
	}
	return out
}

// stripExample drops the value of a "For example:" help, which may hold a
// URL or key names on purpose.
func stripExample(help string) string {
	if strings.HasPrefix(help, "For example: ") {
		return ""
	}
	return help
}

// report writes the review list for the committed catalog.
func report(w io.Writer) error {
	manifest, err := Build()
	if err != nil {
		return err
	}
	root, err := repoRoot()
	if err != nil {
		return err
	}
	translations, err := LoadTranslations(root)
	if err != nil {
		return err
	}
	providers := manifest.DNS01.Providers

	issues := Issues(providers)
	fmt.Fprintf(w, "Field issues: %d\n", len(issues))
	for _, i := range issues {
		fmt.Fprintf(w, "  %s %s [%s] %s\n", i.Provider, i.Key, i.Problem, i.Text)
	}

	phrases := Phrases(providers)
	names := map[PhraseKind]string{PhraseLabel: "labels", PhraseMethod: "method names", PhraseHelp: "help texts"}
	for _, locale := range Locales {
		fmt.Fprintf(w, "\n%s\n", locale)
		for _, kind := range []PhraseKind{PhraseLabel, PhraseMethod, PhraseHelp} {
			missing := Missing(phrases[kind], translations[locale])
			total := len(phrases[kind])
			fmt.Fprintf(w, "  %s: %d/%d translated\n", names[kind], total-len(missing), total)
			for _, m := range missing {
				fmt.Fprintf(w, "    missing: %s\n", m)
			}
		}
		for _, u := range Unused(providers, phrases, translations[locale]) {
			fmt.Fprintf(w, "  unused: %s\n", u)
		}
	}
	return nil
}
