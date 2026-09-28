// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"os"
	"time"

	"datto-bcdr-pp-cli/internal/cliutil"
	"datto-bcdr-pp-cli/internal/store"
)

// nvOpenStore opens the local mirror for the read-only novel commands. When the
// mirror already exists it opens read-only — a read-only handle takes no write
// lock, so several novel commands can run concurrently against the same mirror
// without the SQLITE_BUSY the scorecard's parallel sample probe surfaced. When
// the mirror is absent it is created and migrated (read-write). A short retry
// resolves the first-run race where several novels try to create the DB at
// once: the loser re-checks, finds the winner's file, and opens it read-only.
//
// The returned store is always non-nil on a nil error, so callers keep their
// existing `defer db.Close()` and `db.DB()` usage unchanged.
func nvOpenStore(ctx context.Context, dbPath string) (*store.Store, error) {
	if dbPath == "" {
		dbPath = defaultDBPath("datto-bcdr-cli")
	}
	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
		if _, statErr := os.Stat(dbPath); statErr == nil {
			s, err := store.OpenReadOnly(dbPath)
			if err == nil {
				return s, nil
			}
			lastErr = err
		} else {
			s, err := store.OpenWithContext(ctx, dbPath)
			if err == nil {
				return s, nil
			}
			lastErr = err
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		time.Sleep(time.Duration(20*(attempt+1)) * time.Millisecond)
	}
	return nil, lastErr
}

// MCPStoreDBPath resolves the local mirror path the same way `sync` does, for
// the MCP server's in-process search/sql tools. Press 4.32 scopes the mirror
// to the credential (data-<sha256[:12]>.db) in the CLI, but the generated
// mcpDBPath hardcoded data.db, so after a fresh sync the MCP tools reported
// "No local data store found". Loading the default config and applying the
// same scope keeps both surfaces on one resolver, including the legacy
// unscoped data.db fallback. Hand-wired: reprint-survival ledger
// mcp-store-path-matches-sync.
func MCPStoreDBPath() (string, error) {
	dir, err := cliutil.DataDir()
	if err != nil {
		return "", err
	}
	configureDefaultDBScope("")
	return defaultDBPathInDir(dir), nil
}
