package catalog

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxLabelLength is the length above which a label is split or reported.
const MaxLabelLength = 40

// cleaned is an upstream field description broken into its parts.
type cleaned struct {
	// Text is the description without the parts extracted below.
	Text     string
	Default  string
	Optional bool
	Seconds  bool
	Link     string
	Example  string
}

var (
	reAlias        = regexp.MustCompile("^Alias (?:to|on|of) `?([A-Z0-9_]+)`?\\.?$")
	reSince        = regexp.MustCompile(`(?i)\s*\(since v[0-9.]+\)`)
	reDefaultParen = regexp.MustCompile(`(?i)\s*\((?:by )?default:?\s*([^)]*)\)`)
	reDefaultsTo   = regexp.MustCompile(`(?i),\s*defaults to\s+(\S+)$`)
	reOptional     = regexp.MustCompile(`(?i)\s*\(optional\)`)
	reSeconds      = regexp.MustCompile(`(?i)\s+in seconds\b`)
	reMarkdownLink = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)\s]+)\)`)
	reParenLink    = regexp.MustCompile(`\s*\((https?://[^)\s]+)\)`)
	reTrailingLink = regexp.MustCompile(`\s+(https?://\S+)$`)
	reExampleParen = regexp.MustCompile(`(?i)\s*\((?:ex|e\.g\.)[.,]?:?\s*([^)]*)\)`)
	reExampleTail  = regexp.MustCompile(`\.\s+Ex:\s*(.+)$`)
	reSpaces       = regexp.MustCompile(`\s+`)
)

// aliasTarget returns the key an "Alias to X" description points at.
func aliasTarget(description string) (string, bool) {
	m := reAlias.FindStringSubmatch(strings.TrimSpace(description))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// clean extracts the default, unit, link, example and optional mark from an
// upstream description.
func clean(description string) cleaned {
	var c cleaned
	text := reSpaces.ReplaceAllString(strings.TrimSpace(description), " ")
	text = reSince.ReplaceAllString(text, "")

	if m := reDefaultParen.FindStringSubmatch(text); m != nil {
		// A default that reads like a sentence is left for an override.
		if value := unquote(m[1]); !strings.ContainsAny(value, " \t") {
			c.Default = value
			text = strings.Replace(text, m[0], "", 1)
		}
	} else if m := reDefaultsTo.FindStringSubmatch(text); m != nil {
		c.Default = unquote(m[1])
		text = strings.Replace(text, m[0], "", 1)
	}

	if reOptional.MatchString(text) {
		c.Optional = true
		text = reOptional.ReplaceAllString(text, "")
	}
	if reSeconds.MatchString(text) {
		c.Seconds = true
		text = reSeconds.ReplaceAllString(text, "")
	}

	if m := reExampleParen.FindStringSubmatch(text); m != nil {
		c.Example = strings.TrimSpace(m[1])
		text = strings.Replace(text, m[0], "", 1)
	} else if m := reExampleTail.FindStringSubmatch(text); m != nil {
		c.Example = strings.TrimSpace(m[1])
		text = strings.Replace(text, m[0], "", 1)
	}

	if m := reMarkdownLink.FindStringSubmatch(text); m != nil {
		c.Link = m[2]
		text = strings.Replace(text, m[0], m[1], 1)
	} else if m := reParenLink.FindStringSubmatch(text); m != nil {
		c.Link = m[1]
		text = strings.Replace(text, m[0], "", 1)
	} else if m := reTrailingLink.FindStringSubmatch(text); m != nil {
		c.Link = m[1]
		text = strings.Replace(text, m[0], "", 1)
	}

	c.Text = trimSentence(text)
	return c
}

// unquote trims spaces and one pair of quotes around a default value.
func unquote(value string) string {
	value = strings.TrimSpace(value)
	for _, q := range []string{"'", `"`, "`"} {
		if len(value) >= 2 && strings.HasPrefix(value, q) && strings.HasSuffix(value, q) {
			return value[1 : len(value)-1]
		}
	}
	return value
}

// trimSentence collapses spaces and trims trailing periods, commas and colons.
func trimSentence(text string) string {
	text = reSpaces.ReplaceAllString(strings.TrimSpace(text), " ")
	return strings.TrimRight(text, ".,:; ")
}

// label turns cleaned text into a label and optional help. Long text is
// split at the first sentence or comma when the first part is short enough.
func label(text string) (string, string) {
	for _, prefix := range []string{"The ", "Your "} {
		if rest, ok := strings.CutPrefix(text, prefix); ok && rest != "" {
			text = rest
		}
	}
	text = capitalize(text)
	if utf8.RuneCountInString(text) <= MaxLabelLength {
		return text, ""
	}
	for _, sep := range []string{". ", ", "} {
		head, tail, ok := strings.Cut(text, sep)
		if ok && utf8.RuneCountInString(head) <= MaxLabelLength {
			return trimSentence(head), sentence(tail)
		}
	}
	return text, ""
}

// sentence capitalises text and ends it with a period.
func sentence(text string) string {
	text = trimSentence(text)
	if text == "" {
		return ""
	}
	return capitalize(text) + "."
}

// capitalize upper cases the first letter.
func capitalize(text string) string {
	r, size := utf8.DecodeRuneInString(text)
	if r == utf8.RuneError || unicode.IsUpper(r) {
		return text
	}
	return string(unicode.ToUpper(r)) + text[size:]
}

var (
	reSecretKey   = regexp.MustCompile(`PASSWORD|PASSWD|SECRET|TOKEN|API_?KEY|PRIVATE_KEY|_KEY$`)
	reSecretText  = regexp.MustCompile(`(?i)\b(password|passwords|secret|token|tokens|api key|private key|passphrase|signature)\b`)
	reSecretStart = regexp.MustCompile(`(?i)^(password|secret|private key password)\b`)
	rePublicKey   = regexp.MustCompile(`(?i)\bpublic key\b`)
	// A label ending in one of these names something that is not a secret.
	reNotSecret = regexp.MustCompile(`(?i)\b(id|ids|name|username|user|login|email|url|uri|endpoint|file|path|host|region|ocid|fingerprint|type|mode)$`)
)

// isSecret guesses whether a value should be hidden while typed.
func isSecret(key, label string) bool {
	if reSecretStart.MatchString(label) {
		return true
	}
	if rePublicKey.MatchString(label) || reNotSecret.MatchString(label) || strings.HasSuffix(key, "_ID") || strings.HasSuffix(key, "_FILE") || strings.HasSuffix(key, "_PATH") {
		return false
	}
	return reSecretKey.MatchString(key) || reSecretText.MatchString(label)
}
