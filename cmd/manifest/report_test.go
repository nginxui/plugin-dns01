package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormTextsAreClean(t *testing.T) {
	manifest, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range Issues(manifest.DNS01.Providers) {
		t.Errorf("%s %s: %s: %s", issue.Provider, issue.Key, issue.Problem, issue.Text)
	}
}

func TestFormPhrasesAreTranslated(t *testing.T) {
	manifest, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	translations, err := LoadTranslations(root)
	if err != nil {
		t.Fatal(err)
	}

	providers := manifest.DNS01.Providers
	phrases := Phrases(providers)
	for _, locale := range Locales {
		for _, kind := range []PhraseKind{PhraseLabel, PhraseMethod, PhraseHelp} {
			for _, missing := range Missing(phrases[kind], translations[locale]) {
				t.Errorf("%s has no translation for %q", locale, missing)
			}
		}
		for _, unused := range Unused(providers, phrases, translations[locale]) {
			t.Errorf("%s translates %q, which the manifest does not use", locale, unused)
		}
	}
}

// TestWebappRegistersProviderPhrases keeps the webapp importing every
// locale file the generator checks.
func TestWebappRegistersProviderPhrases(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, locale := range Locales {
		source, err := os.ReadFile(filepath.Join(root, "webapp", "src", "locales", locale+".ts"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(source, []byte("catalog/i18n/"+locale+".json")) {
			t.Errorf("webapp/src/locales/%s.ts does not import catalog/i18n/%s.json", locale, locale)
		}
	}
}

func TestReport(t *testing.T) {
	var buf bytes.Buffer
	if err := report(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Field issues: 0", "zh_CN", "labels:", "method names:", "help texts:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("report lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "missing:") || strings.Contains(out, "unused:") {
		t.Fatalf("report lists gaps:\n%s", out)
	}
}
