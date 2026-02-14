---
phase: 01-foundation-authentication
plan: 01
subsystem: cli-infrastructure
tags: [go, kong, xdg, json5, lipgloss, termenv]

# Dependency graph
requires:
  - phase: none
    provides: none (first phase)
provides:
  - Go module scaffold with Kong CLI framework
  - XDG-compliant config system with JSON5 support
  - Region mapping for all 8 Zoho data centers
  - Three output formatters (JSON, plain, rich)
  - Exit code framework with 11 defined codes
  - Secrets Store interface for Plan 02
affects: [01-02, 01-03, 02-01, 03-01, 04-01, 05-01, 06-01]

# Tech tracking
tech-stack:
  added: [kong@1.14.0, adrg/xdg@0.5.3, yosuke-furukawa/json5@0.1.1, charmbracelet/lipgloss/v2@beta1, muesli/termenv@0.16.0, rodaine/table@1.3.0, golang.org/x/term@0.40.0]
  patterns: [Kong struct-tag commands, XDG Base Directory, FormatterProvider wrapper for interface binding]

key-files:
  created:
    - main.go (CLI entrypoint with Kong bootstrap)
    - internal/cli/cli.go (Command tree with auth/config/version subcommands)
    - internal/cli/globals.go (Global flags with TTY auto-detection)
    - internal/config/config.go (Config Load/Save with JSON5, Get/Set/Unset methods)
    - internal/config/regions.go (8 Zoho region endpoint mappings)
    - internal/config/xdg.go (XDG-compliant path resolution)
    - internal/output/formatter.go (Formatter interface with JSON/plain/rich implementations)
    - internal/output/table.go (Table rendering for rich mode)
    - internal/output/errors.go (Exit code constants and CLIError struct)
    - internal/secrets/store.go (Store interface for credential storage)
  modified: []

key-decisions:
  - "Go 1.24 required for lipgloss v2 dependencies (auto-upgraded from 1.22)"
  - "FormatterProvider wrapper used for Kong interface binding (Kong doesn't bind interfaces directly)"
  - "Empty region default resolves to 'us' in BeforeApply hook (CLI flag > config > us)"
  - "Placeholder commands print hints via PrintHint, return nil error (implemented in Plan 02/03)"

patterns-established:
  - "Kong BeforeApply hook for dependency injection (config, formatter, globals)"
  - "Config defaults when file missing (no crash on fresh install)"
  - "Three-tier output: JSON (stdout machine), plain (TSV piping), rich (TTY styled)"
  - "Exit codes follow sysexits.h convention (EX_TEMPFAIL=75 for rate limits)"

# Metrics
duration: 5 min
completed: 2026-02-14
---

# Phase 1 Plan 1: Project Scaffold Summary

**Complete Go CLI foundation: Kong framework with command tree, XDG config with JSON5 and 8-region mapping, three-mode output formatter (JSON/plain/rich), exit code framework, and Secrets Store interface**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-14T17:10:54Z
- **Completed:** 2026-02-14T17:16:40Z
- **Tasks:** 2
- **Files created:** 10

## Accomplishments

- Working Go binary (`zoh`) with `--help`, `--version`, and placeholder subcommands
- XDG-compliant config system loads defaults when file missing (no crash on fresh system)
- Region resolution for all 8 Zoho data centers with correct accounts/API/mail endpoints
- Three output modes: JSON (stdout, 2-space indent), plain (TSV for pipes), rich (lipgloss styled tables)
- Exit code constants (0-11, 75) with CLIError for structured error handling
- Secrets Store interface ready for keyring and file implementations in Plan 02

## Task Commits

Each task was committed atomically:

1. **Task 1: Go module, Kong CLI skeleton, and config system** - `ab55007` (feat)
   - Files: go.mod, go.sum, main.go, internal/cli/{cli.go,globals.go}, internal/config/{config.go,regions.go,xdg.go}
   - Bootstrap: Kong.Parse with command tree (auth, config, version)
   - Config: Load/Save JSON5, Get/Set/Unset methods, XDG paths
   - Regions: All 8 Zoho DCs mapped (us, eu, in, au, jp, ca, sa, uk)

2. **Task 2: Output formatters, exit codes, and secrets store interface** - `6212d11` (feat)
   - Files: internal/output/{formatter.go,table.go,errors.go}, internal/secrets/store.go
   - Formatters: JSON, plain (TSV), rich (lipgloss + termenv)
   - Exit codes: 11 constants following sysexits.h + CLIError struct
   - Store interface: Get/Set/Delete/List for Plan 02 implementations

**Plan metadata:** (will be added in final metadata commit)

## Files Created/Modified

- `main.go` - CLI entrypoint: Kong.Parse, ctx.Run, error handling with exit codes
- `go.mod` / `go.sum` - Go module with Kong, XDG, JSON5, lipgloss, termenv, table dependencies
- `internal/cli/cli.go` - Root CLI struct with BeforeApply hook, auth/config/version subcommands
- `internal/cli/globals.go` - Global flags (region, output, verbose) with TTY auto-detection
- `internal/config/config.go` - Config struct with Load/Save, Get/Set/Unset, JSON5 parsing
- `internal/config/regions.go` - RegionConfig map with all 8 Zoho DC endpoints
- `internal/config/xdg.go` - XDG path helpers (ConfigDir, CacheDir, DataDir)
- `internal/output/formatter.go` - Formatter interface + JSON/plain/rich implementations
- `internal/output/table.go` - Table rendering with rodaine/table for rich mode
- `internal/output/errors.go` - Exit code constants (0-11, 75) and CLIError struct
- `internal/secrets/store.go` - Store interface for credential storage (impl in Plan 02)

## Decisions Made

1. **Go 1.24 auto-upgrade**: lipgloss v2 requires Go 1.24+. Go toolchain auto-upgraded from 1.22.9 to 1.24.0 during dependency resolution. No manual intervention needed.

2. **FormatterProvider wrapper**: Kong cannot bind interfaces directly (compile error: "couldn't find binding of type output.Formatter"). Created `FormatterProvider` struct wrapping the `Formatter` interface to enable Kong dependency injection.

3. **Region default resolution**: Empty string region default allows three-tier resolution in BeforeApply: CLI flag > config value > "us" default. User sees empty enum option in help, gets sensible default.

4. **Placeholder command pattern**: Auth and config subcommands print hints via `PrintHint()` and return `nil` error. Full implementation deferred to Plan 02 (auth) and Plan 03 (config). Users see helpful "not yet implemented" messages instead of errors.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Installed Go 1.22.9 to user home directory**
- **Found during:** Task 1 initialization
- **Issue:** `go` command not found in PATH. Cannot compile Go project without Go toolchain.
- **Fix:** Downloaded Go 1.22.9 linux-amd64 tarball to `/home/semmy/.local/go/`, added to PATH, verified with `go version`. Go later auto-upgraded to 1.24.0 for dependency compatibility.
- **Files modified:** `/home/semmy/.local/go/` (installation directory), `/home/semmy/.zshrc` (PATH export)
- **Verification:** `go version` returns "go version go1.24.0 linux/amd64", `go build` succeeds
- **Committed in:** Not committed (system-level installation, not project code)

**2. [Rule 1 - Bug] Fixed FormatterProvider wrapper for Kong interface binding**
- **Found during:** Task 1 verification (`./zoh auth login` test)
- **Issue:** Kong error "couldn't find binding of type output.Formatter". Kong cannot bind Go interfaces directly - only concrete types or pointers to concrete types.
- **Fix:** Created `FormatterProvider` struct with `Formatter output.Formatter` field. Updated BeforeApply to bind `&FormatterProvider{Formatter: ...}`. Updated all Run methods to accept `*FormatterProvider` instead of `output.Formatter`.
- **Files modified:** internal/cli/cli.go
- **Verification:** `./zoh auth login` prints hint successfully, no Kong binding errors
- **Committed in:** ab55007 (included in Task 1 commit)

**3. [Rule 1 - Bug] Fixed placeholder Run methods returning PrintHint() result**
- **Found during:** Task 1 build (`go build` compile error)
- **Issue:** `PrintHint()` returns void, but Run methods tried to `return formatter.PrintHint(...)`. Compile error: "no value used as value".
- **Fix:** Changed all placeholder Run methods to call `fp.Formatter.PrintHint(...)` on separate line, then `return nil`.
- **Files modified:** internal/cli/cli.go (8 placeholder commands)
- **Verification:** `go build` succeeds, no compile errors
- **Committed in:** ab55007 (included in Task 1 commit)

---

**Total deviations:** 3 auto-fixed (1 blocking, 2 bugs)
**Impact on plan:** All necessary for correctness. Go installation is environment setup (not plan scope creep). FormatterProvider and PrintHint fixes resolve Kong/Go type system requirements. No scope changes.

## Issues Encountered

None - all issues were auto-fixed via deviation rules.

## User Setup Required

None - no external service configuration required for this plan.

## Next Phase Readiness

Ready for Plan 02 (OAuth2 authentication flows, keyring storage, token refresh).

All dependencies installed, CLI skeleton functional, output formatters and config system ready for auth commands to build on.

## Self-Check: PASSED

All created files verified on disk:
- main.go, go.mod, go.sum
- internal/cli/cli.go, internal/cli/globals.go
- internal/config/config.go, internal/config/regions.go, internal/config/xdg.go
- internal/output/formatter.go, internal/output/table.go, internal/output/errors.go
- internal/secrets/store.go

Both commits verified in git log:
- ab55007: Task 1 (Go module, Kong CLI, config system)
- 6212d11: Task 2 (output formatters, exit codes, secrets interface)

Binary functionality verified:
- `./zoh version` returns "zoh version dev"
- `./zoh --help` displays auth and config subcommands
- `./zoh auth login` prints placeholder hint

---
*Phase: 01-foundation-authentication*
*Completed: 2026-02-14*
