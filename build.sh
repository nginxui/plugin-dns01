#!/usr/bin/env bash
# Build the release artifacts of the DNS-01 plugin.
#
#   ./build.sh              cross compile every supported platform and package
#   ./build.sh --host-only  build only the current platform, no archive
#
# The package layout matches what nginx-ui expects when it installs a plugin:
# plugin.json sits at the root of the archive, next to server/, webapp/ and the
# documentation.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST="${ROOT}/dist"
PKG="${DIST}/pkg"

PLUGIN_ID="com.nginxui.dns01"
BIN_PREFIX="dns01"

HOST_ONLY=0
if [[ "${1:-}" == "--host-only" ]]; then
  HOST_ONLY=1
elif [[ -n "${1:-}" ]]; then
  echo "usage: $0 [--host-only]" >&2
  exit 2
fi

cd "${ROOT}"

if [[ ! -f plugin.json ]]; then
  echo "plugin.json is missing, run: go run ./cmd/manifest" >&2
  exit 1
fi

# The version is read back from the generated manifest so it has one source.
VERSION="$(sed -n 's/^  "version": "\(.*\)",$/\1/p' plugin.json | head -n 1)"
if [[ -z "${VERSION}" ]]; then
  echo "could not read the version from plugin.json" >&2
  exit 1
fi

PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
  "windows/arm64"
)

if [[ "${HOST_ONLY}" -eq 1 ]]; then
  PLATFORMS=("$(go env GOOS)/$(go env GOARCH)")
fi

rm -rf "${PKG}"
mkdir -p "${PKG}/server/dist"

echo "building ${PLUGIN_ID} ${VERSION}"

for platform in "${PLATFORMS[@]}"; do
  goos="${platform%%/*}"
  goarch="${platform##*/}"

  name="${BIN_PREFIX}-${goos}-${goarch}"
  if [[ "${goos}" == "windows" ]]; then
    name="${name}.exe"
  fi

  out="${PKG}/server/dist/${name}"

  echo "  ${goos}/${goarch}"
  GOWORK=off CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
    go build -trimpath -ldflags "-s -w" -o "${out}" .

  size="$(du -h "${out}" | cut -f1 | tr -d '[:space:]')"
  echo "    ${name} (${size})"
done

# The browser bundle is optional; it is copied only when it was built.
if [[ -d "${ROOT}/webapp/dist" ]]; then
  mkdir -p "${PKG}/webapp/dist"
  cp -R "${ROOT}/webapp/dist/." "${PKG}/webapp/dist/"
  echo "  bundled webapp/dist"
fi

cp plugin.json "${PKG}/plugin.json"
for doc in README.md LICENSE CHANGELOG.md; do
  [[ -f "${ROOT}/${doc}" ]] && cp "${ROOT}/${doc}" "${PKG}/${doc}"
done

if [[ "${HOST_ONLY}" -eq 1 ]]; then
  echo "host-only build in ${PKG} (no archive)"
  exit 0
fi

ARCHIVE="${DIST}/${PLUGIN_ID}-${VERSION}.tar.gz"
rm -f "${ARCHIVE}"

# -C makes plugin.json the first entry at the root of the archive.
tar -czf "${ARCHIVE}" -C "${PKG}" .

echo "packaged ${ARCHIVE} ($(du -h "${ARCHIVE}" | cut -f1 | tr -d '[:space:]'))"
