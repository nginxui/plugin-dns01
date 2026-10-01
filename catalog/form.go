package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/nginxui/plugin-sdk-go/protocol"
)

// overridesJSON holds the hand-maintained corrections to the derived forms.
//
//go:embed overrides.json
var overridesJSON []byte

// phrase replaces the label and help derived from a cleaned description.
type phrase struct {
	Label string  `json:"label"`
	Help  *string `json:"help,omitempty"`
	Unit  string  `json:"unit,omitempty"`
}

// fieldOverride corrects one field of one provider. Nil means keep.
type fieldOverride struct {
	// Hidden leaves the field out of the form.
	Hidden   bool    `json:"hidden,omitempty"`
	Group    *string `json:"group,omitempty"`
	Label    *string `json:"label,omitempty"`
	Help     *string `json:"help,omitempty"`
	Optional *bool   `json:"optional,omitempty"`
	Secret   *bool   `json:"secret,omitempty"`
	Default  *string `json:"default,omitempty"`
	Unit     *string `json:"unit,omitempty"`
	Link     *string `json:"link,omitempty"`
}

// addedField is a value the provider reads that its description leaves out.
type addedField struct {
	Key   string `json:"key"`
	Group string `json:"group"`
	fieldOverride
}

// providerOverride corrects one provider.
type providerOverride struct {
	// Hidden leaves the provider out of the plugin.
	Hidden  bool                           `json:"hidden,omitempty"`
	Name    string                         `json:"name,omitempty"`
	Fields  map[string]fieldOverride       `json:"fields,omitempty"`
	Add     []addedField                   `json:"add,omitempty"`
	Methods []protocol.DNS01ProviderMethod `json:"methods,omitempty"`
}

// overrideSet is the shape of overrides.json.
type overrideSet struct {
	// Phrases maps a cleaned description to a better label, for every
	// provider that uses it.
	Phrases   map[string]phrase           `json:"phrases"`
	Providers map[string]providerOverride `json:"providers"`
}

var overrides = mustOverrides()

func mustOverrides() overrideSet {
	var set overrideSet
	if err := json.Unmarshal(overridesJSON, &set); err != nil {
		panic(fmt.Sprintf("catalog: overrides.json: %v", err))
	}
	return set
}

// hiddenCode reports whether overrides.json leaves a provider out.
func hiddenCode(code string) bool {
	return overrides.Providers[code].Hidden
}

// DisplayName is the provider name shown to people.
func (c Config) DisplayName() string {
	if o, ok := overrides.Providers[c.Code]; ok && o.Name != "" {
		return o.Name
	}
	return c.Name
}

// Form derives the credential form layout of a provider. It returns nil when
// the provider has no configuration.
func (c Config) Form() *protocol.DNS01ProviderForm {
	if c.Configuration == nil {
		return nil
	}
	override := overrides.Providers[c.Code]
	aliases := c.aliases()

	form := &protocol.DNS01ProviderForm{}
	add := func(keys []string, descriptions map[string]string, group string) {
		for _, key := range keys {
			if _, isAlias := aliases[key]; isAlias {
				continue
			}
			field := buildField(key, descriptions[key], group)
			if o, ok := override.Fields[key]; ok {
				if o.Hidden {
					continue
				}
				o.apply(&field)
			}
			form.Fields = append(form.Fields, field)
		}
	}
	add(orderedKeys(c.credentialOrder, c.Configuration.Credentials), c.Configuration.Credentials, protocol.DNS01FieldGroupCredential)
	add(orderedKeys(c.additionalOrder, c.Configuration.Additional), c.Configuration.Additional, protocol.DNS01FieldGroupSetting)
	for _, a := range override.Add {
		field := protocol.DNS01ProviderField{Key: a.Key, Group: a.Group}
		a.apply(&field)
		if field.Secret = isSecret(field.Key, field.Label); a.Secret != nil {
			field.Secret = *a.Secret
		}
		form.Fields = append(form.Fields, field)
	}
	// Credentials come first, whatever group an override moved a field to.
	slices.SortStableFunc(form.Fields, func(a, b protocol.DNS01ProviderField) int {
		return groupRank(a.Group) - groupRank(b.Group)
	})
	if len(form.Fields) == 0 {
		return nil
	}

	if override.Methods != nil {
		form.Methods = make([]protocol.DNS01ProviderMethod, 0, len(override.Methods))
		for _, m := range override.Methods {
			m.Fields = append([]string{}, m.Fields...)
			form.Methods = append(form.Methods, m)
		}
	} else {
		form.Methods = c.exampleMethods(form, aliases)
	}
	return form
}

func groupRank(group string) int {
	if group == protocol.DNS01FieldGroupCredential {
		return 0
	}
	return 1
}

// Keys lists every config key the plugin passes on to the provider: the
// form fields and the fixed values of its methods.
func (c Config) Keys() []string {
	form := c.Form()
	if form == nil {
		return nil
	}
	var keys []string
	for _, f := range form.Fields {
		keys = append(keys, f.Key)
	}
	for _, m := range form.Methods {
		for key := range m.Values {
			if !slices.Contains(keys, key) {
				keys = append(keys, key)
			}
		}
	}
	slices.Sort(keys)
	return keys
}

// orderedKeys returns the keys of m in file order, falling back to sorted
// order for a Config not built by load.
func orderedKeys(order []string, m map[string]string) []string {
	if len(order) == len(m) {
		return order
	}
	return sortedKeys(m)
}

// CanonicalKey returns the key an alias stands for, or key itself.
func (c Config) CanonicalKey(key string) string {
	if c.Configuration == nil {
		return key
	}
	if target, ok := c.aliases()[key]; ok {
		return target
	}
	return key
}

// aliases maps every alias key to its canonical key.
func (c Config) aliases() map[string]string {
	out := make(map[string]string)
	all := make(map[string]bool)
	for k := range c.Configuration.Credentials {
		all[k] = true
	}
	for k := range c.Configuration.Additional {
		all[k] = true
	}
	for _, m := range []map[string]string{c.Configuration.Credentials, c.Configuration.Additional} {
		for key, description := range m {
			if target, ok := aliasTarget(description); ok && all[target] {
				out[key] = target
			}
		}
	}
	return out
}

// buildField derives one field from its upstream description.
func buildField(key, description, group string) protocol.DNS01ProviderField {
	c := clean(description)
	field := protocol.DNS01ProviderField{
		Key:      key,
		Group:    group,
		Optional: c.Optional,
		Default:  c.Default,
		Link:     c.Link,
	}
	if c.Seconds {
		field.Unit = protocol.DNS01FieldUnitSeconds
	}

	if p, ok := overrides.Phrases[c.Text]; ok {
		field.Label = p.Label
		if p.Help != nil {
			field.Help = *p.Help
		}
		if p.Unit != "" {
			field.Unit = p.Unit
		}
	} else {
		field.Label, field.Help = label(c.Text)
	}
	if field.Label == "" {
		field.Label = key
	}
	if c.Example != "" && field.Help == "" {
		field.Help = "For example: " + c.Example
	}
	field.Secret = isSecret(key, field.Label)
	return field
}

func (o fieldOverride) apply(field *protocol.DNS01ProviderField) {
	if o.Group != nil {
		field.Group = *o.Group
	}
	if o.Label != nil {
		field.Label = *o.Label
	}
	if o.Help != nil {
		field.Help = *o.Help
	}
	if o.Optional != nil {
		field.Optional = *o.Optional
	}
	if o.Secret != nil {
		field.Secret = *o.Secret
	}
	if o.Default != nil {
		field.Default = *o.Default
	}
	if o.Unit != nil {
		field.Unit = *o.Unit
	}
	if o.Link != nil {
		field.Link = *o.Link
	}
}

// exampleMethods derives the ways to sign in from the upstream example.
func (c Config) exampleMethods(form *protocol.DNS01ProviderForm, aliases map[string]string) []protocol.DNS01ProviderMethod {
	credential := make(map[string]string)
	for _, f := range form.Fields {
		if f.Group == protocol.DNS01FieldGroupCredential {
			credential[f.Key] = f.Label
		}
	}

	var methods []protocol.DNS01ProviderMethod
	for _, block := range parseExample(c.Example) {
		var keys []string
		for _, key := range block.Keys {
			if target, ok := aliases[key]; ok {
				key = target
			}
			if _, ok := credential[key]; ok && !slices.Contains(keys, key) {
				keys = append(keys, key)
			}
		}
		methods = append(methods, protocol.DNS01ProviderMethod{Name: block.Name, Fields: keys})
	}
	return normalizeMethods(methods, credential)
}

// normalizeMethods drops keys every method shares, empty and duplicate
// methods, and names unnamed ones after their fields. Fewer than two
// methods means there is no choice to offer.
func normalizeMethods(methods []protocol.DNS01ProviderMethod, labels map[string]string) []protocol.DNS01ProviderMethod {
	methods = slices.DeleteFunc(slices.Clone(methods), func(m protocol.DNS01ProviderMethod) bool { return len(m.Fields) == 0 })
	if len(methods) < 2 {
		return nil
	}
	shared := slices.Clone(methods[0].Fields)
	for _, m := range methods[1:] {
		shared = slices.DeleteFunc(shared, func(k string) bool { return !slices.Contains(m.Fields, k) })
	}

	var out []protocol.DNS01ProviderMethod
	seen := make(map[string]bool)
	for _, m := range methods {
		m.Fields = slices.DeleteFunc(slices.Clone(m.Fields), func(k string) bool { return slices.Contains(shared, k) })
		id := strings.Join(m.Fields, ",")
		if len(m.Fields) == 0 || seen[id] {
			continue
		}
		seen[id] = true
		if m.Name == "" {
			m.Name = joinLabels(m.Fields, labels)
		}
		out = append(out, m)
	}
	if len(out) < 2 {
		return nil
	}
	return out
}

// joinLabels names a method after its fields, as in "Username and password".
func joinLabels(keys []string, labels map[string]string) string {
	parts := make([]string, 0, len(keys))
	for i, key := range keys {
		l := labels[key]
		if i > 0 {
			l = lowerFirst(l)
		}
		parts = append(parts, l)
	}
	switch len(parts) {
	case 1:
		return parts[0]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
	}
}

// lowerFirst lower cases the first letter unless the word is an acronym.
func lowerFirst(text string) string {
	if len(text) < 2 || (text[1] >= 'A' && text[1] <= 'Z') {
		return text
	}
	return strings.ToLower(text[:1]) + text[1:]
}

// Validate checks a form: unique keys, known groups and
// units, uniquely named methods over credential fields, fixed values that
// are no field or a credential field only other methods list, no two
// identical methods, at most one recommended method.
func Validate(form *protocol.DNS01ProviderForm) error {
	if form == nil {
		return nil
	}
	group := make(map[string]string)
	for _, f := range form.Fields {
		if f.Key == "" || f.Label == "" {
			return fmt.Errorf("field %q has no key or label", f.Key)
		}
		if _, dup := group[f.Key]; dup {
			return fmt.Errorf("field %s is listed twice", f.Key)
		}
		if f.Group != protocol.DNS01FieldGroupCredential && f.Group != protocol.DNS01FieldGroupSetting {
			return fmt.Errorf("field %s has group %q", f.Key, f.Group)
		}
		if f.Unit != "" && f.Unit != protocol.DNS01FieldUnitSeconds {
			return fmt.Errorf("field %s has unit %q", f.Key, f.Unit)
		}
		group[f.Key] = f.Group
	}
	if len(form.Methods) == 1 {
		return fmt.Errorf("a single method offers no choice")
	}
	recommended := 0
	names := make(map[string]bool)
	shapes := make(map[string]string)
	for _, m := range form.Methods {
		if m.Name == "" {
			return fmt.Errorf("a method has no name")
		}
		if m.Fields == nil {
			return fmt.Errorf("method %q has a nil field list", m.Name)
		}
		if names[m.Name] {
			return fmt.Errorf("method %q is listed twice", m.Name)
		}
		names[m.Name] = true
		if m.Recommended {
			recommended++
		}
		for _, key := range m.Fields {
			if group[key] != protocol.DNS01FieldGroupCredential {
				return fmt.Errorf("method %q lists %s, which is not a credential field", m.Name, key)
			}
		}
		for key := range m.Values {
			if key == "" {
				return fmt.Errorf("method %q has an empty value key", m.Name)
			}
			g, isField := group[key]
			if !isField {
				continue
			}
			if g != protocol.DNS01FieldGroupCredential {
				return fmt.Errorf("method %q fixes %s, which is a setting field", m.Name, key)
			}
			if slices.Contains(m.Fields, key) {
				return fmt.Errorf("method %q both lists and fixes %s", m.Name, key)
			}
			if !slices.ContainsFunc(form.Methods, func(o protocol.DNS01ProviderMethod) bool { return slices.Contains(o.Fields, key) }) {
				return fmt.Errorf("method %q fixes field %s, which no method lists", m.Name, key)
			}
		}
		shape := methodShape(m)
		if other, dup := shapes[shape]; dup {
			return fmt.Errorf("methods %q and %q have the same fields and values", other, m.Name)
		}
		shapes[shape] = m.Name
	}
	if recommended > 1 {
		return fmt.Errorf("%d methods are recommended", recommended)
	}
	return nil
}

// methodShape identifies a method by its sorted fields and values.
func methodShape(m protocol.DNS01ProviderMethod) string {
	fields := slices.Sorted(slices.Values(m.Fields))
	values := make([]string, 0, len(m.Values))
	for k, v := range m.Values {
		values = append(values, k+"="+v)
	}
	slices.Sort(values)
	return strings.Join(fields, ",") + "|" + strings.Join(values, ",")
}
