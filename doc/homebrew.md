# Testing the Homebrew Formula

This document explains how to test the diffyml Homebrew formula.

## How it works

GoReleaser automatically updates the Homebrew tap (`szhekpisov/homebrew-diffyml`) on each tagged release. The generated formula installs pre-built binaries — no Go toolchain required on the user's machine.

## Prerequisites

- macOS with Homebrew installed
- A tagged release published via GoReleaser

## Install and test

```bash
# Tap and install
brew tap szhekpisov/diffyml
brew install diffyml

# Verify
diffyml --version

# Run the formula's test block
brew test diffyml --verbose
```

## Local testing with goreleaser

To verify the formula generation locally before a release:

```bash
# Dry-run (will not push to the tap)
goreleaser release --snapshot --clean

# Check the generated formula
cat dist/homebrew/Formula/diffyml.rb
```

## Auditing

```bash
brew audit --strict diffyml
brew style diffyml
```

## Uninstalling

```bash
brew uninstall diffyml
brew untap szhekpisov/diffyml
brew cleanup
```
