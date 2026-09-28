// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.

package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"datto-bcdr-pp-cli/internal/cliutil"
	"datto-bcdr-pp-cli/internal/config"
)

// TestMCPStorePathFollowsCredentialScopedMirror pins the hand-fix recorded as
// mcp-store-path-matches-sync: sync writes data-<sha256(credential)[:12]>.db,
// so the MCP search/sql tools must resolve to that same file, while an install
// with no credential (or only a legacy unscoped mirror) still reads data.db.
func TestMCPStorePathFollowsCredentialScopedMirror(t *testing.T) {
	resetMCPPathEnv(t)
	dataDir, err := cliutil.DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}

	// No credential: the unscoped mirror.
	t.Setenv("DATTO_BCDR_PUBLIC_KEY", "")
	t.Setenv("DATTO_BCDR_SECRET_KEY", "")
	got, err := mcpDBPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dataDir, "data.db"); got != want {
		t.Fatalf("no credential: got %q, want %q", got, want)
	}

	// Credential set, fresh install: the scoped file sync will create.
	t.Setenv("DATTO_BCDR_PUBLIC_KEY", "pub-test")
	t.Setenv("DATTO_BCDR_SECRET_KEY", "sec-test")
	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	cred := cfg.StoreScopeCredential()
	if cred == "" {
		t.Fatal("expected a store scope credential from the env keys")
	}
	sum := sha256.Sum256([]byte(cred))
	scoped := filepath.Join(dataDir, "data-"+hex.EncodeToString(sum[:])[:12]+".db")
	got, err = mcpDBPath()
	if err != nil {
		t.Fatal(err)
	}
	if got != scoped {
		t.Fatalf("fresh scoped install: got %q, want %q (the hardcoded data.db is the defect)", got, scoped)
	}

	// Legacy unscoped mirror present and no scoped mirror yet: keep reading it.
	legacy := filepath.Join(dataDir, "data.db")
	if err := os.WriteFile(legacy, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ = mcpDBPath(); got != legacy {
		t.Fatalf("legacy fallback: got %q, want %q", got, legacy)
	}

	// Scoped mirror present: it wins over the legacy file.
	if err := os.WriteFile(scoped, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ = mcpDBPath(); got != scoped {
		t.Fatalf("scoped mirror present: got %q, want %q", got, scoped)
	}
}
