// Command manifest regenerates plugin.json from the embedded provider catalog.
//
//	go run ./cmd/manifest
//
// The generated file is committed, so the plugin can be packaged without
// running the generator.
//
// With -platform it writes the plugin.json of one per-platform package
// instead: the committed manifest with server.executables narrowed to that
// platform, which is what build.sh puts into each archive.
//
//	go run ./cmd/manifest -platform linux-amd64 -out dist/stage/linux-amd64/plugin.json
//
// With -report it lists the credential form texts that need a look (long or
// uncleaned labels) and the phrases missing a translation.
//
//	go run ./cmd/manifest -report
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/nginxui/plugin-sdk-go/protocol"

	"github.com/nginxui/plugin-dns01/catalog"
)

// Plugin identity. Version is the single source of truth for the release
// artifacts; build.sh reads it back from the generated plugin.json.
const (
	PluginID          = "com.nginxui.dns01"
	PluginName        = "DNS-01 Challenge"
	PluginVersion     = "1.0.0"
	PluginDescription = "Validate domains for certificates through DNS records, with more than 200 DNS providers."
	MinNginxUIVersion = "3.0.0"
	IdleTimeout       = 300
)

// translations are the name and description in every language the plugin
// ships besides English, keyed by host locale code.
var translations = map[string]protocol.ManifestI18n{
	"zh_CN": {
		Name:        "DNS-01 验证",
		Description: "支持 200 多家 DNS 服务商，用 DNS 记录为证书验证域名。",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "通过 DNS 服务商的 API 创建和删除验证记录。",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "签发证书时选择已保存的 DNS 凭据",
			"checks":          "调整验证前对 DNS 记录的检查方式",
			"providers":       "可从 200 多家 DNS 服务商中选择",
			"provider-fields": "每家服务商只需填写它需要的字段",
		},
	},
	"zh_TW": {
		Name:        "DNS-01 驗證",
		Description: "支援 200 多家 DNS 服務商，以 DNS 記錄為憑證驗證網域。",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "透過 DNS 服務商的 API 建立和刪除驗證記錄。",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "簽發憑證時選擇已儲存的 DNS 憑證",
			"checks":          "調整驗證前對 DNS 記錄的檢查方式",
			"providers":       "可從 200 多家 DNS 服務商中選擇",
			"provider-fields": "每家服務商只需填寫它需要的欄位",
		},
	},
	"ja_JP": {
		Name:        "DNS-01 チャレンジ",
		Description: "200 以上の DNS プロバイダーに対応し、DNS レコードで証明書のドメインを検証します。",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "DNS プロバイダーの API を通じて検証用レコードを作成、削除します。",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "証明書の発行時に保存済みの DNS 認証情報を選択",
			"checks":          "検証前の DNS レコードの確認方法を調整",
			"providers":       "200 以上の DNS プロバイダーから選択",
			"provider-fields": "プロバイダーごとに必要な項目だけを入力",
		},
	},
	"ko_KR": {
		Name:        "DNS-01 챌린지",
		Description: "200개가 넘는 DNS 제공자를 지원하며, DNS 레코드로 인증서의 도메인을 검증합니다.",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "DNS 제공자의 API를 통해 검증 레코드를 만들고 삭제합니다.",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "인증서를 발급할 때 저장된 DNS 자격 증명 선택",
			"checks":          "검증 전에 DNS 레코드를 확인하는 방식 조정",
			"providers":       "200개가 넘는 DNS 제공자 중에서 선택",
			"provider-fields": "제공자마다 필요한 항목만 입력",
		},
	},
	"de_DE": {
		Name:        "DNS-01-Challenge",
		Description: "Validiert Domains für Zertifikate über DNS-Einträge, mit mehr als 200 DNS-Anbietern.",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "Um die Validierungseinträge über die API Ihres DNS-Anbieters anzulegen und zu entfernen.",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "Beim Ausstellen eines Zertifikats gespeicherte DNS-Zugangsdaten auswählen",
			"checks":          "Festlegen, wie DNS-Einträge vor der Validierung geprüft werden",
			"providers":       "Mehr als 200 DNS-Anbieter zur Auswahl",
			"provider-fields": "Jeder Anbieter fragt nur die Felder ab, die er braucht",
		},
	},
	"fr_FR": {
		Name:        "Challenge DNS-01",
		Description: "Valide les domaines des certificats par des enregistrements DNS, avec plus de 200 fournisseurs DNS.",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "Pour créer et supprimer les enregistrements de validation via l'API de votre fournisseur DNS.",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "Choisir des identifiants DNS enregistrés lors de l'émission d'un certificat",
			"checks":          "Régler la vérification des enregistrements DNS avant la validation",
			"providers":       "Plus de 200 fournisseurs DNS au choix",
			"provider-fields": "Chaque fournisseur ne demande que les champs dont il a besoin",
		},
	},
	"es": {
		Name:        "Desafío DNS-01",
		Description: "Valida los dominios de los certificados mediante registros DNS, con más de 200 proveedores DNS.",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "Para crear y eliminar los registros de validación a través de la API de su proveedor DNS.",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "Elegir una credencial DNS guardada al emitir un certificado",
			"checks":          "Ajustar cómo se comprueban los registros DNS antes de la validación",
			"providers":       "Más de 200 proveedores DNS para elegir",
			"provider-fields": "Cada proveedor solo pide los campos que necesita",
		},
	},
	"it_IT": {
		Name:        "Challenge DNS-01",
		Description: "Convalida i domini dei certificati tramite record DNS, con oltre 200 provider DNS.",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "Per creare e rimuovere i record di convalida tramite l'API del provider DNS.",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "Scegliere una credenziale DNS salvata durante l'emissione di un certificato",
			"checks":          "Regolare come vengono controllati i record DNS prima della convalida",
			"providers":       "Oltre 200 provider DNS tra cui scegliere",
			"provider-fields": "Ogni provider chiede solo i campi di cui ha bisogno",
		},
	},
	"pt_PT": {
		Name:        "Desafio DNS-01",
		Description: "Valida os domínios dos certificados através de registos DNS, com mais de 200 provedores DNS.",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "Para criar e remover os registos de validação através da API do seu provedor DNS.",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "Escolher uma credencial DNS guardada ao emitir um certificado",
			"checks":          "Ajustar como os registos DNS são verificados antes da validação",
			"providers":       "Mais de 200 provedores DNS à escolha",
			"provider-fields": "Cada provedor pede apenas os campos de que precisa",
		},
	},
	"ru_RU": {
		Name:        "Проверка DNS-01",
		Description: "Проверяет домены для сертификатов через DNS-записи, поддерживает более 200 DNS-провайдеров.",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "Чтобы создавать и удалять проверочные записи через API вашего DNS-провайдера.",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "Выбор сохранённых учётных данных DNS при выпуске сертификата",
			"checks":          "Настройка проверки DNS-записей перед подтверждением",
			"providers":       "Более 200 DNS-провайдеров на выбор",
			"provider-fields": "Каждый провайдер запрашивает только нужные ему поля",
		},
	},
	"uk_UA": {
		Name:        "Перевірка DNS-01",
		Description: "Перевіряє домени для сертифікатів через DNS-записи, підтримує понад 200 DNS-провайдерів.",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "Щоб створювати та видаляти перевірні записи через API вашого DNS-провайдера.",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "Вибір збережених облікових даних DNS під час випуску сертифіката",
			"checks":          "Налаштування перевірки DNS-записів перед підтвердженням",
			"providers":       "Понад 200 DNS-провайдерів на вибір",
			"provider-fields": "Кожен провайдер запитує лише потрібні йому поля",
		},
	},
	"tr_TR": {
		Name:        "DNS-01 Doğrulaması",
		Description: "200'den fazla DNS sağlayıcısıyla, sertifikalar için alan adlarını DNS kayıtları üzerinden doğrular.",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "Doğrulama kayıtlarını DNS sağlayıcınızın API'si üzerinden oluşturmak ve kaldırmak için.",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "Sertifika verirken kayıtlı bir DNS kimlik bilgisi seçin",
			"checks":          "Doğrulamadan önce DNS kayıtlarının nasıl denetleneceğini ayarlayın",
			"providers":       "Seçebileceğiniz 200'den fazla DNS sağlayıcısı",
			"provider-fields": "Her sağlayıcı yalnızca ihtiyaç duyduğu alanları ister",
		},
	},
	"vi_VN": {
		Name:        "Xác thực DNS-01",
		Description: "Xác thực tên miền cho chứng chỉ bằng bản ghi DNS, hỗ trợ hơn 200 nhà cung cấp DNS.",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "Để tạo và xóa bản ghi xác thực qua API của nhà cung cấp DNS của bạn.",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "Chọn thông tin xác thực DNS đã lưu khi cấp chứng chỉ",
			"checks":          "Điều chỉnh cách kiểm tra bản ghi DNS trước khi xác thực",
			"providers":       "Hơn 200 nhà cung cấp DNS để lựa chọn",
			"provider-fields": "Mỗi nhà cung cấp chỉ yêu cầu các trường cần thiết",
		},
	},
	"ar": {
		Name:        "تحدي DNS-01",
		Description: "يتحقق من نطاقات الشهادات عبر سجلات DNS، مع أكثر من 200 مزود DNS.",
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "لإنشاء سجلات التحقق وإزالتها عبر API مزود DNS لديك.",
		},
		ScreenshotCaptions: map[string]string{
			"credentials":     "اختيار بيانات اعتماد DNS محفوظة عند إصدار شهادة",
			"checks":          "ضبط طريقة فحص سجلات DNS قبل التحقق",
			"providers":       "أكثر من 200 مزود DNS للاختيار منها",
			"provider-fields": "لا يطلب كل مزود إلا الحقول التي يحتاجها",
		},
	},
}

// screenshots are the catalog images under docs/screenshots, with English
// captions. translations carries the other languages by id.
var screenshots = []protocol.ManifestScreenshot{
	{ID: "credentials", Path: "docs/screenshots/1-credentials.png", DarkPath: "docs/screenshots/1-credentials-dark.png", Caption: "Pick a saved DNS credential when issuing a certificate"},
	{ID: "checks", Path: "docs/screenshots/2-checks.png", DarkPath: "docs/screenshots/2-checks-dark.png", Caption: "Adjust how DNS records are checked before validation"},
	{ID: "providers", Path: "docs/screenshots/3-providers.png", DarkPath: "docs/screenshots/3-providers-dark.png", Caption: "More than 200 DNS providers to choose from"},
	{ID: "provider-fields", Path: "docs/screenshots/4-provider-fields.png", DarkPath: "docs/screenshots/4-provider-fields-dark.png", Caption: "Each provider asks only for the fields it needs"},
}

// platforms are the targets build.sh cross compiles, in manifest order.
var platforms = []struct{ OS, Arch string }{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"linux", "386"},
	{"linux", "arm"},
	{"linux", "riscv64"},
	{"linux", "loong64"},
	{"linux", "mips"},
	{"linux", "mipsle"},
	{"linux", "mips64"},
	{"linux", "mips64le"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
	{"windows", "amd64"},
	{"windows", "arm64"},
	{"windows", "386"},
}

// ExecutablePath returns the packaged path of one platform's binary.
func ExecutablePath(goos, goarch string) string {
	name := fmt.Sprintf("dns01-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return "server/dist/" + name
}

// Build assembles the manifest. It is deterministic: the provider list comes
// from the catalog sorted by name, and every map is encoded in key order.
func Build() (*protocol.Manifest, error) {
	providers, err := dns01Providers()
	if err != nil {
		return nil, err
	}

	executables := make(map[string]string, len(platforms))
	for _, p := range platforms {
		executables[p.OS+"-"+p.Arch] = ExecutablePath(p.OS, p.Arch)
	}

	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	shared, err := webappSharedRanges(root)
	if err != nil {
		return nil, err
	}

	return &protocol.Manifest{
		ID:                PluginID,
		Name:              PluginName,
		Version:           PluginVersion,
		Description:       PluginDescription,
		APIVersion:        protocol.APIVersion,
		MinNginxUIVersion: MinNginxUIVersion,
		I18n:              translations,
		Screenshots:       screenshots,
		Server: &protocol.ManifestServer{
			Executables:        executables,
			Lifecycle:          protocol.LifecycleOnDemand,
			IdleTimeoutSeconds: IdleTimeout,
		},
		IconPath: "webapp/dist/icon.svg",
		Webapp: &protocol.ManifestWebapp{
			BundlePath: "webapp/dist/main.js",
			StylePath:  "webapp/dist/style.css",
			Shared:     shared,
		},
		Capabilities: []string{protocol.CapabilityDNS01},
		Permissions:  []string{protocol.PermissionNetwork},
		PermissionReasons: map[string]string{
			protocol.PermissionNetwork: "To create and remove the validation records through the API of your DNS provider.",
		},
		DNS01: &protocol.ManifestDNS01{Providers: providers},
		SettingsSchema: &protocol.SettingsSchema{
			Settings: []protocol.SettingsField{
				{
					Key:         "recursive_nameservers",
					Type:        "list",
					DisplayName: "Recursive DNS servers",
					HelpText:    "Used to check that DNS records are visible. Empty means the system resolvers.",
					Default:     []string{},
				},
				{
					Key:         "default_propagation_timeout_seconds",
					Type:        "number",
					DisplayName: "Default wait time (seconds)",
					HelpText:    "How long to wait at most for a record to become visible when the provider does not say.",
					Default:     120,
				},
			},
		},
	}, nil
}

// webappManifestFragment is the shape @nginxui/plugin-sdk/vite writes to
// webapp/dist/manifest.webapp.json. Only Shared is consumed here: bundle_path
// and style_path are fixed by the layout build.sh packages.
type webappManifestFragment struct {
	Shared map[string]string `json:"shared"`
}

// webappSharedRanges reads the semver ranges the webapp bundle was built
// against, if the bundle has been built. A missing file is not an error: the
// Go tests and cross compilation do not depend on the JS toolchain having run.
func webappSharedRanges(root string) (map[string]string, error) {
	path := filepath.Join(root, "webapp", "dist", "manifest.webapp.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var fragment webappManifestFragment
	if err := json.Unmarshal(data, &fragment); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return fragment.Shared, nil
}

// dns01Providers converts the catalog into manifest entries.
func dns01Providers() ([]protocol.DNS01Provider, error) {
	list, err := catalog.List()
	if err != nil {
		return nil, err
	}

	out := make([]protocol.DNS01Provider, 0, len(list))
	for _, c := range list {
		form := c.Form()
		if form == nil {
			form = &protocol.DNS01ProviderForm{}
		}
		if form.Fields == nil {
			form.Fields = []protocol.DNS01ProviderField{}
		}
		if err := catalog.Validate(form); err != nil {
			return nil, fmt.Errorf("provider %s: %w", c.Code, err)
		}

		entry := protocol.DNS01Provider{Name: c.DisplayName(), Code: c.Code, Form: *form}
		if c.Links != nil && c.Links.API != "" {
			entry.Links = &protocol.DNS01ProviderLinks{API: c.Links.API}
		}

		out = append(out, entry)
	}
	return out, nil
}

// Render encodes the manifest the way it is committed: two space indentation
// and a trailing newline.
func Render() ([]byte, error) {
	manifest, err := Build()
	if err != nil {
		return nil, err
	}
	return encode(manifest)
}

// encode writes a manifest in the committed layout.
func encode(manifest *protocol.Manifest) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(manifest); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// FilterPlatform narrows a rendered manifest to one "<goos>-<goarch>"
// executable. A per-platform package must declare exactly the platform it
// ships, while the catalog release keeps the full
// map in its manifest snapshot.
func FilterPlatform(data []byte, platform string) ([]byte, error) {
	var manifest protocol.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	if manifest.Server == nil {
		return nil, fmt.Errorf("the manifest has no server block")
	}
	executable, ok := manifest.Server.Executables[platform]
	if !ok {
		return nil, fmt.Errorf("the manifest declares no executable for %s", platform)
	}
	manifest.Server.Executables = map[string]string{platform: executable}
	return encode(&manifest)
}

// repoRoot resolves the module root from this file's location.
func repoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("manifest: unable to locate the generator source")
	}
	return filepath.Abs(filepath.Join(filepath.Dir(file), "..", ".."))
}

func main() {
	platform := flag.String("platform", "", `write the manifest of the "<goos>-<goarch>" package instead of regenerating plugin.json`)
	in := flag.String("in", "", "manifest to narrow with -platform (default: the committed plugin.json)")
	out := flag.String("out", "", "output file (default: the committed plugin.json, required with -platform)")
	review := flag.Bool("report", false, "list form texts that need a look and missing translations, write nothing")
	flag.Parse()

	if *review {
		if err := report(os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "manifest:", err)
			os.Exit(1)
		}
		return
	}

	if err := run(*platform, *in, *out); err != nil {
		fmt.Fprintln(os.Stderr, "manifest:", err)
		os.Exit(1)
	}
}

func run(platform, in, out string) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}

	var data []byte
	switch {
	case platform == "":
		if data, err = Render(); err != nil {
			return err
		}
		if out == "" {
			out = filepath.Join(root, "plugin.json")
		}
	case out == "":
		return fmt.Errorf("-platform needs -out, the committed plugin.json keeps every platform")
	default:
		if in == "" {
			in = filepath.Join(root, "plugin.json")
		}
		source, err := os.ReadFile(in)
		if err != nil {
			return err
		}
		if data, err = FilterPlatform(source, platform); err != nil {
			return err
		}
	}

	if err = os.WriteFile(out, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d bytes)\n", out, len(data))
	return nil
}
