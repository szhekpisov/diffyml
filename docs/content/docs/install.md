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

## Release binaries

Pre-built binaries for Linux, macOS, and Windows are published on the [releases page](https://github.com/szhekpisov/diffyml/releases). Download the archive matching your platform, extract, and place `diffyml` on your `PATH`.

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
