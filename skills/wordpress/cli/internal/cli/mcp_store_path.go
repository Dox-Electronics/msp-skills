// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import "wordpress-pp-cli/internal/cliutil"

// MCPStoreDBPath resolves the local store the way the CLI's sync does:
// credential-scoped data-<hash>.db, with the legacy unscoped data.db kept when
// it already exists. The MCP search/sql tools used a hard-coded data.db, so on
// a fresh install a successful `sync` (run through the MCP shell-out or the
// CLI) wrote data-<hash>.db and search/sql then reported "no local data store".
// Hand-fix mcp-store-path-matches-sync (skills/wordpress/handfixes.json).
func MCPStoreDBPath() (string, error) {
	dir, err := cliutil.DataDir()
	if err != nil {
		return "", err
	}
	configureDefaultDBScope("")
	return defaultDBPathInDir(dir), nil
}
