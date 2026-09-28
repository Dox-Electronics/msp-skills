package cli

// Hand-authored (not generated): see skills/huntress/handfixes.json entry
// "mcp-store-path-matches-cli".

import "huntress-pp-cli/internal/cliutil"

// MCPStorePath resolves the local SQLite store exactly the way the CLI does,
// so the MCP search/sql tools read the file the CLI `sync` wrote.
//
// Press 4.32 scopes a FRESH store by a hash of the configured credential
// (data-<hash>.db) while the generated MCP resolver still hard-codes data.db.
// On a new install every sync - including one the MCP server itself shells out
// to - lands in the scoped file and search/sql then report "no store".
// Routing both through defaultDBPathInDir keeps them on one file: an existing
// data.db still wins, and a fresh install uses the scoped name on both sides.
func MCPStorePath() (string, error) {
	dir, err := cliutil.DataDir()
	if err != nil {
		return "", err
	}
	configureDefaultDBScope("")
	return defaultDBPathInDir(dir), nil
}
