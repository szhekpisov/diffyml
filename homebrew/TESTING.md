# Testing the Homebrew Formula

This document explains how to test the diffyml Homebrew formula locally before submitting to homebrew/core or a custom tap.

## Prerequisites

- macOS with Homebrew installed
- Go toolchain installed (`brew install go`)
- Git repository with a tagged release

## Testing Methods

### Method 1: Test with Local Formula File

The simplest way to test is using a local formula file:

```bash
# 1. Create a release tag (if not already done)
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 2. Download the tarball and calculate checksum
VERSION="1.0.0"
curl -sL "https://github.com/USER/diffyml/archive/refs/tags/v${VERSION}.tar.gz" -o /tmp/diffyml.tar.gz
shasum -a 256 /tmp/diffyml.tar.gz

# 3. Update the formula with the correct checksum
# Edit homebrew/diffyml.rb.example with the calculated SHA256

# 4. Install from the local formula
brew install --build-from-source ./homebrew/diffyml.rb.example

# 5. Test the installation
diffyml --version
diffyml --help
```

### Method 2: Test with a Custom Tap

For more realistic testing, create a custom tap:

```bash
# 1. Create a new repository named homebrew-diffyml (or homebrew-tap)
# Structure:
#   homebrew-diffyml/
#   └── Formula/
#       └── diffyml.rb

# 2. Copy the formula (update USER, VERSION, and CHECKSUM first)
mkdir -p Formula
cp diffyml.rb.example Formula/diffyml.rb

# 3. Commit and push to GitHub

# 4. Tap and install
brew tap USER/diffyml
brew install diffyml

# 5. Test
diffyml --version
```

### Method 3: Test Build from Source

Test that the project builds correctly with Homebrew's Go environment:

```bash
# 1. Install Go via Homebrew (matches formula dependency)
brew install go

# 2. Build with the same flags Homebrew will use
VERSION="1.0.0"
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${DATE}" -o diffyml

# 3. Verify the binary
./diffyml --version
./diffyml --help
```

## Running Formula Tests

Homebrew formulas include a `test` block that runs automatically:

```bash
# Run the formula's test block
brew test diffyml

# Or with verbose output
brew test diffyml --verbose
```

The test block in diffyml.rb verifies:
1. `--version` flag outputs the version number
2. `--help` flag outputs usage information
3. Basic YAML diff functionality works

## Verifying the Formula

Before submitting, run Homebrew's audit tools:

```bash
# Check formula style and best practices
brew audit --strict diffyml

# Check formula with online checks
brew audit --strict --online diffyml

# Lint the Ruby syntax
brew style diffyml
```

## Common Issues and Solutions

### Issue: SHA256 Mismatch

**Symptom**: Installation fails with checksum error

**Solution**: Recalculate the checksum from the actual release tarball:
```bash
curl -sL "https://github.com/USER/diffyml/archive/refs/tags/v1.0.0.tar.gz" | shasum -a 256
```

### Issue: Build Fails with Go Errors

**Symptom**: `go build` fails during installation

**Solution**:
1. Ensure `go.mod` is valid: `go mod verify`
2. Check Go version compatibility in `go.mod`
3. Test build locally: `go build -o diffyml`

### Issue: Version Not Displayed

**Symptom**: `diffyml --version` shows "dev" instead of version number

**Solution**: Ensure ldflags are correctly set in the formula:
```ruby
ldflags = %W[
  -X main.version=#{version}
]
system "go", "build", *std_go_args(ldflags:)
```

### Issue: Test Block Fails

**Symptom**: `brew test diffyml` fails

**Solution**:
1. Check that the test creates proper YAML files
2. Verify exit codes are handled correctly (exit 1 for diffs is expected)
3. Run tests manually to debug:
   ```bash
   diffyml --version
   diffyml --help
   echo "a: 1" > /tmp/a.yaml
   echo "a: 2" > /tmp/b.yaml
   diffyml /tmp/a.yaml /tmp/b.yaml
   ```

### Issue: Formula Audit Warnings

**Symptom**: `brew audit` reports warnings

**Common fixes**:
- `desc` should not start with "A" or "An"
- `desc` should be under 80 characters
- Use SPDX license identifier (e.g., "MIT" not "MIT License")
- Ensure `homepage` URL is valid and accessible

## Uninstalling

To clean up after testing:

```bash
# Uninstall the formula
brew uninstall diffyml

# Remove the tap (if using custom tap)
brew untap USER/diffyml

# Clean up cached files
brew cleanup
```

## Next Steps

After successful local testing:

1. **For homebrew/core submission**:
   - Fork https://github.com/Homebrew/homebrew-core
   - Add formula to `Formula/y/diffyml.rb`
   - Run `brew audit --strict --online diffyml`
   - Submit a pull request

2. **For custom tap**:
   - Create `homebrew-diffyml` repository
   - Add formula to `Formula/diffyml.rb`
   - Document installation: `brew tap USER/diffyml && brew install diffyml`
