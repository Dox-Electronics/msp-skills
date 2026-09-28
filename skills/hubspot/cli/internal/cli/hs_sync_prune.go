// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.

package cli

// hsSyncPruneAllowed decides whether a sync may run deletion reconciliation.
//
// cli-printing-press 4.32 made `sync --full` prune local rows the API did not
// return on a complete walk. That is only sound when the walk enumerated the
// WHOLE collection. An operator-supplied query parameter (--param,
// --resource-param, --global-param) can narrow the collection - for example
// `--param archived=true` lists only archived contacts - and pruning against a
// filtered walk would delete every cached row outside the filter. The shipped
// 4.24 tree never pruned, so a filtered full sync keeps that behaviour: no
// prune. --no-prune still disables pruning everywhere.
// Hand-wired (handfixes.json: sync-prune-skips-filtered-walk).
func hsSyncPruneAllowed(full, noPrune bool, paramFlags, resourceParamFlags, globalParamFlags []string) bool {
	if !full || noPrune {
		return false
	}
	return len(paramFlags) == 0 && len(resourceParamFlags) == 0 && len(globalParamFlags) == 0
}
