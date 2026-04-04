# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.5.20] - 2026-04-04

### Added

- Highlight partial changes within scalar values (#93) ([#93](https://github.com/szhekpisov/diffyml/pull/93))

## [1.5.19] - 2026-04-03

### Added

- Recurse into subdirectories for directory comparison (#91) ([#91](https://github.com/szhekpisov/diffyml/pull/91))

## [1.5.18] - 2026-03-31

### Fixed

- Show K8s document name in detailed output for single-doc files (#89) ([#89](https://github.com/szhekpisov/diffyml/pull/89))

## [1.5.17] - 2026-03-30

### Fixed

- Upload SLSA provenance to draft release before publishing (#87) ([#87](https://github.com/szhekpisov/diffyml/pull/87))

## [1.5.16] - 2026-03-30

### Fixed

- Use draft release to allow SLSA provenance upload (#86) ([#86](https://github.com/szhekpisov/diffyml/pull/86))

## [1.5.15] - 2026-03-30

### Fixed

- Produce precise nested diffs with --ignore-order-changes (#85) ([#85](https://github.com/szhekpisov/diffyml/pull/85))

## [1.5.14] - 2026-03-27

### Added

- Add YAML config file support (.diffyml.yml) (#77) ([#77](https://github.com/szhekpisov/diffyml/pull/77))
- Add configurable colors for accessibility (#80) ([#80](https://github.com/szhekpisov/diffyml/pull/80))

## [1.5.13] - 2026-03-21

### Fixed

- Use line-by-line diff for multiline strings with whitespace-only changes (#73) ([#73](https://github.com/szhekpisov/diffyml/pull/73))
- Use line-by-line diff for multiline strings with whitespace-only changes (#74) ([#74](https://github.com/szhekpisov/diffyml/pull/74))

## [1.5.12] - 2026-03-20

### Added

- Add JSON Patch output format (--output json-patch) (#68) ([#68](https://github.com/szhekpisov/diffyml/pull/68))
- Add --format-strings flag to canonicalize embedded JSON before comparison (#71) ([#71](https://github.com/szhekpisov/diffyml/pull/71))
- Add per-element YAML syntax coloring in TrueColor mode (#72) ([#72](https://github.com/szhekpisov/diffyml/pull/72))

## [1.5.11] - 2026-03-19

### Added

- Show K8s resource identifier in multi-document diff output (#70) ([#70](https://github.com/szhekpisov/diffyml/pull/70))

### Documentation

- Mention generateName support in Kubernetes resource matching (#67) ([#67](https://github.com/szhekpisov/diffyml/pull/67))

## [1.5.10] - 2026-03-16

### Added

- Add JSON output format (--output json) (#63) ([#63](https://github.com/szhekpisov/diffyml/pull/63))
- Support GIT_EXTERNAL_DIFF calling convention (#66) ([#66](https://github.com/szhekpisov/diffyml/pull/66))

## [1.5.9] - 2026-03-15

### Fixed

- Bracket-quote YAML map keys containing dots (#62) ([#62](https://github.com/szhekpisov/diffyml/pull/62))

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

[1.5.20]: https://github.com/szhekpisov/diffyml/compare/v1.5.19...v1.5.20
[1.5.19]: https://github.com/szhekpisov/diffyml/compare/v1.5.18...v1.5.19
[1.5.18]: https://github.com/szhekpisov/diffyml/compare/v1.5.17...v1.5.18
[1.5.17]: https://github.com/szhekpisov/diffyml/compare/v1.5.16...v1.5.17
[1.5.16]: https://github.com/szhekpisov/diffyml/compare/v1.5.15...v1.5.16
[1.5.15]: https://github.com/szhekpisov/diffyml/compare/v1.5.14...v1.5.15
[1.5.14]: https://github.com/szhekpisov/diffyml/compare/v1.5.13...v1.5.14
[1.5.13]: https://github.com/szhekpisov/diffyml/compare/v1.5.12...v1.5.13
[1.5.12]: https://github.com/szhekpisov/diffyml/compare/v1.5.11...v1.5.12
[1.5.11]: https://github.com/szhekpisov/diffyml/compare/v1.5.10...v1.5.11
[1.5.10]: https://github.com/szhekpisov/diffyml/compare/v1.5.9...v1.5.10
[1.5.9]: https://github.com/szhekpisov/diffyml/compare/v1.5.8...v1.5.9
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

