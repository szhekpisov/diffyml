#!/bin/sh
# diffyml installer
#
# Usage:
#   curl -fsSL https://szhekpisov.github.io/diffyml/install.sh | sh
#
# Environment variables:
#   DIFFYML_VERSION  version to install, e.g. 1.6.0 (default: latest release)
#   INSTALL_DIR      install directory (default: /usr/local/bin)
#   VERIFY           verification mode: sha256 (default), cosign, none

set -eu

REPO="szhekpisov/diffyml"
DIFFYML_VERSION="${DIFFYML_VERSION:-}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERIFY="${VERIFY:-sha256}"

err() { printf 'error: %s\n' "$1" >&2; exit 1; }
info() { printf '%s\n' "$1"; }
require() { command -v "$1" >/dev/null 2>&1 || err "$1 not found in PATH"; }

sha256_of() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        err "no sha256sum or shasum found in PATH"
    fi
}

require curl
require tar
require uname

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
    linux|darwin) ;;
    *) err "unsupported OS: $os (only linux and darwin are supported)" ;;
esac

arch=$(uname -m)
case "$arch" in
    x86_64|amd64)    arch=amd64 ;;
    aarch64|arm64)   arch=arm64 ;;
    *) err "unsupported architecture: $arch (only amd64 and arm64 are supported)" ;;
esac

if [ -z "$DIFFYML_VERSION" ]; then
    info "fetching latest version..."
    DIFFYML_VERSION=$(
        curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
            | grep '"tag_name"' \
            | head -1 \
            | cut -d'"' -f4
    )
    [ -n "$DIFFYML_VERSION" ] || err "could not determine latest version"
fi
DIFFYML_VERSION="${DIFFYML_VERSION#v}"

archive="diffyml_${DIFFYML_VERSION}_${os}_${arch}.tar.gz"
base_url="https://github.com/${REPO}/releases/download/v${DIFFYML_VERSION}"

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

info "downloading ${archive}..."
curl -fsSL -o "${tmpdir}/${archive}" "${base_url}/${archive}" \
    || err "download failed: ${base_url}/${archive}"

case "$VERIFY" in
    none)
        info "skipping verification (VERIFY=none)"
        ;;
    sha256|cosign)
        curl -fsSL -o "${tmpdir}/checksums.txt" "${base_url}/checksums.txt" \
            || err "could not fetch checksums.txt"

        if [ "$VERIFY" = cosign ]; then
            require cosign
            info "verifying cosign signature on checksums.txt..."
            curl -fsSL -o "${tmpdir}/checksums.txt.sigstore.json" \
                "${base_url}/checksums.txt.sigstore.json" \
                || err "could not fetch cosign bundle"
            cosign verify-blob "${tmpdir}/checksums.txt" \
                --bundle "${tmpdir}/checksums.txt.sigstore.json" \
                --certificate-identity-regexp "https://github.com/${REPO}/" \
                --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
                >/dev/null 2>&1 \
                || err "cosign verification failed"
            info "cosign signature verified"
        fi

        info "verifying SHA256..."
        expected=$(awk -v f="$archive" '$2 == f {print $1}' "${tmpdir}/checksums.txt")
        [ -n "$expected" ] || err "no SHA256 entry for ${archive} in checksums.txt"
        actual=$(sha256_of "${tmpdir}/${archive}")
        [ "$actual" = "$expected" ] \
            || err "SHA256 mismatch (expected $expected, got $actual)"
        info "SHA256 verified"
        ;;
    *)
        err "unknown VERIFY mode: $VERIFY (use sha256, cosign, or none)"
        ;;
esac

info "extracting..."
tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"
[ -f "${tmpdir}/diffyml" ] || err "diffyml binary not found in archive"
chmod +x "${tmpdir}/diffyml"

if mkdir -p "$INSTALL_DIR" 2>/dev/null && [ -w "$INSTALL_DIR" ]; then
    mv "${tmpdir}/diffyml" "${INSTALL_DIR}/diffyml"
elif command -v sudo >/dev/null 2>&1; then
    info "installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mkdir -p "$INSTALL_DIR"
    sudo mv "${tmpdir}/diffyml" "${INSTALL_DIR}/diffyml"
else
    err "cannot write to ${INSTALL_DIR} and sudo not available; set INSTALL_DIR"
fi

info ""
info "diffyml ${DIFFYML_VERSION} installed to ${INSTALL_DIR}/diffyml"

case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *) info "note: ${INSTALL_DIR} is not in your PATH" ;;
esac
