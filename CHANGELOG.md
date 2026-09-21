# Changelog

All notable changes to this plugin are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[semantic versioning](https://semver.org/).

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
- `webapp/`: a browser bundle, built with `@nginx-ui/plugin-sdk`, that
  replaces the host's fallback DNS-01 challenge form with a credential
  selector plus switches for CNAME following and the authoritative/recursive
  propagation checks, translated into zh_CN, zh_TW and ja_JP.
- `cmd/manifest` now fills `webapp.shared` from `webapp/dist/manifest.webapp.json`
  when the webapp has been built.
