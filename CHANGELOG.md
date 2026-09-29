# Changelog

All notable changes to this plugin are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[semantic versioning](https://semver.org/).

## [Unreleased]

### Added

- Every provider in `plugin.json` describes its credential form: short labels
  in plain words instead of variable names, the ways to sign in where there
  is more than one (Cloudflare, Alibaba Cloud, OVH and others, with the
  recommended one marked), defaults and units, optional and secret fields,
  and provider settings kept apart from the credentials. Alias variables are
  left out. The labels, help texts and method names ship in Simplified
  Chinese, Traditional Chinese and Japanese.
- More providers offer a choice of how to sign in, checked against the
  provider code: Azure (client secret, client certificate, federated token,
  managed identity), Oracle Cloud (API signing key, instance principal,
  session token profile), Google Cloud, Route 53, Lightsail, Akamai EdgeDNS,
  RFC 2136 (TSIG key, key file, Kerberos, no key), Joker (API key, account
  or dynamic DNS credentials), Gandi, Hetzner and ClouDNS. A method can need
  no input at all and can set a fixed value that selects it, such as the
  authentication mode.
- Fields the provider reads but the upstream description left out: session
  tokens for Route 53 and Lightsail, the Lightsail region, the Hetzner legacy
  key, the Azure certificate password and federated token, the Oracle
  private key as text, the Name.com and NIFCLOUD endpoints, and the Webglobe
  token, which the description misnamed.
- The external program provider has a form: the program path, its mode and
  the timing settings.
- `plugin.json` no longer carries `configuration` or `links.go_client`; the
  `form` replaces the raw variable list. An error about a missing value names
  the key the form shows, also when the provider reported an alias.

### Changed

- Optional marks and groups follow the provider code: many values that were
  shown as required are optional (Route 53 and Lightsail keys, ALWAYSDATA
  account, Liara team, VegaDNS key pair, INWX two-factor secret, HTTP request
  sign-in), and sign-in values listed as settings moved to the credentials.
- The Hosttech password is gone; the provider only reads the API token.
- `s3` (an HTTP-01 solver) and `stackpath` (shut down) are no longer offered.
- Only a vendor's clearly preferred sign-in method is marked as recommended.
- lego v5.5.2.

- The DNS challenge form follows the compact row layout of the certificate
  form. The credential select has a Manage button next to it and a New
  credential entry in its list, and it asks for another credential when the
  saved one was deleted instead of showing a bare number.
- The three validation switches moved under Advanced options, collapsed by
  default with a summary of what was changed. They now read as positive
  statements (Follow CNAME, Check authoritative servers, Check public
  resolvers) and are on by default. The saved options keep their meaning.
- The recursive DNS servers setting is a list. A comma separated value saved
  by an earlier version is still read.
- Clearer names and help for the plugin settings and a plugin description
  that no longer names the underlying library.

- `build.sh` packages every platform separately as
  `com.nginxui.dns01-<version>-<goos>-<goarch>.tar.gz`, with a `plugin.json`
  that declares only that platform's executable and a `.sha256` file next to
  each archive. The six-platform archive is gone: it unpacked to about
  345 MiB, over the 256 MiB package limit of the host, and made every node
  download binaries it never runs.
- Every package carries `plugin.sums`, the sha256 of each file in it, and
  `build.sh` signs that list into `plugin.sums.minisig` when `MINISIGN_KEY`
  names a minisign secret key. NGINX UI derives the trust level from this
  signature, so the release assets no longer include `.minisig` files. The
  `.sha256` file next to each archive stays as a download integrity check.
- `build.sh` reads the signing key password from `MINISIGN_PASSWORD` when it
  is set, and `.github/workflows/release.yml` builds, signs and publishes the
  packages of a `v<version>` tag as a GitHub Release.
- `./build.sh --host-only` now also writes the package of the current
  platform, and the build prints the list of files it produced.
- `cmd/manifest -platform <goos>-<goarch> -out <file>` writes the narrowed
  `plugin.json` of one per-platform package.
- `webapp/dist/manifest.webapp.json` is no longer copied into the package, it
  only feeds `cmd/manifest`.

## [1.0.0] - 2026-09-22

### Added

- DNS-01 challenge solving for the 224 DNS providers of lego v5.4.1, with the
  credential fields, help text and documentation links taken from the lego
  provider catalog.
- `dns01.validate`: build the provider from the entered credentials and report
  the first missing field, without contacting the vendor API.
- `dns01.options`: report the propagation timeout, polling interval and
  sequential interval the selected provider asks for.
- `dns01.check`: propagation check against the zone's authoritative
  nameservers and the configured recursive resolvers, with CNAME delegation
  support and per-certificate switches to disable each step.
- Settings for the recursive nameservers and for the propagation timeout used
  by providers that do not report one.
- `cmd/lego_config` to refresh the provider catalog from a lego release, and
  `cmd/manifest` to regenerate `plugin.json` from it.
- Release packaging for linux, darwin and windows on amd64 and arm64.
- `webapp/`: a browser bundle, built with `@nginxui/plugin-sdk`, that
  replaces the host's fallback DNS-01 challenge form with a credential
  selector plus switches for CNAME following and the authoritative/recursive
  propagation checks, translated into zh_CN, zh_TW and ja_JP.
- `cmd/manifest` now fills `webapp.shared` from `webapp/dist/manifest.webapp.json`
  when the webapp has been built.
