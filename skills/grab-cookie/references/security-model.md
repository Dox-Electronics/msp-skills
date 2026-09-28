# Security model - secrets into the OS credential store, never into context

Vendored from the sibling `connect-tool` Skill's `references/security-model.md`
(Servosity/msp-skills, Apache-2.0) and adapted to what grab-cookie actually ships.
It is here so the file `scripts/credstore.py` cites resolves after a plugin
install, where `../connect-tool/` does not exist. NOTICE lists the divergences.

## The one structural rule

The model authors the *profile* (which header, which store name, which consumer
file, which verify call); it never sees the *value*. Every secret flows through a
shipped helper that consumes it *inside the process* and prints **only** a redacted
receipt: `len`, `sha256[:8]`, `last4`. That process boundary is the whole security
model. `scripts/credgrab.py --selfcheck` asserts the value never appears in stdout.

Structural supports for that boundary, all enforced in code:

- no `shell=True` anywhere, so a header value or a path is never parsed by `cmd.exe`
  or bash
- fixed error strings; a captured value is never interpolated into an exception
  message (`CredError`)
- captured output is never logged, and never returned up the stack
- `curlparse.py` walks the paste once, honouring quote state, so text inside one
  quoted value can never be re-parsed as another `-H` (see selfchecks 10-20)

## What this does NOT claim

Being precise here matters more than sounding strong.

- **"The complete secret never enters the agent's context."** That is the claim.
  The redacted receipt is deliberate: `last4` is how an operator matches a stored
  value against what a portal displays, and `sha256[:8]` lets two captures be
  compared without either being shown. Below 12 characters both are withheld: an
  exact length beside an unsalted digest is a verification oracle for a short
  secret. Lane C prints the same receipt; there is no receipt-free path.
- **The consumer is a separate trust boundary.** Once `wire` writes the credential
  into the consumer config file, the value lives in that file and in whatever tool
  reads it. If that tool prints it on `--debug` or ships telemetry, that is outside
  what grab-cookie can control.
- **Python strings cannot be reliably zeroized.** The Windows credential buffer is
  wiped best-effort after the write; the Python string it came from is not.
- **Deleting a file is not shredding it**, especially on an SSD. The capture file
  holds the whole DevTools request (every cookie for that origin, every auth
  header) until you delete it, and deleting it is hygiene, not erasure.

## Two lanes, in decision order

1. **Lane A - Copy as cURL paste (the normal path).** The user pastes the DevTools
   request into `captures/<profile>.curl.txt`; `seed` parses it, extracts each
   configured value, stores it, wires the consumer, and verifies. The value crosses
   one file on the user's own disk and one process. Nothing about it is printed.
2. **Lane C - user paste at a hidden prompt (fallback).** Any case where the value
   is easier to type than to capture, or the capture is unreliable. The user runs
   this in their OWN terminal (in Claude Code, prefix it with `!`):

   ```
   python "<this-skill-dir>/scripts/credstore.py" --store --service <SVC> --account <acct>
   ```

   The value never enters a file or the agent's context, and never argv on
   Windows; on macOS it still crosses the `security -w` argv residual below.
   Your only confirmation is the receipt, then the profile's verify call.

There is no Lane B here on purpose: grab-cookie drives no browser and reads no
DOM. connect-tool owns the displayed-value and OAuth lanes.

## Where the secret is stored

| Platform | Store | Secret in a command line? |
|---|---|---|
| macOS | Keychain, via `security add-generic-password` | **Yes, briefly.** See the residual below |
| Windows | Credential Manager, via `CredWriteW` through `ctypes` | **No.** Passed as a memory buffer |

The Windows path is the stronger of the two, and deliberately so. `ctypes` is used
instead of PowerShell `Add-Type` P/Invoke because Constrained Language Mode, which
AppLocker and WDAC commonly induce on a managed endpoint, blocks Win32 P/Invoke
from PowerShell entirely; and calling the API directly from Python means the secret
is never an argument to any process, so it never appears in a Sysmon Event 1 or a
4688 audit record.

Both stores are namespaced so grab-cookie's entries cannot collide with
connect-tool's for the same service and account, and are identifiable for revocation:

- Windows Credential Manager target: `credgrab/<service>/<account>`, written with
  `CRED_PERSIST_LOCAL_MACHINE` (not `ENTERPRISE`, which roams with a roaming
  profile). Generic blobs are capped at 2560 bytes; larger fails loudly.
- macOS Keychain service: `credgrab/<service>`, on every `security` call including
  delete (`credstore.py --selfcheck` asserts this).

After an administrative password reset, a stored credential may become unreadable.
Treat a read failure as revocation and re-seed.

## Residual surfaces (designed around)

- **The capture file.** A Copy-as-cURL paste is live credential material for the
  whole origin. `.gitignore` in this Skill covers `captures/`, `*.curl.txt` and
  `state.json`. Delete the capture once the seed verifies.
- **`security ... -w <value>` argv on macOS** is briefly visible in the *local*
  process table, on the user's own machine, not in model context. Accepted. On an
  endpoint with process-creation auditing this would be recorded, which is exactly
  why the Windows path does not use argv at all.
- **The consumer config file.** Written with mode 0600 by default on POSIX (a
  profile may set `wire.mode`; do not loosen it); on Windows the mode is
  best-effort and the file inherits its parent's NTFS ACLs. It is a cache
  regenerated from the store, not a second secret store.
- **Screenshots and page reads.** Never screenshot or `extract` a page rendering a
  session value. Proof is the redacted receipt, not a pixel.

## Verify by use, never by printing

A credential is unverified until the profile's verify command, a real read-only
authed call, succeeds (`credgrab.py verify`). Write that command so it fails on
an unauthenticated response rather than exiting zero with nothing; the helper
trusts its exit status. A seed whose verify fails is rolled back and reported as
failed: a stored credential that does not work fails later and somewhere else.
`doctor --all` re-probes every SEEDED profile (those recorded in `state.json`; a
Lane C store alone is not scheduled) and warns before a known expiry. To confirm
two captures match, compare `sha256[:8]`, never values.
