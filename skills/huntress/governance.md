# huntress skill - governance and safety model

> Unofficial. Community-built skill for the Huntress API. Not affiliated with,
> endorsed by, or sponsored by the vendor.
> This page tells an MSP owner exactly what the huntress skill can touch and how to
> scope it, so you can decide what to let an AI agent do.

## What it authenticates as

The skill drives the `huntress-cli` binary (and `huntress-mcp`),
authenticating with `HUNTRESS_API_KEY`, `HUNTRESS_API_SECRET`. Credentials come from the environment, or from
`huntress-cli auth set-credentials`, which saves them to an owner-only (0600) credentials file in the CLI's
data directory (next to the local store). They are never logged and never sent anywhere except the Huntress API.

## Default-safe behavior

- **`--dry-run` is opt-in - use it.** Mutating commands send immediately unless you pass `--dry-run` first to preview the request without sending. Make your agent's policy: preview, show the exact command, get approval, then run the write.
- **Read commands are always safe to run** (reports, rollups, search); they cannot
  change anything.
- **Agent mode is explicit.** `--agent` produces JSON for scripting but does not
  add any write gating - the preview-then-approve policy above still applies. See
  AGENTS.md.

## Permission tiers

The safe default for an autonomous agent is **read plus planned (dry-run) writes**;
require a human for anything below the line.

| Tier | What it does | Examples | Recommended agent policy |
| --- | --- | --- | --- |
| **Read** | Reports, rollups, search. No change. | the cross-entity views and any non-mutating command | Allow |
| **Write (routine)** | Day-to-day mutations. | `accounts memberships update-parameters`, `accounts organizations update-parameters`, `accounts update-parameters`, `memberships update-parameters`, `organizations update-parameters`, `reseller subscription-update-parameters`, `unwanted-access-rules update-parameters` | Preview with `--dry-run`, then an approved write (where a command documents its own confirm gate, use it too) |
| **Credential / security** | Touches tokens, keys, MFA. | (none detected) | Human-in-the-loop only |
| **Destructive** | Irreversible data or config loss. | `accounts delete-v1-id`, `accounts memberships delete-v1-accounts-account-id-id`, `accounts organizations delete-v1-accounts-account-id-id`, `memberships delete-v1-id`, `organizations delete-v1-id`, `unwanted-access-rules delete-v1-id` | Human-in-the-loop only, explicit confirmation |
| **Admin** | Back-office administration. | (none detected) | Operator-only, not for agents |

## Local learning loop (engine 4.32 and later)

The `teach`, `recall`, `learnings`, `playbook`, `teach-pattern`, `teach-lookup`,
and `teach-playbook` commands belong to the CLI's self-learning loop. They read
and write ONLY the local SQLite store (`data.db` under the data directory);
nothing in this loop sends data to the Huntress API or anywhere else. Treat them
as Read tier for the vendor account, with two local caveats: `learnings purge`
and `learnings forget` delete locally taught rows (recoverable only by teaching
again), and the loop journals every invocation locally. Disable it per call with
`--no-learn` or for a session with `HUNTRESS_NO_LEARN=true` when you want
deterministic agent runs. `export` writes a local file and, over MCP, cannot be
pointed at an arbitrary path (the filesystem-destination gate refuses `--output`).

## How to lock it down

- **Scope the credential** to only what your workflow needs. A read/report workflow
  does not need a credential that can run the Destructive or Credential tiers.
- **Keep autonomous agents to Read + previewed writes.** Have a human approve the
  actual write for Write tier and above - the gate lives in your agent's policy,
  not in the binary's defaults.
- **Never let an agent run Credential, Destructive, or Admin tier commands
  unattended.** Treat them like a production database drop: human, reviewed, logged.
- **Rotate the credential if it is ever exposed** (for example after bridging the
  MCP server to a public endpoint for ChatGPT - see mcp-install.md).

## Why an MSP owner can be comfortable

The full source of the CLI and MCP server is in this repository under
[`cli/`](./cli) (Apache-2.0). You supply the credential, the binary uses it against
the Huntress API, and you can read every line of how it does so. The skill is
read-first, plan-by-default, and scoped to your own account.
