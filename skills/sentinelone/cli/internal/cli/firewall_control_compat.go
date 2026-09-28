// Copyright 2026 Servosity Inc. and msp-skills contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// firewallControlLeafArgs guards the three firewall-control leaf endpoints
// (copy-rules, move-rules, set-location) against stray positionals.
//
// Up to v0.1.x (cli-printing-press 4.24) these three names were GROUPS whose
// only child was `by-category`, and the leaf endpoints were unreachable. From
// the 4.32.5 reprint on, the names are the leaf POST endpoints and the
// by-category operations live under `item-<name> by-category`. The generated
// leaf commands ignore positionals, so a script still typing the old spelling
// `firewall-control move-rules by-category ...` would silently POST its body to
// the wrong mutation (/firewall-control/move-rules). This rejects any
// positional before a request is built and names the new path when the old
// spelling is used. Hand-wired: handfixes.json firewall-control-renamed-by-category.
func firewallControlLeafArgs(name string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		if args[0] == "by-category" {
			return fmt.Errorf("`firewall-control %s by-category` moved to `firewall-control item-%s by-category` (the reprint to cli-printing-press 4.32.5 made `firewall-control %s` the leaf endpoint); nothing was sent", name, name, name)
		}
		return fmt.Errorf("`firewall-control %s` takes no positional arguments, got %q; nothing was sent", name, strings.Join(args, " "))
	}
}
