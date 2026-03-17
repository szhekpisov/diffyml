// Package diffyml compares YAML documents structurally and reports the
// differences.  It understands YAML semantics (ordered maps, multi-document
// streams, anchors/aliases) and optionally detects Kubernetes resources,
// matching them by apiVersion + kind + metadata.name instead of document
// position.
//
// # Comparing YAML
//
// The central entry point is [Compare].  It accepts two byte slices of YAML
// content (single- or multi-document) and returns a slice of [Difference]
// values.  Each Difference carries the dot-notation path, the type of change
// ([DiffAdded], [DiffRemoved], [DiffModified], [DiffOrderChanged]) and the
// old/new values.
//
//	diffs, err := diffyml.Compare(oldYAML, newYAML, nil)
//	for _, d := range diffs {
//	    fmt.Printf("%d %s\n", d.Type, d.Path)
//	}
//
// Pass an [Options] struct to control comparison behaviour: ignore list order,
// ignore whitespace, enable Kubernetes-aware matching, detect renames, navigate
// to a subtree via chroot, and more.
//
// # Loading content
//
// [LoadContent] reads YAML from a local file path or an HTTP/HTTPS URL.
// [IsRemoteSource] checks whether a string looks like a URL.
//
//	from, _ := diffyml.LoadContent("old.yaml")
//	to, _   := diffyml.LoadContent("https://example.com/new.yaml")
//	diffs, _ := diffyml.Compare(from, to, nil)
//
// # Formatting output
//
// Eight built-in formatters render differences for different audiences:
//
//   - [DetailedFormatter] — full human-readable output with inline diffs
//   - [CompactFormatter] — one line per change
//   - [BriefFormatter] — summary counts only
//   - [GitHubFormatter] — GitHub Actions workflow annotations
//   - [GitLabFormatter] — GitLab CI Code Quality JSON
//   - [GiteaFormatter] — Gitea CI (GitHub-compatible annotations)
//   - [JSONFormatter] — machine-readable JSON with typed values
//   - [JSONPatchFormatter] — RFC 6902 JSON Patch operations
//
// All formatters implement the [Formatter] interface.  Use [FormatterByName]
// to obtain a formatter by string name, and [DefaultFormatOptions] for
// sensible defaults.
//
//	f, _ := diffyml.FormatterByName("compact")
//	fmt.Print(f.Format(diffs, diffyml.DefaultFormatOptions()))
//
// # Filtering
//
// [FilterDiffs] selects or excludes differences by exact path prefix.
// [FilterDiffsWithRegexp] adds regular-expression support.  Both accept a
// [FilterOptions] struct.
//
// # Kubernetes awareness
//
// When [Options].DetectKubernetes is set to true, multi-document YAML
// files are matched by Kubernetes resource identity rather than position.
// The CLI enables this by default; library callers must opt in explicitly.  [IsKubernetesResource] checks whether a parsed document looks
// like a Kubernetes resource, and [K8sResourceIdentifier] returns its
// canonical identifier string.
//
// # Directory comparison
//
// [IsDirectory], [DiscoverFiles], and [BuildFilePairPlan] support comparing
// two directories of YAML files, making diffyml a drop-in
// KUBECTL_EXTERNAL_DIFF provider.
//
// # Parsing helpers
//
// [OrderedMap] preserves YAML key order during parsing.
// [ParseWithOrder] parses YAML content into documents using [OrderedMap].
package diffyml
