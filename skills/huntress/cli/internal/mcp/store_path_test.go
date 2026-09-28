package mcp

// Hand-authored (not generated): see skills/huntress/handfixes.json entry
// "mcp-store-path-matches-cli".

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A fresh install with a credential configured: the CLI `sync` writes the
// credential-scoped data-<hash>.db, so the MCP search/sql tools must resolve
// that same file rather than a data.db nothing ever wrote.
func TestMCPStorePathFollowsCredentialScopedCLIStore(t *testing.T) {
	resetMCPPathEnv(t)
	t.Setenv("HUNTRESS_API_KEY", "test-key")
	t.Setenv("HUNTRESS_API_SECRET", "test-secret")

	got, err := mcpDBPath()
	if err != nil {
		t.Fatalf("mcpDBPath() error = %v", err)
	}
	base := filepath.Base(got)
	if !strings.HasPrefix(base, "data-") || !strings.HasSuffix(base, ".db") {
		t.Fatalf("fresh credentialed install: MCP store = %q, want the CLI's scoped data-<hash>.db", got)
	}

	// An existing legacy data.db still wins on both sides.
	legacy := filepath.Join(filepath.Dir(got), "data.db")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = mcpDBPath()
	if err != nil {
		t.Fatalf("mcpDBPath() error = %v", err)
	}
	if got != legacy {
		t.Fatalf("legacy install: MCP store = %q, want existing %q", got, legacy)
	}
}
