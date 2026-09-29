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
- `plugin.json` no longer carries `configuration` or `links.go_client`; the
  `form` replaces the raw variable list. An error about a missing value names
  the key the form shows, also when the provider reported an alias.

### Changed

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
