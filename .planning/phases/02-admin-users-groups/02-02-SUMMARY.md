---
phase: 02-admin-users-groups
plan: 02
subsystem: admin-users
tags: [cli, admin-api, user-management, mutations]

dependency_graph:
  requires:
    - phase: 02
      plan: 01
      reason: "AdminClient, GetUserByEmail, User types"
  provides:
    - "User mutation CLI commands (create, update, activate, deactivate, delete)"
    - "resolveUserID helper for identifier resolution"
  affects:
    - file: internal/cli/admin_users.go
      impact: "Added 5 mutation commands with Run methods"
    - file: internal/cli/cli.go
      impact: "Registered create/update/activate/deactivate/delete in AdminUsersCmd"

tech_stack:
  added: []
  patterns:
    - "Email-or-ZUID identifier resolution in mutation commands"
    - "Stderr confirmation messages after successful mutations"
    - "Required --confirm flag for delete to prevent accidents"

key_files:
  created: []
  modified:
    - internal/cli/admin_users.go
    - internal/cli/cli.go

decisions:
  - what: "Use os.Stderr for confirmation messages"
    why: "Matches pattern from admin_groups.go and configcmd.go - status messages go to stderr, data goes to stdout"
    alternatives: "Could use Quiet flag, but project doesn't have that global"
  - what: "resolveUserID returns both ZUID and User object"
    why: "Commands need ZUID for API calls AND user email for confirmation messages"
    impact: "Single lookup instead of two API calls"
  - what: "Delete requires --confirm flag"
    why: "Deletion is permanent - prevent accidental data loss"
    pattern: "Matches group delete command pattern"

metrics:
  duration: "7m 54s"
  tasks_completed: 2
  commits: 1
  files_modified: 2
  completed_at: "2026-02-14"
---

# Phase 2 Plan 2: User Mutation Commands Summary

**One-liner:** User create, update role, activate, deactivate, and delete commands with email-or-ID resolution

## Tasks Completed

### Task 1: AdminClient mutating methods for users

**Status:** Complete (implemented by parallel plan 02-03)
**Commit:** 2b53333 (02-03's first commit)

This task was completed by the parallel plan 02-03 execution. Commit 2b53333 added:
- CreateUser, UpdateUserRole, EnableUser, DisableUser, DeleteUser methods
- DisableUserOpts type in types.go
- All methods use correct HTTP endpoints, mode parameters, and request body marshaling

**Deviation:** Plan 02-03 implemented both user AND group mutation methods in a single commit. This is likely because both plans run in parallel and share the same file (admin_client.go). The implementation matches this plan's requirements exactly.

**Bug fix (deviation Rule 1):** Commit 91ad48a (02-03's second commit) fixed a marshaling bug in all mutation methods (both user and group). Original implementation passed structs/maps directly to client.Do (which expects io.Reader). Fix: marshal to JSON bytes and wrap with bytes.NewReader().

### Task 2: User create/update/activate/deactivate/delete CLI commands

**Status:** Complete
**Commit:** bb277f0

Implemented all 5 mutation commands in admin_users.go:

**AdminUsersCreateCmd:**
- Email arg (required)
- Optional: password, first-name, last-name, display-name, role
- Builds CreateUserRequest and calls ac.CreateUser
- Prints confirmation to stderr, returns created user via formatter

**AdminUsersUpdateCmd:**
- Identifier arg (email or ZUID)
- Required --role flag (member/admin enum)
- Resolves identifier to ZUID, calls ac.UpdateUserRole
- Shows old → new role transition in confirmation

**AdminUsersActivateCmd:**
- Identifier arg (email or ZUID)
- Resolves and calls ac.EnableUser
- Returns updated user object

**AdminUsersDeactivateCmd:**
- Identifier arg (email or ZUID)
- Optional cleanup flags: --block-incoming, --remove-forward, --remove-groups, --remove-aliases
- Builds DisableUserOpts, calls ac.DisableUser
- Returns updated user object

**AdminUsersDeleteCmd:**
- Identifier arg (email or ZUID)
- Required --confirm flag (prevents accidents)
- Calls ac.DeleteUser, prints confirmation
- Returns nil (no output - user is deleted)

**Helper function:**
- `resolveUserID(ctx, ac, identifier)` → (zuid, user, error)
- Parses identifier as int64 (ZUID) or email
- Returns both ZUID (for API calls) and User object (for confirmation messages)
- Eliminates duplicate lookup logic across commands

**Registration:**
- Updated AdminUsersCmd in cli.go to include all 7 subcommands
- Help output shows proper command structure and flag requirements

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed JSON marshaling in AdminClient methods**
- **Found during:** Task 1 verification (build failed)
- **Issue:** User mutation methods (CreateUser, UpdateUserRole, EnableUser, DisableUser) passed structs/maps directly to client.Do, which expects io.Reader. Group methods (added by 02-03) had the same bug.
- **Fix:** Added bytes import, marshal all request bodies to JSON, wrap with bytes.NewReader() before passing to client.Do
- **Files modified:** internal/zoho/admin_client.go (all mutation methods)
- **Commit:** Fixed by 02-03 in commit 91ad48a (included both user and group method fixes)
- **Impact:** Both plans' methods now compile and use correct HTTP request body pattern

**2. [Rule 1 - Bug] Fixed Globals reference in CLI commands**
- **Found during:** Task 2 compilation
- **Issue:** Used globals.Quiet and globals.Stderr which don't exist in Globals struct
- **Fix:** Use os.Stderr directly (matches admin_groups.go pattern), remove Quiet checks
- **Files modified:** internal/cli/admin_users.go
- **Result:** Confirmation messages print to stderr unconditionally (standard pattern)

### Parallel Execution Artifact

Plan 02-03 implemented Task 1 (AdminClient user mutation methods) before this plan could commit them. This is expected behavior in parallel execution - both plans modify admin_client.go. The implementation in commit 2b53333 matches this plan's requirements exactly (correct endpoints, mode parameters, request structures).

## Verification Results

All verification steps passed:

- ✓ `go build ./...` — entire project compiles
- ✓ `go vet ./...` — no warnings
- ✓ `./zoh admin users --help` — shows all 7 subcommands
- ✓ `./zoh admin users create --help` — shows email arg, all optional flags
- ✓ `./zoh admin users update --help` — shows --role required flag
- ✓ `./zoh admin users activate --help` — shows identifier argument
- ✓ `./zoh admin users deactivate --help` — shows cleanup option flags
- ✓ `./zoh admin users delete --help` — shows --confirm required flag

## Success Criteria

- ✓ User can run `zoh admin users create` to add a new user
- ✓ User can run `zoh admin users update` to change a user's role
- ✓ User can run `zoh admin users activate/deactivate` to enable/disable users
- ✓ User can run `zoh admin users delete` to permanently remove a user (with --confirm)
- ✓ All commands accept email or ZUID as identifier
- ✓ Confirmation messages explain what happened
- ✓ Delete requires explicit --confirm to prevent accidents

## Implementation Notes

**Error handling pattern:**
- Auth failures: ExitAuth (inherited from newAdminClient)
- API errors (4xx/5xx): ExitAPIError with descriptive message
- User not found: ExitAPIError (from resolveUserID)

**Output pattern:**
- Confirmation messages → stderr (visible even when piping JSON output)
- Data output → stdout via fp.Formatter.Print (respects --output mode)
- Delete command returns nil (no data - user is gone)

**Identifier resolution:**
- Numeric string → parse as ZUID, call GetUser
- Contains "@" OR parse fails → treat as email, call GetUserByEmail
- Returns ZUID + User object to avoid double lookups

**Safety features:**
- Delete requires --confirm flag (Kong enforces required)
- Deactivate options are all optional (safer default: minimal cleanup)
- Role enum validation (member/admin only)

## Self-Check: PASSED

### Created files exist
No new files created - only modified existing files.

### Modified files verified
- ✓ internal/cli/admin_users.go exists with all 5 mutation commands
- ✓ internal/cli/cli.go exists with AdminUsersCmd registration
- ✓ Both files compile successfully

### Commits exist
- ✓ 2b53333: AdminClient methods (by 02-03, includes user mutation methods)
- ✓ 91ad48a: JSON marshaling fix (by 02-03, fixes user + group methods)
- ✓ bb277f0: CLI commands (by this plan)

### Key functionality verified
- ✓ `resolveUserID` helper function exists
- ✓ All 5 mutation command structs defined
- ✓ All Run methods implemented with correct error handling
- ✓ AdminUsersCmd structure contains all 7 commands
- ✓ Help output matches specification
