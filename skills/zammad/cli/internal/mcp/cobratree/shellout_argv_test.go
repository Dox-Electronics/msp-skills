// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored (handfixes.json: mcp-argv-value-joined): every value-carrying
// branch of cliArgsFromMCP must emit exactly ONE argv element, including the
// default branch that stringifies a nested object. The generated
// TestCliArgsFromMCP_ValueCannotSmuggleBlockedFlag proves the string and bool
// shapes; this pins the float, list and object shapes so no branch can ever
// split a value back into a token pflag would read as a second flag.

package cobratree

import "testing"

func TestCliArgsFromMCP_EveryValueBranchIsJoined(t *testing.T) {
	for _, tc := range []struct {
		name string
		args map[string]any
		want string
	}{
		{"float", map[string]any{"limit": 5.0}, "--limit=5"},
		{"list", map[string]any{"ids": []any{"a", "b"}}, "--ids=a,b"},
		{"object", map[string]any{"filter": map[string]any{"a": "1"}}, "--filter=map[a:1]"},
	} {
		got := cliArgsFromMCP(tc.args, blockedRootFlags)
		if len(got) != 1 || got[0] != tc.want {
			t.Fatalf("%s: got %#v, want exactly [%q]", tc.name, got, tc.want)
		}
	}
}
