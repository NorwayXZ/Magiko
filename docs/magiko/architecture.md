# Magiko integration architecture

Upstream base: `yetone/magpie@588f87a126834c2ed1c6348a35c6f031924c1855`.

## Identity

`internal/brand` has mutually exclusive `magiko` / `!magiko` implementations.
Upstream builds retain their identity so existing path, port and wiring tests
continue to express upstream contracts. Fork builds use `Magiko`, `magiko`,
`com.norwayxz.magiko`, port 3525 and a separate single-instance ID.

`appdir` configuration/cache/system-cache paths, macOS login startup label,
macOS daemon label, CLI command name and window/tray branding derive from those constants.
`versionedPage` brands HTML before hashing it and removes the fork page/script
from non-fork responses. Native app resources use a separate geometric M icon.
The v1 bundle intentionally registers no URL scheme: upstream import-link
parsers are still Magpie-specific. Existing provider/table/token identifiers
remain upstream-compatible, not globally renamed.
Portable mode still follows upstream `data`/`.portable` conventions and is not
audited for side-by-side portable forks. Use the normal installed configuration
for isolation; do not point both forks at one portable data folder.

Fork `update.Released` is false and `LatestIn` returns a manual-update notice
before making any network request. This prevents an upstream release overwriting
the fork. `MAGPIE_*` environment variables remain upstream-compatible for now;
use them only intentionally, especially `MAGPIE_ADDR` which overrides the fork port.

## Request preferences

`internal/magiko` stores `magiko-preferences.json` inside this application's
config. Defaults are disabled, empty text. Only valid UTF-8 <=12,000 bytes is
accepted. An enabled empty preference is invalid. Saves are serialized/atomic;
existing settings in other files are not rewritten.

The gateway calls `magiko.Apply` at the common `gatewayHooks` entry, **before**
its no-plugin early return. Both `serveAgent` and the native Codex subscription
handler call this function. No Bun dependency is needed for this feature.

Only `gatewayOrigin == local` is eligible: gateway-key callers, forwarded and
non-loopback clients retain the body they sent. The preference is ordinary
context visible to the upstream service, not a claim of elevated authority.

Protocol behavior:
- Responses: append to a string `instructions` (missing/null treated as empty).
- Chat: prepend a separate system message, preserving all existing messages.
- Anthropic: append to a system string or append a text block to a system array,
  retaining existing cache-control fields.
- Unsupported protocol/shape, malformed body, invalid configuration: unchanged.

Untouched fields are retained as `json.RawMessage`, including unknown fields,
model, stream and tools. An exact matching preference block is not duplicated.
There is no request retry, response mutation, account flag or authorization
trigger. A storage read error leaves the request unchanged and is surfaced
when the workspace reads settings. Preferences affect all eligible local
requests, not selected sessions, and are not a per-session memory feature.

## Skills and Library

Two extracted MIT skill folders are embedded in `internal/magiko/skills`.
`StageSkills` extracts to `magiko-pack-v1` under Magiko's config on explicit
request. Existing edited files and symlink entries cause an error rather than
being overwritten. Original skill licences remain beside the text/scripts.

The POST import route probes this fixed local source and delegates to the
existing Library with **empty Agents**. No new deployment or backup mechanism
is invented: assignment, disabling, removal, conflict handling and backup
remain the Library's responsibility. Its change operation can resync any items
the user already assigned; import is not a dry-run of the whole Library.
Installing skills does not execute their scripts. Local tool dependencies are
optional and separately installed by the user.

## UI and API

`plugins/magiko` registers an essential fork-only workspace in the existing
plugin kernel. Embedded `plugins/magiko/client.js` adds the page, while the
shell supplies navigation and dirty-edit confirmation.

- `GET /api/magiko`: current preferences, fixed skill names, own config path.
- `POST /api/magiko/preferences`: bounded strict JSON, validation, atomic save.
- `POST /api/magiko/skills`: explicit extraction/import into Library.

Routes are registered within the existing GUI handler and retain its desktop
and web authentication and same-origin protections. No new public listener is
created. Errors are shown with textContent; user text is never injected as HTML.

## Compatibility and limits

The first binary is native macOS ARM64. The new context module is portable Go,
but other OS bundle/startup branding has not been audited for distribution.
No automatic live configuration migration is performed. Subscription routing is
inherited and fixture-tested; actual sign-in and account billing are not tested.
The frontend currently has Simplified/Traditional Chinese and English text;
Japanese/German use the English fallback. Runtime language changes require
reloading the workspace for its labels to follow. This is a preview, not an
upstream-compatible release/update source.
