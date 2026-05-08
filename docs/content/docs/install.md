---
title: "Install"
weight: 10
---

# Install

## Homebrew

```bash
brew tap szhekpisov/diffyml
brew install diffyml
```

## Go install

```bash
go install github.com/szhekpisov/diffyml@latest
```

Make sure `$GOPATH/bin` is in your `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

## Docker

Multi-arch images (linux/amd64, linux/arm64) are published to GitHub Container Registry:

```bash
docker pull ghcr.io/szhekpisov/diffyml:latest

# Compare two files from the current directory
docker run --rm -v "$PWD:/work" -w /work ghcr.io/szhekpisov/diffyml:latest old.yaml new.yaml
```

Images are built from a [distroless](https://github.com/GoogleContainerTools/distroless) base and run as a non-root user. Use `:latest` or pin to a specific version (e.g. `:1.5.25`).

## Install script (Linux / macOS)

```bash
curl -fsSL https://szhekpisov.github.io/diffyml/install.sh | sh
```

Detects your OS and architecture, downloads the matching release archive, verifies its SHA256 against the signed `checksums.txt`, and installs the binary to `/usr/local/bin/diffyml`.

Environment variables:

| Variable | Default | Notes |
|---|---|---|
| `DIFFYML_VERSION` | latest release | Pin a specific version, e.g. `1.6.0`. |
| `INSTALL_DIR` | `/usr/local/bin` | Falls back to `sudo` if the directory isn't writable. |
| `VERIFY` | `sha256` | Use `cosign` to verify the cosign signature on `checksums.txt` first (requires `cosign` in `PATH`), or `none` to skip verification. |

Example pinning a version, installing into `~/bin`, and adding cosign verification:

```bash
DIFFYML_VERSION=1.6.0 INSTALL_DIR="$HOME/bin" VERIFY=cosign \
  sh -c "$(curl -fsSL https://szhekpisov.github.io/diffyml/install.sh)"
```

## Linux packages

Native `.deb`, `.rpm`, and `.apk` packages for amd64 and arm64 are attached to every [release](https://github.com/szhekpisov/diffyml/releases). The binary installs to `/usr/bin/diffyml`.

```bash
# Debian / Ubuntu
curl -LO "https://github.com/szhekpisov/diffyml/releases/download/v1.6.0/diffyml_1.6.0_linux_amd64.deb"
sudo dpkg -i diffyml_1.6.0_linux_amd64.deb

# RHEL / Fedora / openSUSE
curl -LO "https://github.com/szhekpisov/diffyml/releases/download/v1.6.0/diffyml_1.6.0_linux_amd64.rpm"
sudo rpm -i diffyml_1.6.0_linux_amd64.rpm

# Alpine
curl -LO "https://github.com/szhekpisov/diffyml/releases/download/v1.6.0/diffyml_1.6.0_linux_amd64.apk"
sudo apk add --allow-untrusted diffyml_1.6.0_linux_amd64.apk
```

## Direct binary download

If you'd rather not pipe a script to `sh`, the same archives are attached to every [release](https://github.com/szhekpisov/diffyml/releases) for Linux and macOS (amd64 and arm64). Download, extract, and move onto your `PATH`:

```bash
VERSION=1.6.0  # check the releases page for the latest
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -L "https://github.com/szhekpisov/diffyml/releases/download/v${VERSION}/diffyml_${VERSION}_${OS}_${ARCH}.tar.gz" \
  | tar -xz
sudo mv diffyml /usr/local/bin/
```

Archives are named `diffyml_<VERSION>_<os>_<arch>.tar.gz`. See [Verifying releases](#verifying-releases) to check signatures and provenance before installing.

## From source

```bash
git clone https://github.com/szhekpisov/diffyml.git
cd diffyml
go build -o diffyml
```

Requires Go 1.26.2 or later.

## Verifying releases

Every release ships:

- **Checksums** (`checksums.txt`) — SHA256 hashes for all archives
- **Cosign signature** (`checksums.txt.sigstore.json`) — keyless Sigstore signature
- **SBOMs** (`*.spdx.json`) — SPDX Software Bill of Materials per archive
- **SLSA provenance** — Level 3 attestation

```bash
cosign verify-blob checksums.txt \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp 'https://github.com/szhekpisov/diffyml/' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com'

# Linux
sha256sum --check checksums.txt --ignore-missing
# macOS
shasum -a 256 --check checksums.txt --ignore-missing
```

Verify SLSA provenance with `gh attestation`:

```bash
gh attestation verify diffyml_<VERSION>_linux_amd64.tar.gz --repo szhekpisov/diffyml
```

Verify a container image:

```bash
cosign verify \
  --registry-referrers-mode=oci-1-1 \
  --certificate-identity-regexp 'https://github.com/szhekpisov/diffyml/' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  ghcr.io/szhekpisov/diffyml:<VERSION>
```
