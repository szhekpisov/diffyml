# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.5.8] - 2026-03-13

### Fixed

- Unify document prefix to 0-based (document N) format (#57) ([#57](https://github.com/szhekpisov/diffyml/pull/57))

## [1.5.7] - 2026-03-12

### Fixed

- Remove normalizeFilePath stderr warning (#56) ([#56](https://github.com/szhekpisov/diffyml/pull/56))
- Display document-level diffs as YAML documents (#55) ([#55](https://github.com/szhekpisov/diffyml/pull/55))

### Documentation

- Fact-check and fix comparison table in README (#54) ([#54](https://github.com/szhekpisov/diffyml/pull/54))

## [1.5.6] - 2026-03-12

### Fixed

- Suppress relative-path warning for /dev/ paths (#52) ([#52](https://github.com/szhekpisov/diffyml/pull/52))

## [1.5.5] - 2026-03-10

### Added

- Add Homebrew tap auto-update to GoReleaser (#50) ([#50](https://github.com/szhekpisov/diffyml/pull/50))

## [1.5.4] - 2026-03-10

### Added

- Add SBOM generation and cosign signed releases (#49) ([#49](https://github.com/szhekpisov/diffyml/pull/49))

### Documentation

- Strengthen README with comparison table, reordered sections, and accurate metrics (#48) ([#48](https://github.com/szhekpisov/diffyml/pull/48))

## [1.5.3] - 2026-03-09

### Added

- Replace LCS with Myers diff algorithm for line-level diffs (#45) ([#45](https://github.com/szhekpisov/diffyml/pull/45))

### Fixed

- Render removed diffs before added diffs for consistent output ordering (#44) ([#44](https://github.com/szhekpisov/diffyml/pull/44))

## [1.5.1] - 2026-03-02

### Fixed

- Render nested structured types as YAML instead of Go's %v representation (#36) ([#36](https://github.com/szhekpisov/diffyml/pull/36))
- Resolve fixture naming collisions for duplicated test numbers (#37) ([#37](https://github.com/szhekpisov/diffyml/pull/37))

## [1.5.0] - 2026-03-01

### Added

- Use dyff-style colon notation for multi-document paths (#29) ([#29](https://github.com/szhekpisov/diffyml/pull/29))
- Implement x509 certificate inspection for PEM certs in YAML diffs (#33) ([#33](https://github.com/szhekpisov/diffyml/pull/33))
- Implement rename detection for Kubernetes resources in multi-document YAML diffs (#34) ([#34](https://github.com/szhekpisov/diffyml/pull/34))

## [1.4.0] - 2026-02-27

### Added

- Add --ignore-api-version flag for K8s apiVersion-agnostic matching (#25) ([#25](https://github.com/szhekpisov/diffyml/pull/25))

### Fixed

- Kubectl KUBECTL_EXTERNAL_DIFF compatibility (#27) ([#27](https://github.com/szhekpisov/diffyml/pull/27))

### Documentation

- Update README with --ignore-api-version flag documentation (#26) ([#26](https://github.com/szhekpisov/diffyml/pull/26))

## [1.3.0] - 2026-02-26

### Added

- Add AI-powered summary of YAML differences via --summary flag (#22) ([#22](https://github.com/szhekpisov/diffyml/pull/22))

### Fixed

- Exclude self-referential changelog entries from release notes (#23) ([#23](https://github.com/szhekpisov/diffyml/pull/23))

## [1.2.0] - 2026-02-26

### Added

- Support metadata.generateName for K8s resource matching (#21) ([#21](https://github.com/szhekpisov/diffyml/pull/21))

## [1.1.0] - 2026-02-26

### Added

- Kubernetes compatibility (#7) ([#7](https://github.com/szhekpisov/diffyml/pull/7))
- Update formatting output for CI platform compliance (#9) ([#9](https://github.com/szhekpisov/diffyml/pull/9))
- GitLab Code Quality output compliance (#10) ([#10](https://github.com/szhekpisov/diffyml/pull/10))
- File-aware GitHub Actions annotations with directory mode and limits (#11) ([#11](https://github.com/szhekpisov/diffyml/pull/11))
- Add OpenSSF Scorecard workflow and badge (#12) ([#12](https://github.com/szhekpisov/diffyml/pull/12))
- Add automatic CHANGELOG.md generation with git-cliff (#14) ([#14](https://github.com/szhekpisov/diffyml/pull/14))
- Rename --color values from on/off to GNU-standard always/never (#13) ([#13](https://github.com/szhekpisov/diffyml/pull/13))

### Fixed

- Align --help flag descriptions to consistent column (#8) ([#8](https://github.com/szhekpisov/diffyml/pull/8))

## [1.0.0] - 2026-02-23

### Added

- Add automated release pipeline and Homebrew distribution (#5) ([#5](https://github.com/szhekpisov/diffyml/pull/5))

### Documentation

- Reorder sections and list all golangci-lint linters
- Add GOPATH/bin PATH hint to Go Install section (#3) ([#3](https://github.com/szhekpisov/diffyml/pull/3))
- Cleanup (#6) ([#6](https://github.com/szhekpisov/diffyml/pull/6))

[1.5.8]: https://github.com/szhekpisov/diffyml/compare/v1.5.7...v1.5.8
[1.5.7]: https://github.com/szhekpisov/diffyml/compare/v1.5.6...v1.5.7
[1.5.6]: https://github.com/szhekpisov/diffyml/compare/v1.5.5...v1.5.6
[1.5.5]: https://github.com/szhekpisov/diffyml/compare/v1.5.4...v1.5.5
[1.5.4]: https://github.com/szhekpisov/diffyml/compare/v1.5.3...v1.5.4
[1.5.3]: https://github.com/szhekpisov/diffyml/compare/v1.5.2...v1.5.3
[1.5.1]: https://github.com/szhekpisov/diffyml/compare/v1.5.0...v1.5.1
[1.5.0]: https://github.com/szhekpisov/diffyml/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/szhekpisov/diffyml/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/szhekpisov/diffyml/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/szhekpisov/diffyml/compare/v1.1.1...v1.2.0
[1.1.0]: https://github.com/szhekpisov/diffyml/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/szhekpisov/diffyml/releases/tag/v1.0.0

