#!/usr/bin/env bash
# Build the release artifacts of the DNS-01 plugin.
#
#   ./build.sh              build every supported platform, one package each
#   ./build.sh --host-only  build and package the current platform only
#
# Every platform gets its own package, dist/<id>-<version>-<goos>-<goarch>.tar.gz,
# holding one binary and a plugin.json whose server.executables names only that
# platform (plugin spec PKG-12). One binary is 54 to 61 MiB: a package with all
# six would unpack to about 345 MiB, over the 256 MiB a host accepts (PKG-7),
# and every node would download five binaries it never runs. A <archive>.sha256
# file sits next to each archive for the catalog.
#
# The package layout matches what nginx-ui expects when it installs a plugin:
# plugin.json sits at the root of the archive, next to server/, webapp/ and the
# documentation.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST="${ROOT}/dist"
STAGE="${DIST}/stage"
BIN="${DIST}/bin"

PLUGIN_ID="com.nginxui.dns01"
BIN_PREFIX="dns01"

usage() {
  echo "usage: $0 [--host-only]"
}

HOST_ONLY=0
for arg in "$@"; do
  case "${arg}" in
    --host-only) HOST_ONLY=1 ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      usage >&2
      exit 2
      ;;
  esac
done

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

# Keep macOS tar from adding AppleDouble "._*" entries for extended attributes.
export COPYFILE_DISABLE=1

# sha256_line prints "<digest>  <file name>", the format sha256sum -c reads.
sha256_line() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$(dirname "${file}")" && sha256sum "$(basename "${file}")")
  else
    (cd "$(dirname "${file}")" && shasum -a 256 "$(basename "${file}")")
  fi
}

# binary_name is the packaged file name of one platform's executable.
binary_name() {
  local goos="$1" goarch="$2"
  local name="${BIN_PREFIX}-${goos}-${goarch}"
  if [[ "${goos}" == "windows" ]]; then
    name="${name}.exe"
  fi
  echo "${name}"
}

# stage_common copies what every package ships besides the binaries.
stage_common() {
  local dir="$1"
  # The browser bundle is optional; it is copied only when it was built. The
  # manifest fragment only feeds cmd/manifest and is not part of a package.
  if [[ -d "${ROOT}/webapp/dist" ]]; then
    mkdir -p "${dir}/webapp/dist"
    cp -R "${ROOT}/webapp/dist/." "${dir}/webapp/dist/"
    rm -f "${dir}/webapp/dist/manifest.webapp.json"
  fi
  for doc in README.md LICENSE CHANGELOG.md; do
    if [[ -f "${ROOT}/${doc}" ]]; then
      cp "${ROOT}/${doc}" "${dir}/${doc}"
    fi
  done
}

# package_dir writes one archive with plugin.json as its first entry. The top
# level entries are listed explicitly so the archive has no "./" root entry.
package_dir() {
  local dir="$1" archive="$2"
  local entries=(plugin.json)
  for entry in README.md LICENSE CHANGELOG.md server webapp; do
    if [[ -e "${dir}/${entry}" ]]; then
      entries+=("${entry}")
    fi
  done
  rm -f "${archive}" "${archive}.sha256"
  tar -czf "${archive}" -C "${dir}" "${entries[@]}"
  sha256_line "${archive}" >"${archive}.sha256"
  OUTPUTS+=("${archive}" "${archive}.sha256")
}

rm -rf "${STAGE}" "${BIN}" "${DIST}/pkg"
rm -f "${DIST}/${PLUGIN_ID}-${VERSION}"*.tar.gz "${DIST}/${PLUGIN_ID}-${VERSION}"*.tar.gz.sha256
mkdir -p "${STAGE}" "${BIN}"

# The manifest tool narrows plugin.json to one platform per package.
MANIFEST_TOOL="${DIST}/.manifest-tool"
GOWORK=off go build -o "${MANIFEST_TOOL}" ./cmd/manifest

echo "building ${PLUGIN_ID} ${VERSION}"

OUTPUTS=()
for platform in "${PLATFORMS[@]}"; do
  goos="${platform%%/*}"
  goarch="${platform##*/}"
  key="${goos}-${goarch}"
  name="$(binary_name "${goos}" "${goarch}")"

  echo "  ${goos}/${goarch}"
  GOWORK=off CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
    go build -trimpath -ldflags "-s -w" -o "${BIN}/${name}" .
  echo "    ${name} ($(du -h "${BIN}/${name}" | cut -f1 | tr -d '[:space:]'))"

  dir="${STAGE}/${key}"
  mkdir -p "${dir}/server/dist"
  cp "${BIN}/${name}" "${dir}/server/dist/${name}"
  stage_common "${dir}"
  "${MANIFEST_TOOL}" -in "${ROOT}/plugin.json" -platform "${key}" -out "${dir}/plugin.json" >/dev/null

  package_dir "${dir}" "${DIST}/${PLUGIN_ID}-${VERSION}-${key}.tar.gz"
done

# Every binary also sits in its stage directory, which stays for inspection.
rm -rf "${MANIFEST_TOOL}" "${BIN}"

echo "packages:"
for file in "${OUTPUTS[@]}"; do
  printf "  %-6s %s\n" "$(du -h "${file}" | cut -f1 | tr -d '[:space:]')" "${file#"${ROOT}/"}"
done
