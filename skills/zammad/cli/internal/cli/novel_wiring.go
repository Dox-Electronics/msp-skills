// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: wires the hand-built commands the generator does not wire.
//
// root.go registers the root-level novels (agent-load, overdue, ticket, ...)
// itself and kb.go wires the `kb browse` / `kb get` / `kb search` novels
// (both come from research.json), but two kinds of hand-authored command
// have no generated registration and need a hook:
//
//   - `articles sync` (deep article sync into the local store) hangs under
//     the generated `articles` resource parent and is not a research.json
//     novel, so nothing generated adds it.
//   - `stale` / `orphans` / `load` were project-management workflow commands
//     the 4.28.0 press emitted and wired in root.go; the 4.32.5 profiler
//     classes Zammad as the "content" archetype and no longer emits them.
//     They are carried as hand files (pm_*.go) so the shipped surface and the
//     MCP tools of the same names survive the reprint.
//
// registerNovelCommand is the generator's own extension point: hooks run
// after every generated parent is attached, from init, without editing the
// generated root.go, so this wiring survives a force regeneration.

package cli

import "github.com/spf13/cobra"

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		if articles := findChildCommand(root, "articles"); articles != nil {
			addNovelCommandIfAbsent(articles, newNovelArticlesSyncCmd(flags))
		}
		addNovelCommandIfAbsent(root, newStaleCmd(flags))
		addNovelCommandIfAbsent(root, newOrphansCmd(flags))
		addNovelCommandIfAbsent(root, newLoadCmd(flags))
	})
}

// findChildCommand returns the direct child of parent named name, or nil.
func findChildCommand(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
