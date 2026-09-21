// gen-cli-ref renders a Hugo Markdown page describing every diffyml CLI flag.
//
// Source of truth is pkg/diffyml/cli.FlagDocs(); a unit test in that package
// guarantees parity between FlagDocs and the registered flag.FlagSet.
//
// Usage:
//
//	go run ./doc/gen-cli-ref > docs/content/docs/reference.md
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/szhekpisov/diffyml/pkg/diffyml/cli"
)

func main() {
	if err := render(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func render(w io.Writer) error {
	if _, err := io.WriteString(w, frontMatter); err != nil {
		return err
	}

	docs := cli.FlagDocs()
	groups := groupByCategory(docs)

	for _, cat := range categoryOrder {
		entries, ok := groups[cat]
		if !ok {
			continue
		}
		fmt.Fprintf(w, "## %s\n\n", cat)
		fmt.Fprintln(w, "| Flag | Type | Default | Description |")
		fmt.Fprintln(w, "|------|------|---------|-------------|")
		for _, d := range entries {
			fmt.Fprintf(w, "| %s | `%s` | %s | %s |\n",
				renderFlagName(d), d.Type, renderDefault(d.Default), d.Usage)
		}
		fmt.Fprintln(w)
	}

	if _, err := io.WriteString(w, versionTrailer); err != nil {
		return err
	}
	return nil
}

func groupByCategory(docs []cli.FlagDoc) map[string][]cli.FlagDoc {
	out := make(map[string][]cli.FlagDoc)
	for _, d := range docs {
		out[d.Category] = append(out[d.Category], d)
	}
	return out
}

// categoryOrder controls the section order in the rendered page.
var categoryOrder = []string{
	"Output",
	"Comparison",
	"Filtering",
	"Neat",
	"Masking",
	"Display",
	"Chroot",
	"AI Summary",
	"Configuration",
	"Other",
}

func renderFlagName(d cli.FlagDoc) string {
	var parts []string
	if d.Short != "" {
		parts = append(parts, "`-"+d.Short+"`")
	}
	parts = append(parts, "`--"+d.Long+"`")
	return strings.Join(parts, ", ")
}

func renderDefault(def string) string {
	if def == "" {
		return "—"
	}
	return "`" + def + "`"
}

const frontMatter = `---
title: "Reference"
description: "Complete reference for every diffyml command-line flag, auto-generated from the Go source."
weight: 90
---

This page is auto-generated from ` + "`pkg/diffyml/cli/flag_metadata.go`" + ` by ` + "`make docs-gen`" + `.
Do not edit by hand — your changes will be overwritten on the next build.

` + "```" + `
diffyml [flags] <from> <to>
` + "```" + `

`

// versionTrailer explains the one behavior the flag tables cannot: --help and
// --version are ordinary flags (listed under "Other" above, so no table here
// would repeat them) that short-circuit argument parsing.
const versionTrailer = `## Help and version

` + "`-h`/`--help`" + ` and ` + "`-V`/`--version`" + ` are parsed like any other flag, but they
short-circuit before the ` + "`<from> <to>`" + ` arguments are required and before the
config file is read — so they work with no arguments at all, and keep working
when a config file is missing or malformed.
`
