package catalog

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/nginxui/plugin-sdk-go/protocol"
)

func mustGet(t *testing.T, code string) Config {
	t.Helper()
	c, ok := Get(code)
	if !ok {
		t.Fatalf("%s is missing from the catalog", code)
	}
	return c
}

func fieldByKey(form *protocol.DNS01ProviderForm, key string) (protocol.DNS01ProviderField, bool) {
	for _, f := range form.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return protocol.DNS01ProviderField{}, false
}

func fieldKeys(form *protocol.DNS01ProviderForm) []string {
	keys := make([]string, 0, len(form.Fields))
	for _, f := range form.Fields {
		keys = append(keys, f.Key)
	}
	return keys
}

func TestFormCloudflare(t *testing.T) {
	form := mustGet(t, "cloudflare").Form()

	wantKeys := []string{
		"CF_API_EMAIL", "CF_API_KEY", "CF_DNS_API_TOKEN", "CF_ZONE_API_TOKEN",
		"CLOUDFLARE_POLLING_INTERVAL", "CLOUDFLARE_PROPAGATION_TIMEOUT", "CLOUDFLARE_TTL",
		"CLOUDFLARE_HTTP_TIMEOUT", "CLOUDFLARE_BASE_URL",
	}
	if got := fieldKeys(form); !reflect.DeepEqual(got, wantKeys) {
		t.Fatalf("keys = %v, want %v (aliases dropped, file order kept)", got, wantKeys)
	}

	token, _ := fieldByKey(form, "CF_DNS_API_TOKEN")
	want := protocol.DNS01ProviderField{
		Key: "CF_DNS_API_TOKEN", Label: "API token", Help: "Needs the DNS edit permission.",
		Group: protocol.DNS01FieldGroupCredential, Secret: true,
	}
	if token != want {
		t.Fatalf("CF_DNS_API_TOKEN = %+v, want %+v", token, want)
	}
	if zone, _ := fieldByKey(form, "CF_ZONE_API_TOKEN"); !zone.Optional || !zone.Secret {
		t.Fatalf("CF_ZONE_API_TOKEN = %+v, want optional and secret", zone)
	}

	ttl, _ := fieldByKey(form, "CLOUDFLARE_TTL")
	if ttl.Label != "TXT record TTL" || ttl.Default != "120" || ttl.Unit != protocol.DNS01FieldUnitSeconds || ttl.Group != protocol.DNS01FieldGroupSetting {
		t.Fatalf("CLOUDFLARE_TTL = %+v", ttl)
	}
	if timeout, _ := fieldByKey(form, "CLOUDFLARE_HTTP_TIMEOUT"); timeout.Default != "" {
		t.Fatalf("an empty upstream default must stay empty, got %q", timeout.Default)
	}

	wantMethods := []protocol.DNS01ProviderMethod{
		{Name: "API token", Recommended: true, Fields: []string{"CF_DNS_API_TOKEN", "CF_ZONE_API_TOKEN"}},
		{Name: "Global API key", Fields: []string{"CF_API_EMAIL", "CF_API_KEY"}},
	}
	if !reflect.DeepEqual(form.Methods, wantMethods) {
		t.Fatalf("methods = %+v", form.Methods)
	}
}

func TestFormAlibabaCloud(t *testing.T) {
	for _, code := range []string{"alidns", "aliesa"} {
		form := mustGet(t, code).Form()
		if len(form.Methods) != 2 || form.Methods[0].Name != "AccessKey" || form.Methods[0].Recommended || form.Methods[1].Name != "Instance RAM role" {
			t.Fatalf("%s methods = %+v", code, form.Methods)
		}
		for _, f := range form.Fields {
			if f.Label == "Instance RAM role" && f.Link == "" {
				t.Fatalf("%s RAM role lost its documentation link", code)
			}
			if f.Label == "STS security token" && !f.Optional {
				t.Fatalf("%s security token must be optional", code)
			}
		}
	}
	region, _ := fieldByKey(mustGet(t, "alidns").Form(), "ALICLOUD_REGION_ID")
	if region.Default != "cn-hangzhou" || region.Group != protocol.DNS01FieldGroupSetting {
		t.Fatalf("ALICLOUD_REGION_ID = %+v", region)
	}
}

func TestFormMethodsFromExample(t *testing.T) {
	// Designate has no method override for its keys shared by every method.
	c := mustGet(t, "designate")
	derived := c.exampleMethods(c.Form(), c.aliases())
	want := []protocol.DNS01ProviderMethod{
		{Name: "Username and password", Fields: []string{"OS_USERNAME", "OS_PASSWORD"}},
		{Name: "Application credential ID and application credential secret", Fields: []string{"OS_APPLICATION_CREDENTIAL_ID", "OS_APPLICATION_CREDENTIAL_SECRET"}},
	}
	if !reflect.DeepEqual(derived, want) {
		t.Fatalf("designate derived methods = %+v", derived)
	}

	c = mustGet(t, "cloudflare")
	derived = c.exampleMethods(c.Form(), c.aliases())
	if len(derived) != 2 || !slices.Equal(derived[0].Fields, []string{"CF_API_EMAIL", "CF_API_KEY"}) || !slices.Equal(derived[1].Fields, []string{"CF_DNS_API_TOKEN"}) {
		t.Fatalf("cloudflare aliases in the example must resolve, got %+v", derived)
	}
}

func TestEveryFormIsValid(t *testing.T) {
	list, err := List()
	if err != nil {
		t.Fatal(err)
	}
	withMethods := 0
	for _, c := range list {
		form := c.Form()
		if err := Validate(form); err != nil {
			t.Errorf("%s: %v", c.Code, err)
		}
		if form == nil {
			continue
		}
		if len(form.Methods) > 0 {
			withMethods++
		}
		aliases := c.aliases()
		override := overrides.Providers[c.Code]
		for _, f := range form.Fields {
			if _, isAlias := aliases[f.Key]; isAlias {
				t.Errorf("%s lists alias %s", c.Code, f.Key)
			}
			_, inCredentials := c.Configuration.Credentials[f.Key]
			_, inAdditional := c.Configuration.Additional[f.Key]
			added := slices.ContainsFunc(override.Add, func(a addedField) bool { return a.Key == f.Key })
			if !inCredentials && !inAdditional && !added {
				t.Errorf("%s lists undeclared key %s", c.Code, f.Key)
			}
			if added || override.Fields[f.Key].Group != nil {
				continue
			}
			if (f.Group == protocol.DNS01FieldGroupCredential) != inCredentials {
				t.Errorf("%s %s is in group %s", c.Code, f.Key, f.Group)
			}
		}
		for i := 1; i < len(form.Fields); i++ {
			if groupRank(form.Fields[i-1].Group) > groupRank(form.Fields[i].Group) {
				t.Errorf("%s lists setting %s before a credential", c.Code, form.Fields[i-1].Key)
			}
		}
	}
	if withMethods < 7 {
		t.Fatalf("%d providers offer a choice of sign-in methods, want at least 7", withMethods)
	}
}

// TestOverridesAreUsed keeps overrides.json free of entries that no longer
// match the catalog.
func TestOverridesAreUsed(t *testing.T) {
	list, err := List()
	if err != nil {
		t.Fatal(err)
	}
	texts := make(map[string]bool)
	for _, c := range list {
		if c.Configuration == nil {
			continue
		}
		for _, m := range []map[string]string{c.Configuration.Credentials, c.Configuration.Additional} {
			for _, d := range m {
				texts[clean(d).Text] = true
			}
		}
	}
	for text := range overrides.Phrases {
		if !texts[text] {
			t.Errorf("phrase override %q matches no description of an offered provider", text)
		}
	}

	byCode := make(map[string]Config)
	for _, c := range everyConfig {
		byCode[c.Code] = c
	}
	for code, o := range overrides.Providers {
		c, ok := byCode[code]
		if !ok {
			t.Errorf("override for unknown provider %s", code)
			continue
		}
		for key := range o.Fields {
			_, inCredentials := c.Configuration.Credentials[key]
			_, inAdditional := c.Configuration.Additional[key]
			if !inCredentials && !inAdditional {
				t.Errorf("override for unknown field %s of %s", key, code)
			}
		}
		for _, a := range o.Add {
			_, inCredentials := c.Configuration.Credentials[a.Key]
			_, inAdditional := c.Configuration.Additional[a.Key]
			if inCredentials || inAdditional {
				t.Errorf("%s adds %s, which the catalog already declares", code, a.Key)
			}
		}
	}
}

func TestFormOfConfigWithoutOrder(t *testing.T) {
	c := Config{Code: "x", Configuration: &Configuration{
		Credentials: map[string]string{"X_TOKEN": "API token", "X_TOKEN_ALIAS": "Alias to X_TOKEN"},
		Additional:  map[string]string{"X_TTL": "The TTL of the TXT record used for the DNS challenge in seconds (Default: 60)"},
	}}
	form := c.Form()
	want := []protocol.DNS01ProviderField{
		{Key: "X_TOKEN", Label: "API token", Group: protocol.DNS01FieldGroupCredential, Secret: true},
		{Key: "X_TTL", Label: "TXT record TTL", Group: protocol.DNS01FieldGroupSetting, Default: "60", Unit: protocol.DNS01FieldUnitSeconds},
	}
	if !reflect.DeepEqual(form.Fields, want) || form.Methods != nil {
		t.Fatalf("form = %+v", form)
	}
	if (Config{Code: "y"}).Form() != nil {
		t.Fatal("a provider without configuration has no form")
	}
}

func TestValidateRejects(t *testing.T) {
	cred := func(key string) protocol.DNS01ProviderField {
		return protocol.DNS01ProviderField{Key: key, Label: key, Group: protocol.DNS01FieldGroupCredential}
	}
	setting := protocol.DNS01ProviderField{Key: "S", Label: "S", Group: protocol.DNS01FieldGroupSetting}
	tests := map[string]*protocol.DNS01ProviderForm{
		"duplicate key": {Fields: []protocol.DNS01ProviderField{cred("A"), cred("A")}},
		"bad group":     {Fields: []protocol.DNS01ProviderField{{Key: "A", Label: "A", Group: "other"}}},
		"bad unit":      {Fields: []protocol.DNS01ProviderField{{Key: "A", Label: "A", Group: protocol.DNS01FieldGroupSetting, Unit: "minutes"}}},
		"no label":      {Fields: []protocol.DNS01ProviderField{{Key: "A", Group: protocol.DNS01FieldGroupCredential}}},
		"one method": {Fields: []protocol.DNS01ProviderField{cred("A")},
			Methods: []protocol.DNS01ProviderMethod{{Name: "A", Fields: []string{"A"}}}},
		"setting in method": {Fields: []protocol.DNS01ProviderField{cred("A"), setting},
			Methods: []protocol.DNS01ProviderMethod{{Name: "A", Fields: []string{"A"}}, {Name: "S", Fields: []string{"S"}}}},
		"two recommended": {Fields: []protocol.DNS01ProviderField{cred("A"), cred("B")},
			Methods: []protocol.DNS01ProviderMethod{{Name: "A", Recommended: true, Fields: []string{"A"}}, {Name: "B", Recommended: true, Fields: []string{"B"}}}},
		"duplicate method name": {Fields: []protocol.DNS01ProviderField{cred("A"), cred("B")},
			Methods: []protocol.DNS01ProviderMethod{{Name: "A", Fields: []string{"A"}}, {Name: "A", Fields: []string{"B"}}}},
		"empty method": {Fields: []protocol.DNS01ProviderField{cred("A")},
			Methods: []protocol.DNS01ProviderMethod{{Name: "A", Fields: []string{"A"}}, {Name: "B"}}},
	}
	for name, form := range tests {
		if Validate(form) == nil {
			t.Errorf("%s: Validate accepted %+v", name, form)
		}
	}
	if err := Validate(nil); err != nil {
		t.Fatalf("Validate(nil) = %v", err)
	}
}

func TestDisplayName(t *testing.T) {
	c := mustGet(t, "cloudflare")
	if c.DisplayName() != "Cloudflare" {
		t.Fatalf("display name = %q", c.DisplayName())
	}
}

func TestCanonicalKey(t *testing.T) {
	c := mustGet(t, "cloudflare")
	if got := c.CanonicalKey("CLOUDFLARE_DNS_API_TOKEN"); got != "CF_DNS_API_TOKEN" {
		t.Fatalf("CanonicalKey(alias) = %q", got)
	}
	if got := c.CanonicalKey("CF_API_KEY"); got != "CF_API_KEY" {
		t.Fatalf("CanonicalKey(canonical) = %q", got)
	}
	if got := (Config{Code: "y"}).CanonicalKey("X"); got != "X" {
		t.Fatalf("CanonicalKey without configuration = %q", got)
	}
}

func TestValidateMethodValues(t *testing.T) {
	cred := func(key string) protocol.DNS01ProviderField {
		return protocol.DNS01ProviderField{Key: key, Label: key, Group: protocol.DNS01FieldGroupCredential}
	}
	ok := &protocol.DNS01ProviderForm{
		Fields: []protocol.DNS01ProviderField{cred("A")},
		Methods: []protocol.DNS01ProviderMethod{
			{Name: "Key", Fields: []string{"A"}, Values: map[string]string{"MODE": "key"}},
			{Name: "Same key, other mode", Fields: []string{"A"}, Values: map[string]string{"MODE": "other"}},
			{Name: "Role", Fields: []string{}, Values: map[string]string{"MODE": "role"}},
		},
	}
	if err := Validate(ok); err != nil {
		t.Fatalf("Validate = %v", err)
	}

	// A credential field one method lists and another fixes.
	shared := &protocol.DNS01ProviderForm{
		Fields: []protocol.DNS01ProviderField{cred("KEY"), cred("ALG"), cred("USER")},
		Methods: []protocol.DNS01ProviderMethod{
			{Name: "Key", Fields: []string{"KEY", "ALG"}},
			{Name: "Kerberos", Fields: []string{"USER"}, Values: map[string]string{"ALG": "gss-tsig"}},
		},
	}
	if err := Validate(shared); err != nil {
		t.Fatalf("Validate(shared) = %v", err)
	}

	bad := map[string][]protocol.DNS01ProviderMethod{
		"value key is a field no method lists": {{Name: "A", Fields: []string{}, Values: map[string]string{"A": "x"}}, {Name: "B", Fields: []string{}}},
		"method lists and fixes a field":       {{Name: "A", Fields: []string{"A"}, Values: map[string]string{"A": "x"}}, {Name: "B", Fields: []string{}}},
		"empty value key":                      {{Name: "A", Fields: []string{}, Values: map[string]string{"": "x"}}, {Name: "B", Fields: []string{"A"}}},
		"identical methods":                    {{Name: "A", Fields: []string{"A"}, Values: map[string]string{"M": "1"}}, {Name: "B", Fields: []string{"A"}, Values: map[string]string{"M": "1"}}},
		"nil fields":                           {{Name: "A"}, {Name: "B", Fields: []string{"A"}}},
	}
	setting := protocol.DNS01ProviderField{Key: "S", Label: "S", Group: protocol.DNS01FieldGroupSetting}
	bad["value key is a setting field"] = []protocol.DNS01ProviderMethod{{Name: "A", Fields: []string{"A"}}, {Name: "B", Fields: []string{}, Values: map[string]string{"S": "x"}}}
	for name, methods := range bad {
		form := &protocol.DNS01ProviderForm{Fields: []protocol.DNS01ProviderField{cred("A"), setting}, Methods: methods}
		if Validate(form) == nil {
			t.Errorf("%s: Validate accepted %+v", name, methods)
		}
	}
}

func TestDocTables(t *testing.T) {
	doc := "\n## Base Configuration\n\n| Environment Variable Name | Description |\n|---|---|\n| `EXEC_MODE` | `RAW`, none |\n| `EXEC_PATH` | The path of the program. |\n\n## Additional Configuration\n\n| Name | Description |\n|---|---|\n| `EXEC_POLLING_INTERVAL` | Time between DNS propagation check in seconds (Default: 3). |\n\n## Description\n\n| `NOT_A_VARIABLE` | ignored |\n"
	cfg, credentials, additional := docTables(doc)
	if cfg == nil || !reflect.DeepEqual(credentials, []string{"EXEC_MODE", "EXEC_PATH"}) || !reflect.DeepEqual(additional, []string{"EXEC_POLLING_INTERVAL"}) {
		t.Fatalf("docTables = %+v, %v, %v", cfg, credentials, additional)
	}
	if cfg.Credentials["EXEC_PATH"] != "The path of the program." {
		t.Fatalf("EXEC_PATH = %q", cfg.Credentials["EXEC_PATH"])
	}
	if c, _, _ := docTables("no tables here"); c != nil {
		t.Fatal("a document without tables has no configuration")
	}
}

func TestFormsFromAudit(t *testing.T) {
	exec := mustGet(t, "exec").Form()
	path, ok := fieldByKey(exec, "EXEC_PATH")
	if !ok || path.Optional || path.Group != protocol.DNS01FieldGroupCredential {
		t.Fatalf("exec EXEC_PATH = %+v", path)
	}
	if mode, _ := fieldByKey(exec, "EXEC_MODE"); mode.Group != protocol.DNS01FieldGroupSetting || !mode.Optional {
		t.Fatalf("exec EXEC_MODE = %+v", mode)
	}

	joker := mustGet(t, "joker").Form()
	if _, listed := fieldByKey(joker, "JOKER_API_MODE"); listed {
		t.Fatal("joker lists its mode, which the methods fix")
	}
	if len(joker.Methods) != 3 || joker.Methods[2].Values["JOKER_API_MODE"] != "SVC" {
		t.Fatalf("joker methods = %+v", joker.Methods)
	}

	dnsupdate := mustGet(t, "dnsupdate").Form()
	if alg, _ := fieldByKey(dnsupdate, "DNSUPDATE_TSIG_ALGORITHM"); alg.Group != protocol.DNS01FieldGroupCredential || alg.Help != "" {
		t.Fatalf("dnsupdate algorithm = %+v", alg)
	}
	for _, m := range dnsupdate.Methods {
		kerberos := strings.HasPrefix(m.Name, "Kerberos")
		if kerberos != (m.Values["DNSUPDATE_TSIG_ALGORITHM"] == "gss-tsig") {
			t.Fatalf("dnsupdate method %q values = %v", m.Name, m.Values)
		}
	}

	oracle := mustGet(t, "oraclecloud").Form()
	if !oracle.Methods[0].Recommended || oracle.Methods[1].Values["OCI_AUTH_TYPE"] != "instance_principal" || len(oracle.Methods[1].Fields) != 0 {
		t.Fatalf("oraclecloud methods = %+v", oracle.Methods)
	}

	if _, listed := fieldByKey(mustGet(t, "hosttech").Form(), "HOSTTECH_PASSWORD"); listed {
		t.Fatal("hosttech lists a password the provider never reads")
	}
	if _, listed := fieldByKey(mustGet(t, "webglobe").Form(), "WEBGLOBE_TOKEN"); !listed {
		t.Fatal("webglobe misses its token")
	}
	for _, code := range []string{"s3", "stackpath"} {
		if _, ok := Get(code); ok {
			t.Fatalf("%s is offered", code)
		}
	}
	keys := mustGet(t, "joker").Keys()
	if !slices.Contains(keys, "JOKER_API_MODE") || !slices.Contains(keys, "JOKER_API_KEY") {
		t.Fatalf("joker keys = %v", keys)
	}
}
