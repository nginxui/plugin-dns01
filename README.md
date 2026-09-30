# DNS-01 Challenge

The official [NGINX UI](https://github.com/0xJacky/nginx-ui) plugin for the
ACME DNS-01 challenge. It publishes the `_acme-challenge` TXT record through
any of the 222 DNS providers [lego](https://github.com/go-acme/lego) supports,
waits for the record to propagate and cleans it up afterwards.

* Plugin id: `com.nginxui.dns01`
* Requires NGINX UI 2.7.0 or newer
* Plugin API version 1

## Features

* 222 DNS providers, taken straight from the lego provider catalog, with their
  credential fields, help text and vendor documentation links.
* Credential validation without issuing a certificate: the plugin builds the
  provider from the values you entered and reports the first field that is
  missing.
* Propagation check against the zone's authoritative nameservers and against
  your recursive resolvers, with CNAME delegation support.
* Per-certificate switches to disable CNAME following, the authoritative check
  or the recursive check when a network does not allow them.
* Propagation timings reported per provider, so NGINX UI waits as long as the
  vendor actually needs instead of using one global timeout.
* Runs on demand: the host starts the process when a certificate needs it and
  stops it again after five idle minutes.

## Capabilities

| Capability | Methods |
| --- | --- |
| `dns01` | `dns01.present`, `dns01.cleanup`, `dns01.validate`, `dns01.options`, `dns01.check` |

The plugin declares no other capability. It serves no HTTP route and registers
no cron entry.

## Permissions and why

| Permission | Why it is needed |
| --- | --- |
| `network` | The provider code talks to your DNS vendor's API, and the propagation check sends DNS queries to your resolvers and to the zone's authoritative nameservers. Nothing else in the plugin opens a connection. |

The plugin does not request `kv`, `cron`, `notify`, `metrics.read`,
`core_api` or any `credentials.read:*` permission. Credentials arrive as part
of the request the host sends; the plugin never reads them from storage on its
own.

## Settings

| Key | Type | Default | Meaning |
| --- | --- | --- | --- |
| `recursive_nameservers` | list | empty | `host:port` entries used for the propagation check and the zone lookup. An entry without a port gets `:53`. Empty means the system resolvers from `/etc/resolv.conf`, falling back to `1.1.1.1` and `1.0.0.1`. A comma separated string from older hosts is still accepted. |
| `default_propagation_timeout_seconds` | number | 120 | Applied to providers that do not report a propagation timeout of their own. Providers that do report one always win. lego's own default is 60 seconds. |

Per-certificate options travel with the request rather than the settings:

| Option | Effect |
| --- | --- |
| `disable_cname` | Do not follow the challenge record's CNAME. Also exported as `LEGO_DISABLE_CNAME_SUPPORT` for the duration of the call. |
| `disable_authoritative_ns_propagation` | Skip the authoritative nameserver check. |
| `disable_recursive_ns_propagation` | Skip the recursive nameserver check. |

## Credential fields

Each provider declares its own environment variables. They are listed in
`plugin.json` under `dns01.providers[].form` and rendered by NGINX UI as a
form, so there is nothing to memorise. For example, Cloudflare accepts
either `CF_API_EMAIL` plus `CF_API_KEY`, or `CF_DNS_API_TOKEN` (optionally with
`CF_ZONE_API_TOKEN`).

Values are exported into the process environment only for the duration of one
call and the previous environment is restored afterwards, so two certificates
using different accounts of the same vendor never see each other's values. A
value wrapped in matching single or double quotes is unquoted first, which
makes a pasted `"token"` behave like a bare token.

Credential values are never written to a log line, never included in an error
message and never sent to the host. When a provider cannot be built, the plugin
reports the name of the offending field, not its value.

## Supported platforms

| OS | Architectures |
| --- | --- |
| Linux | amd64, arm64 |
| macOS | amd64, arm64 |
| Windows | amd64, arm64 |

The binaries are statically linked (`CGO_ENABLED=0`) and built with
`-trimpath -ldflags "-s -w"`. Each platform ships as its own package, see
[Packaging](#packaging).

## What data leaves the machine

* **To your DNS vendor.** The credentials you entered, the challenge record
  name and its TXT value, through the vendor's own API endpoint. The endpoint
  is the one lego's provider uses, or the one you configured through that
  provider's base URL variable.
* **To your DNS resolvers and the zone's authoritative nameservers.** Plain
  DNS queries for the challenge record's SOA, NS, CNAME and TXT records.
* **Nothing else.** The plugin contacts no telemetry endpoint, reports no
  usage, and its provider catalog is embedded in the binary, so listing the
  providers needs no network access at all.

Refreshing the catalog with `go run ./cmd/lego_config` is the one operation
that downloads anything, and it is a maintainer tool, not part of running the
plugin.

## Web bundle

`webapp/` is a small Vue 3 + TypeScript project, built with
[@nginxui/plugin-sdk](https://github.com/nginxui/plugin-sdk-web),
that replaces the host's built-in DNS-01 challenge form
(`certificate.challenge.form:dns01`) with one that also exposes the plugin's
own per-certificate options: disabling CNAME following, and skipping the
authoritative or the recursive nameserver propagation check.

```bash
cd webapp
bun install
bun run build     # writes webapp/dist/{main.js,style.css,manifest.webapp.json}
```

`webapp/dist/manifest.webapp.json` is not part of the package; it only tells
`go run ./cmd/manifest` which semver ranges the bundle was built against, so
`plugin.json`'s `webapp.shared` always matches the versions of vue,
vue-router, pinia, antdv-next and @vueuse/core the bundle actually used.
Build the webapp **before** regenerating the manifest:

```bash
(cd webapp && bun install && bun run build)
go run ./cmd/manifest
```

### Credential form

Every provider in `plugin.json` carries a `form` (plugin spec DNS01-14), the
only description of the values it accepts: plain labels instead of variable
names, the ways to sign in, defaults, units and which fields are secret or
optional. `cmd/manifest` derives it from the catalog descriptions and the
examples in `catalog/data`. What the rules cannot work out lives in
`catalog/overrides.json`: `phrases` maps a cleaned upstream description to a
better label and help for every provider, `providers` corrects single fields
(label, group, optional, hidden), adds the ones the upstream description
misses, names the sign-in methods with the fixed values that select them on
the provider side, and hides providers the plugin does not offer (`s3`,
which serves HTTP-01, and `stackpath`, which has shut down). The overrides
come from reading the provider code of the pinned lego release, so check
them again when bumping it. A test fails when a catalog provider is missing
from lego's registry.

Labels, help texts and method names are English source strings. Their
translations are in `catalog/i18n/<locale>.json`, which the webapp registers
with the host. The tests fail when a phrase has no translation or a
translation is no longer used; `go run ./cmd/manifest -report` lists long or
uncleaned labels and the coverage per language, which is where to look after
refreshing the catalog.

`build.sh` copies `webapp/dist` into the package automatically when the
directory exists, so a plugin built without Bun still works, minus the custom
DNS-01 form (the host falls back to its own generic DNS challenge UI).

## Packaging

One binary is 54 to 61 MiB. An archive with all six would unpack to about
345 MiB, more than the 256 MiB a host accepts (spec PKG-7), and every node
would download five binaries it never runs. The release is therefore split
into one package per platform:

```text
dist/com.nginxui.dns01-<version>-linux-amd64.tar.gz
dist/com.nginxui.dns01-<version>-linux-amd64.tar.gz.sha256
dist/com.nginxui.dns01-<version>-linux-arm64.tar.gz
...
dist/com.nginxui.dns01-<version>-windows-arm64.tar.gz
```

Every package holds one binary under `server/dist/`, the web bundle, the
documentation and a `plugin.json` whose `server.executables` names only that
platform, as the plugin spec requires for a per-platform package (PKG-12). The
committed `plugin.json` keeps all six platforms; it is what the catalog
publishes as the release manifest snapshot. `go run ./cmd/manifest -platform
<goos>-<goarch> -out <file>` writes the narrowed copy, which is what
`build.sh` puts into each archive.

Every package also carries these files at its root, right after `plugin.json`:

* `plugin.sums` lists the sha256 of every file of the package except itself
  and the signature, one `<sha256>  <path>` line each, sorted by path. Running
  `sha256sum -c plugin.sums` inside an extracted package checks it.
* `plugin.sums.minisig`, in a signed package, is the minisign signature over
  `plugin.sums`. NGINX UI derives the trust level of the package from the key
  that made it.

`build.sh` signs only when `MINISIGN_KEY` names a minisign secret key file:

```bash
MINISIGN_KEY=/path/to/plugin.key ./build.sh
```

Without it the packages are unsigned, and a host installs them only in
developer mode. minisign asks for the key password once per package;
`MINISIGN_PASSWORD` answers the prompt without a terminal, and a key created
without a password (`minisign -G -W`) never asks. The signature lives inside
the archive, so a release publishes no `.minisig` files.

The `.sha256` file next to each archive is in `sha256sum` format and feeds the
`downloads` map of the catalog release, where it serves as a download
integrity check:

```json
{
  "downloads": {
    "linux-amd64": {
      "url": "<release asset base url>/com.nginxui.dns01-1.0.0-linux-amd64.tar.gz",
      "sha256": "<digest from the .sha256 file>"
    }
  }
}
```

NGINX UI picks the package of the platform it runs on; cluster sync fetches
the package of each child node's platform when that node cannot reach the
catalog itself. For an offline node, `nginx-ui plugin fetch com.nginxui.dns01
--platform <goos>-<goarch>` (or `--platform all`) downloads the packages to
carry over, and dropping them into the node's `plugins/packages/` directory
installs the one matching that node.

## Releasing

Set the version in `cmd/manifest`, regenerate `plugin.json`, move the
`Unreleased` notes in `CHANGELOG.md` under the new version, then push a tag
`v<version>` that matches `plugin.json`. `.github/workflows/release.yml`
rebuilds the webapp, runs the tests, signs the six packages with the key kept
in the `release` environment and publishes them as a GitHub Release with the
changelog section as its notes. The catalog polls this repository's releases
and opens a pull request for the new version on its own.

## Development

```bash
go run ./cmd/lego_config          # refresh catalog/data from the latest lego release
(cd webapp && bun install && bun run build)  # optional: build the webapp bundle first
go run ./cmd/manifest             # regenerate plugin.json
go run ./cmd/manifest -report     # form texts to review, translation coverage
go test -race -count=1 ./...      # run the tests
./build.sh --host-only            # build and package the current platform only
./build.sh                        # cross compile, one package per platform
MINISIGN_KEY=plugin.key ./build.sh  # the same, with signed packages
```

The plugin depends on the
[plugin-sdk-go](https://github.com/nginxui/plugin-sdk-go) module and the
webapp on `@nginxui/plugin-sdk` from npm
([plugin-sdk-web](https://github.com/nginxui/plugin-sdk-web)). Until they are
published, and to work against local checkouts, point at the checkouts next to
this repository; `go.work` is ignored by git and `bun link` leaves
`package.json` as it is:

```bash
go work init . ../plugin-sdk-go
go work edit -replace=github.com/nginxui/plugin-sdk-go@v0.1.0=../plugin-sdk-go
(cd ../plugin-sdk-web && bun install && bun run build && bun link)
(cd webapp && bun link @nginxui/plugin-sdk)
```

## Support

Report problems at
<https://github.com/0xJacky/nginx-ui/issues>. Include the provider code, the
NGINX UI version and the plugin log lines from stderr. Never paste credential
values into an issue.

## License

AGPL-3.0. See [LICENSE](LICENSE).
