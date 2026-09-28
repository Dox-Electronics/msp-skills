// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored regression guard for the msp-skills hand-fix ledger entry
// "mcp-argv-value-joined" (skills/wordpress/handfixes.json). Press >= 4.32.5
// emits the joined --flag=value argv natively and ships its own
// TestCliArgsFromMCP_ValueCannotSmuggleBlockedFlag; this file keeps the one
// case that test does not pin - the default (nested object) branch - so a
// future template that splits any single branch fails the build.
package cobratree

import "testing"

func TestCliArgsFromMCP_EveryValueBranchEmitsOneElement(t *testing.T) {
	for _, tc := range []struct {
		name string
		args map[string]any
		want string
	}{
		{"float", map[string]any{"limit": 5.0}, "--limit=5"},
		{"string", map[string]any{"query": "--deliver=webhook:https://x/"}, "--query=--deliver=webhook:https://x/"},
		{"list", map[string]any{"ids": []any{"a", "b"}}, "--ids=a,b"},
		{"object", map[string]any{"filter": map[string]any{"a": "1"}}, "--filter=map[a:1]"},
	} {
		got := cliArgsFromMCP(tc.args, blockedRootFlags)
		if len(got) != 1 || got[0] != tc.want {
			t.Fatalf("%s: got %#v, want exactly [%q]", tc.name, got, tc.want)
		}
	}
}
