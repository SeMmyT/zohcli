---
phase: 01-foundation-authentication
plan: 02
subsystem: authentication
tags: [oauth2, keyring, encryption, token-cache, file-locking]

# Dependency graph
requires:
  - phase: 01-01
    provides: Config system with region mapping
provides:
  - OS keyring credential storage via 99designs/keyring
  - AES-256-GCM encrypted file fallback for WSL/headless
  - OAuth2 interactive flow (browser + localhost callback)
  - OAuth2 manual flow (paste redirect URL)
  - File-locked token cache implementing oauth2.TokenSource
  - Proactive token refresh (5-minute window)
affects: [01-03, 02-01, 03-01, 04-01, 05-01]

# Tech tracking
tech-stack:
  added: [99designs/keyring@1.2.2, gofrs/flock@0.13.0, golang.org/x/oauth2@0.35.0]
  patterns: [OS keyring with file fallback, file locking for token refresh, proactive token refresh, Zoho-specific OAuth2 (comma-separated scopes)]

key-files:
  created:
    - internal/secrets/keyring.go (OS keyring implementation)
    - internal/secrets/file.go (AES-256-GCM encrypted file storage)
    - internal/secrets/detect.go (Platform detection for WSL/headless)
    - internal/auth/scopes.go (OAuth2 scope definitions)
    - internal/auth/server.go (Localhost callback server)
    - internal/auth/flows.go (InteractiveLogin and ManualLogin)
    - internal/auth/token.go (File-locked token cache)
    - pkg/browser/browser.go (Cross-platform browser launcher)
  modified: []

key-decisions:
  - "99designs/keyring for OS keyring (supports macOS Keychain, Linux Secret Service, Windows Credential Manager)"
  - "AES-256-GCM encryption for file fallback (sha256 key derivation as simple v1 approach, noted scrypt/argon2 for future)"
  - "Machine-specific default encryption key when password empty (hostname + username hash, with warning)"
  - "WSL detection via /proc/version (case-insensitive match for 'microsoft' or 'WSL')"
  - "Headless detection via DISPLAY and WAYLAND_DISPLAY env vars on Linux"
  - "Zoho OAuth2 quirks: comma-separated scopes, access_type=offline for refresh token, prompt=consent"
  - "gofrs/flock for file locking (10-second timeout, prevents concurrent refresh stampede)"
  - "Proactive token refresh at 5-minute window (reduces auth errors during API calls)"
  - "Refresh token in secrets Store, access token in XDG cache (different persistence needs)"

patterns-established:
  - "Platform detection pattern: try keyring first, fall back to encrypted file with stderr warning"
  - "OAuth2 callback pattern: auto-select available port, auto-shutdown after one request"
  - "Token cache pattern: file lock → check cache → refresh if needed → update cache → release lock"
  - "Error handling: CLIError with ExitAuth for auth failures, hints point to 'zoh auth login'"

# Metrics
duration: 4 min
completed: 2026-02-14
---

# Phase 1 Plan 2: OAuth2 & Token Cache Summary

**OAuth2 authentication with browser/manual flows, OS keyring + encrypted file credential storage, and file-locked token cache with proactive refresh**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-14T17:19:28Z
- **Completed:** 2026-02-14T17:23:56Z
- **Tasks:** 2
- **Files created:** 8

## Accomplishments

- Secure credential storage with OS keyring (macOS, Linux, Windows) and encrypted file fallback for WSL/headless
- OAuth2 interactive flow opens browser, receives callback on localhost, exchanges code for tokens
- OAuth2 manual flow prints URL, reads pasted redirect, exchanges code (for SSH/headless environments)
- File-locked token cache prevents concurrent refresh stampede, implements oauth2.TokenSource interface
- Proactive token refresh within 5-minute expiry window reduces auth errors during API calls
- Zoho-specific OAuth2 handling (comma-separated scopes, access_type=offline, prompt=consent)

## Task Commits

Each task was committed atomically:

1. **Task 1: Secrets store (keyring + file fallback)** - `37fe503` (feat)
   - Files: internal/secrets/{keyring.go,file.go,detect.go}, go.mod, go.sum
   - KeyringStore: OS keyring via 99designs/keyring with XDG paths
   - FileStore: AES-256-GCM encrypted file (sha256 key derivation, machine-specific default)
   - Platform detection: WSL via /proc/version, headless via DISPLAY/WAYLAND_DISPLAY
   - Auto-selection: keyring first, file fallback with stderr warning

2. **Task 2: OAuth2 flows and token cache** - `b11ee0e` (feat)
   - Files: internal/auth/{scopes.go,server.go,flows.go,token.go}, pkg/browser/browser.go, go.mod, go.sum
   - InteractiveLogin: browser + localhost callback (auto-port selection)
   - ManualLogin: print URL, paste redirect (for SSH/headless)
   - TokenCache: file-locked, implements oauth2.TokenSource, 5-min proactive refresh
   - Browser launcher: cross-platform (macOS, Linux, Windows)

**Plan metadata:** (will be added in final metadata commit)

## Files Created/Modified

- `internal/secrets/keyring.go` - OS keyring implementation (99designs/keyring, XDG paths, Store interface)
- `internal/secrets/file.go` - AES-256-GCM encrypted file storage (sha256 key derivation, JSON map)
- `internal/secrets/detect.go` - Platform detection (WSL, headless) and auto-selection (NewStore)
- `internal/auth/scopes.go` - OAuth2 scope definitions (comma-separated for Zoho)
- `internal/auth/server.go` - Localhost callback server (auto-port, auto-shutdown, HTML success/error pages)
- `internal/auth/flows.go` - InteractiveLogin (browser + callback) and ManualLogin (paste URL)
- `internal/auth/token.go` - File-locked token cache (gofrs/flock, oauth2.TokenSource, proactive refresh)
- `pkg/browser/browser.go` - Cross-platform browser launcher (macOS open, Linux xdg-open, Windows rundll32)

## Decisions Made

1. **99designs/keyring for OS keyring**: Mature library with support for macOS Keychain, Linux Secret Service (GNOME Keyring, KWallet), and Windows Credential Manager. Handles platform quirks.

2. **AES-256-GCM for file encryption**: Standard authenticated encryption. Used sha256 for key derivation (simple v1 approach). Noted scrypt/argon2 as future improvement for better security.

3. **Machine-specific default key**: When password empty, derive key from hostname + username. Prints warning to stderr. Less secure than user password, but better than nothing for WSL/headless.

4. **Platform detection strategy**: WSL via `/proc/version` (case-insensitive "microsoft" or "WSL" match), headless via DISPLAY/WAYLAND_DISPLAY env vars on Linux. Keyring unavailable on WSL even though technically Linux.

5. **Zoho OAuth2 quirks**: Comma-separated scopes (not space), `access_type=offline` required for refresh token, `prompt=consent` to force refresh token issuance. Different from standard OAuth2.

6. **gofrs/flock for file locking**: Cross-platform file locking with timeout support. Prevents concurrent refresh stampede when multiple processes try to refresh simultaneously.

7. **5-minute proactive refresh window**: Refresh token when within 5 minutes of expiry, even if still valid. Reduces risk of auth errors during API calls.

8. **Two-tier token storage**: Refresh token (long-lived, high-value) in secrets Store (keyring or encrypted file). Access token (short-lived, frequently refreshed) in XDG cache as plain JSON. Different security needs.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all code compiled and passed go vet on first attempt after fixing import paths and CLIError pointer types.

## User Setup Required

None - no external service configuration required for this plan. OAuth2 credentials (client_id, client_secret) will be set via config commands in Plan 03.

## Next Phase Readiness

Ready for Plan 03 (CLI command implementations: auth login/logout/status, config get/set/unset).

Auth engine complete. All components compile and satisfy their interfaces:
- KeyringStore and FileStore implement secrets.Store
- TokenCache implements oauth2.TokenSource
- InteractiveLogin and ManualLogin return *oauth2.Token

Plan 03 will wire these into the CLI commands created in Plan 01.

## Self-Check: PASSED

All created files verified on disk:
- internal/secrets/keyring.go, file.go, detect.go
- internal/auth/scopes.go, server.go, flows.go, token.go
- pkg/browser/browser.go

Both commits verified in git log:
- 37fe503: Task 1 (secrets store with keyring + file fallback)
- b11ee0e: Task 2 (OAuth2 flows and file-locked token cache)

Build verification passed:
- `go build ./...` - all packages compile
- `go vet ./...` - no warnings
- Interface satisfaction verified:
  - KeyringStore implements Store: Get, Set, Delete, List
  - FileStore implements Store: Get, Set, Delete, List
  - TokenCache implements oauth2.TokenSource: Token() (*oauth2.Token, error)

---
*Phase: 01-foundation-authentication*
*Completed: 2026-02-14*
