# Releasing diffyml

This document describes the release process for diffyml, including semantic versioning guidelines, GitHub release creation, and the pre-release checklist.

## Semantic Versioning

diffyml follows [Semantic Versioning 2.0.0](https://semver.org/) for all releases.

### Version Format

```
vMAJOR.MINOR.PATCH
```

- **MAJOR**: Increment when making incompatible API changes
- **MINOR**: Increment when adding functionality in a backward-compatible manner
- **PATCH**: Increment when making backward-compatible bug fixes

### Examples

| Version | Description |
|---------|-------------|
| `v1.0.0` | First stable release |
| `v1.1.0` | New feature added (backward-compatible) |
| `v1.1.1` | Bug fix (backward-compatible) |
| `v2.0.0` | Breaking change in CLI flags or output format |

### When to Bump Each Version

**Bump MAJOR when:**
- Removing or renaming CLI flags
- Changing the default output format
- Changing exit codes for existing conditions
- Any change that breaks existing scripts or workflows

**Bump MINOR when:**
- Adding new CLI flags
- Adding new output formats
- Adding new features (e.g., new diff detection capabilities)
- Performance improvements

**Bump PATCH when:**
- Fixing bugs without changing behavior
- Fixing typos in documentation
- Internal code refactoring without API changes

## GitHub Release Process

### Step 1: Ensure All Tests Pass

Before creating a release, verify that all tests pass:

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Check test coverage
go test -cover ./...
```

### Step 2: Update Version Information

Determine the new version number based on the changes since the last release:

```bash
# View recent commits
git log --oneline v1.0.0..HEAD

# Determine version bump (major/minor/patch)
NEW_VERSION="v1.1.0"  # Example
```

### Step 3: Create and Push the Git Tag

```bash
# Create an annotated tag
git tag -a v1.1.0 -m "Release v1.1.0"

# Push the tag to GitHub
git push origin v1.1.0
```

### Step 4: Create GitHub Release

1. Go to the repository's **Releases** page on GitHub
2. Click **Draft a new release**
3. Select the tag you just created
4. Set the release title: `diffyml v1.1.0`
5. Write release notes (see format below)
6. Ensure **Set as the latest release** is checked
7. Click **Publish release**

### Release Notes Format

```markdown
## What's Changed

### New Features
- Added support for XYZ (#123)

### Bug Fixes
- Fixed issue with ABC (#124)

### Documentation
- Updated README with new examples

## Checksums

SHA256 checksums for source archives:
- `diffyml-v1.1.0.tar.gz`: [checksum]
- `diffyml-v1.1.0.zip`: [checksum]

## Installation

### Homebrew (coming soon)
```bash
brew install diffyml
```

### From Source
```bash
go install github.com/USER/diffyml@v1.1.0
```

**Full Changelog**: https://github.com/USER/diffyml/compare/v1.0.0...v1.1.0
```

### Release Artifact URLs

GitHub automatically generates source archives for each tag:

- **Tarball**: `https://github.com/USER/diffyml/archive/refs/tags/v1.1.0.tar.gz`
- **Zip**: `https://github.com/USER/diffyml/archive/refs/tags/v1.1.0.zip`

These URLs are used by Homebrew formulas to download the source code.

## Pre-Release Checklist

Before creating a release, ensure all of the following are complete:

### Code Quality

- [ ] All tests pass: `go test ./...`
- [ ] No race conditions: `go test -race ./...`
- [ ] Code builds successfully: `go build -o diffyml`
- [ ] Binary runs correctly: `./diffyml --version`

### Documentation

- [ ] README.md is up-to-date
- [ ] All new features are documented
- [ ] CLI help text matches README
- [ ] CHANGELOG is updated (if maintained)

### Version Management

- [ ] Version follows semantic versioning (vMAJOR.MINOR.PATCH)
- [ ] Tag is annotated (not lightweight)
- [ ] Version can be injected via ldflags

### Build Verification

```bash
# Build with version injection
VERSION="1.1.0"
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${DATE}" -o diffyml

# Verify version output
./diffyml --version
```

### Dependency Verification

- [ ] All dependencies are versioned: `go mod verify`
- [ ] No security vulnerabilities: `go list -m all`
- [ ] All dependencies have open-source licenses

### Repository Structure

- [ ] LICENSE file exists and is MIT
- [ ] README.md has all required sections
- [ ] main.go is in repository root
- [ ] go.mod and go.sum are up-to-date

### Release Requirements

- [ ] Release tag follows format: `vMAJOR.MINOR.PATCH`
- [ ] Release is marked as stable (not pre-release)
- [ ] Release notes describe all changes
- [ ] Source archives are generated automatically

## Homebrew Formula

After creating a GitHub release, the Homebrew formula can be updated:

### Formula URL Pattern

```ruby
url "https://github.com/USER/diffyml/archive/refs/tags/v1.1.0.tar.gz"
```

### Computing SHA256

After the release is created, compute the SHA256 checksum:

```bash
curl -sL https://github.com/USER/diffyml/archive/refs/tags/v1.1.0.tar.gz | shasum -a 256
```

### Formula Test

The formula should include a test that verifies the version:

```ruby
test do
  assert_match "v1.1.0", shell_output("#{bin}/diffyml --version")
end
```

## Troubleshooting

### Tag Already Exists

If you need to re-create a tag:

```bash
# Delete local tag
git tag -d v1.1.0

# Delete remote tag
git push origin :refs/tags/v1.1.0

# Create new tag
git tag -a v1.1.0 -m "Release v1.1.0"
git push origin v1.1.0
```

### Build Fails on Tag

If the build fails after tagging:

1. Do not delete the tag if a release has been published
2. Fix the issue in a new commit
3. Create a new patch version (e.g., v1.1.1)

### Wrong Version in Binary

Ensure ldflags are correctly set during build:

```bash
# Check what version is embedded
./diffyml --version

# If incorrect, rebuild with explicit ldflags
go build -ldflags "-X main.version=1.1.0" -o diffyml
```
