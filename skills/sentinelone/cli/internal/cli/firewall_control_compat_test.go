// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestFirewallControlOldBySpellingIsRefused pins both directions of the
// firewall-control rename guard: the pre-reprint spelling `<name> by-category`
// must be refused before any request is sent, while the bare leaf and the new
// `item-<name> by-category` path still resolve.
func TestFirewallControlOldBySpellingIsRefused(t *testing.T) {
	root := RootCmd()
	for _, name := range []string{"copy-rules", "move-rules", "set-location"} {
		cmd, rest, err := root.Find([]string{"firewall-control", name, "by-category"})
		if err != nil {
			t.Fatalf("%s: find: %v", name, err)
		}
		if cmd.Name() != name || len(rest) != 1 {
			t.Fatalf("%s: old spelling resolved to %q rest=%v", name, cmd.Name(), rest)
		}
		if cmd.Args == nil {
			t.Fatalf("%s: leaf has no positional guard", name)
		}
		err = cmd.Args(cmd, rest)
		if err == nil || !strings.Contains(err.Error(), "item-"+name+" by-category") {
			t.Fatalf("%s: old spelling not refused with the new path, got %v", name, err)
		}
		if err := cmd.Args(cmd, nil); err != nil {
			t.Fatalf("%s: bare leaf refused: %v", name, err)
		}
		item, _, err := root.Find([]string{"firewall-control", "item-" + name, "by-category"})
		if err != nil || item.Name() != "by-category" || item.Parent().Name() != "item-"+name {
			t.Fatalf("%s: new path does not resolve: %v %v", name, item, err)
		}
	}
}

// TestFirewallControlOldSpellingSendsNothing executes the old spelling against
// a live listener and proves no request reaches it.
func TestFirewallControlOldSpellingSendsNothing(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()
	t.Setenv("SENTINELONE_BASE_URL", srv.URL)
	t.Setenv("SENTINELONE_API_TOKEN", "test-token")
	t.Setenv("HOME", t.TempDir())

	root := RootCmd()
	root.SetArgs([]string{"firewall-control", "move-rules", "by-category", "--no-cache"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "item-move-rules by-category") {
		t.Fatalf("old spelling was not refused: %v", err)
	}
	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Fatalf("old spelling sent %d request(s)", n)
	}
}
