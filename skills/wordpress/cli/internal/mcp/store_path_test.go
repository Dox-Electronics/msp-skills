// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.

// Hand-fix mcp-store-path-matches-sync (skills/wordpress/handfixes.json):
// MCP search/sql must open the same store file sync writes.
package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wordpress-pp-cli/internal/cliutil"
	"wordpress-pp-cli/internal/cliutil/testenv"
)

func isolateStoreHome(t *testing.T) string {
	t.Helper()
	restore, err := cliutil.SetHomeOverride("")
	if err != nil {
		t.Fatalf("reset home override: %v", err)
	}
	t.Cleanup(restore)
	home := testenv.Isolate(t, cliutil.ConfigDir, cliutil.DataDir, cliutil.StateDir, cliutil.CacheDir)
	t.Setenv("WORDPRESS_HOME", home)
	t.Setenv("WORDPRESS_CONFIG", "")
	return home
}

func TestMCPStorePathFollowsCredentialScopedSyncPath(t *testing.T) {
	isolateStoreHome(t)
	t.Setenv("WORDPRESS_BASIC_AUTH", "operator:app-password")

	got, err := mcpDBPath()
	if err != nil {
		t.Fatalf("mcpDBPath: %v", err)
	}
	base := filepath.Base(got)
	if base == "data.db" || !strings.HasPrefix(base, "data-") || !strings.HasSuffix(base, ".db") {
		t.Fatalf("fresh install with a credential: mcpDBPath = %s, want the credential-scoped data-<hash>.db sync writes", got)
	}
}

func TestMCPStorePathKeepsLegacyUnscopedStore(t *testing.T) {
	isolateStoreHome(t)
	t.Setenv("WORDPRESS_BASIC_AUTH", "operator:app-password")
	dir, err := cliutil.DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(dir, "data.db")
	if err := os.WriteFile(legacy, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := mcpDBPath()
	if err != nil {
		t.Fatalf("mcpDBPath: %v", err)
	}
	if filepath.Base(got) != "data.db" {
		t.Fatalf("existing legacy store: mcpDBPath = %s, want data.db", got)
	}
}
