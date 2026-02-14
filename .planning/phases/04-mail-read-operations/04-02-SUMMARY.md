---
phase: 04-mail-read-operations
plan: 02
subsystem: mail-client
tags: [mail-api, search, threads, attachments, mail-read]
dependency_graph:
  requires: [phase-04-plan-01]
  provides: [search-query-builder, message-search, thread-view, attachment-operations]
  affects: []
tech_stack:
  added: [zoho-search-syntax]
  patterns: [search-query-builder, client-side-thread-filtering, binary-streaming, auto-filename-detection]
key_files:
  created:
    - internal/zoho/search.go
  modified:
    - internal/zoho/mail_client.go
    - internal/cli/mail_messages.go
    - internal/cli/cli.go
decisions:
  - "SearchQuery builder uses method chaining for fluent API construction"
  - "URL-encoding via url.QueryEscape for search query parameters"
  - "GetThread uses client-side filtering (no dedicated API endpoint) with configurable scan limit"
  - "GetThread defaults to 200 message scan limit to prevent unbounded fetching"
  - "DownloadAttachment streams binary response with io.Copy (no memory buffering)"
  - "Best-effort cleanup of partial downloads on error via os.Remove"
  - "Auto-filename detection: ListAttachments lookup if --output-path not specified"
  - "Attachment download confirmation printed to os.Stderr (matches CLI pattern)"
  - "formatBytes helper reused from mail_messages.go (already existed for message size display)"
  - "OutputPath field renamed to avoid global --output flag conflict"
metrics:
  duration: 4 min
  tasks_completed: 2
  files_created: 1
  files_modified: 3
  commits: 2
  lines_added: 549
  completed_date: 2026-02-14
---

# Phase 04 Plan 02: Message Search, Threads, and Attachments Summary

**One-liner:** Search query builder with chainable methods, message search/thread commands, and attachment list/download with binary streaming.

## What Was Built

### Task 1: Search Query Builder and MailClient Methods (Commit: 4baa5d1)

Created search infrastructure and extended MailClient:

**internal/zoho/search.go** - Fluent search query builder (85 lines):
- `SearchQuery` struct with `parts []string` accumulator
- `NewSearchQuery()` - Creates empty builder
- Chainable methods (all return `*SearchQuery`):
  - `From(email)` - Appends `from:{email}`
  - `To(email)` - Appends `to:{email}`
  - `Subject(text)` - Appends `subject:{text}`
  - `DateAfter(date)` - Appends `after:2006/01/02` format
  - `DateBefore(date)` - Appends `before:2006/01/02` format
  - `HasAttachment()` - Appends `has:attachment`
  - `IsUnread()` - Appends `is:unread`
  - `Text(query)` - Appends free-text query
- `Build()` - Returns space-joined parts
- `IsEmpty()` - Returns true if no criteria added

**internal/zoho/mail_client.go** - Extended with 4 new methods (+134 lines):

`SearchMessages(ctx, searchKey, start, limit)`:
- GET `/api/accounts/{accountId}/messages/search?searchKey={urlencoded}&start={start}&limit={limit}`
- Uses `url.QueryEscape` for safe query encoding
- Returns `[]MessageSummary`

`GetThread(ctx, folderID, threadID, limit)`:
- No dedicated thread API endpoint - client-side filtering approach
- Paginates through folder messages (200 per page max)
- Filters by matching `ThreadID` field
- Configurable scan limit (default 200) prevents unbounded fetching
- Returns filtered slice, error if no matches found

`ListAttachments(ctx, folderID, messageID)`:
- GET `/api/accounts/{accountId}/folders/{folderId}/messages/{messageId}/attachments`
- Returns `[]Attachment` with metadata (name, size, type, ID)

`DownloadAttachment(ctx, folderID, messageID, attachmentID, destPath)`:
- GET `/api/accounts/{accountId}/folders/{folderId}/messages/{messageId}/attachments/{attachmentId}`
- Binary response (application/octet-stream), NOT JSON
- Streams to file with `io.Copy` (no memory buffering)
- Best-effort cleanup via `os.Remove` on download failure
- Returns error if file creation fails before attempting download

### Task 2: Search, Thread, and Attachment CLI Commands (Commit: 93962fd)

Added 4 new commands and helper:

**internal/cli/mail_messages.go** - Extended with search, thread, attachments (+325 lines):

`MailMessagesSearchCmd`:
- Optional `Query` arg for free-text search
- Flags: `--from`, `--subject`, `--after` (YYYY-MM-DD), `--before` (YYYY-MM-DD), `--unread`, `--has-attachment`, `--limit` (default 50)
- Builds `SearchQuery` from flags, validates at least one criterion
- Parses dates with `time.Parse("2006-01-02", ...)`
- Returns error `ExitUsage` if no search criteria
- Outputs MessageListRow display (same columns as list command)

`MailMessagesThreadCmd`:
- `ThreadID` arg (required)
- Flags: `--folder` (default "Inbox"), `--limit` (default 200 scan limit)
- Resolves folder name/ID via `GetFolderByName` fallback pattern
- Calls `mc.GetThread` with configurable scan limit
- Outputs MessageListRow chronologically

`MailAttachmentsListCmd`:
- `MessageID` arg (required)
- Flag: `--folder` (required)
- Calls `mc.ListAttachments`
- Outputs `AttachmentListRow` display struct:
  - Columns: Name, Size (formatted), Type, ID
  - Size formatting via `formatBytes` helper (already existed)

`AttachmentListRow` display struct:
- Name, Size (string, human-readable), Type, AttachmentID

`MailAttachmentsDownloadCmd`:
- `AttachmentID` arg (required)
- Flags: `--message-id` (required), `--folder` (required), `--output-path` (optional)
- Auto-filename detection: if `--output-path` not specified, calls `ListAttachments` to get attachment name
- Streams download via `mc.DownloadAttachment`
- Confirmation to `os.Stderr`: "Downloaded: {filename} ({size})" or "Downloaded: {filename}"
- No stdout output (file is the output)

**internal/cli/cli.go** - Command tree updates (+10 lines):
- Added `Search` and `Thread` to `MailMessagesCmd`
- Added `Attachments MailAttachmentsCmd` to `MailCmd`
- `MailAttachmentsCmd` with `List` and `Download` subcommands

Updated command hierarchy:
- `zoh mail messages search [query] --from X --subject X --after X --before X --unread --has-attachment`
- `zoh mail messages thread <thread-id> --folder Inbox`
- `zoh mail attachments list <message-id> --folder <name-or-id>`
- `zoh mail attachments download <attachment-id> --message-id <id> --folder <name-or-id> [--output-path path]`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] OutputPath renamed to avoid global flag conflict**
- **Found during:** Task 2, building binary
- **Issue:** `MailAttachmentsDownloadCmd.Output` conflicted with global `Globals.Output` flag (`--output` for format selection). Kong error: "duplicate flag --output"
- **Fix:** Renamed field to `OutputPath`, added `name:"output-path"` tag to create `--output-path` flag
- **Files modified:** internal/cli/mail_messages.go
- **Commit:** 93962fd (Task 2 commit)

**2. [Rule 1 - Bug] Fixed exit code constant name**
- **Found during:** Task 2, initial build
- **Issue:** Used `output.ExitUsageError` constant which doesn't exist
- **Fix:** Changed to `output.ExitUsage` (correct constant from internal/output/errors.go)
- **Files modified:** internal/cli/mail_messages.go (3 occurrences)
- **Commit:** 93962fd (Task 2 commit)

## Key Patterns Established

1. **Fluent search query builder** - Chainable methods for constructing Zoho search syntax, enables flexible search composition
2. **Client-side thread filtering** - No dedicated thread API endpoint, pagination + filter approach with scan limit safety
3. **Binary streaming with io.Copy** - Attachment download streams directly to disk, no memory buffering for large files
4. **Auto-filename detection** - Download command fetches attachment metadata to determine filename when not specified
5. **Best-effort cleanup** - Partial downloads cleaned up with `os.Remove` on error
6. **Folder resolution reuse** - Same `GetFolderByName` fallback pattern from Phase 04 Plan 01
7. **Display struct reuse** - AttachmentListRow follows MessageListRow pattern for formatted output

## Technical Decisions

- **SearchQuery builder uses method chaining**: Fluent API pattern enables readable query construction: `sq.From("x").Subject("y").HasAttachment().Build()`
- **URL-encoding for search queries**: `url.QueryEscape` ensures safe handling of special characters in search syntax
- **Client-side thread filtering approach**: Zoho API lacks dedicated thread GET endpoint, pagination + filtering is reliable but requires scan limit to prevent unbounded fetching
- **Default 200 message scan limit**: Balances thread completeness with performance for large folders
- **Binary streaming via io.Copy**: Attachment downloads stream response body directly to file, prevents memory issues with large attachments
- **Auto-filename from attachment metadata**: Better UX - users can omit `--output-path` and CLI fetches filename from API
- **OutputPath field naming**: Avoids conflict with global `--output` flag (format selection), uses `--output-path` for file destination

## Verification Results

All verification steps passed:
- ✓ `go build ./...` - Entire project compiles
- ✓ `go vet ./...` - No warnings
- ✓ `./zoh mail messages search --help` - Shows all search flags (from, subject, after, before, unread, has-attachment, limit)
- ✓ `./zoh mail messages thread --help` - Shows thread-id arg and folder/limit flags
- ✓ `./zoh mail attachments list --help` - Shows message-id arg and folder flag
- ✓ `./zoh mail attachments download --help` - Shows attachment-id arg, message-id, folder, output-path flags
- ✓ SearchQuery builder exists in internal/zoho/search.go
- ✓ MailClient has SearchMessages, GetThread, ListAttachments, DownloadAttachment methods

## Commits

| Hash    | Type | Description                                                      |
|---------|------|------------------------------------------------------------------|
| 4baa5d1 | feat | Search query builder and MailClient search/thread/attachment methods |
| 93962fd | feat | Search, thread, and attachment CLI commands                     |

## Files Changed

**Created (1 file):**
- `internal/zoho/search.go` (85 lines) - SearchQuery builder for Zoho search syntax

**Modified (3 files):**
- `internal/zoho/mail_client.go` (+134 lines) - SearchMessages, GetThread, ListAttachments, DownloadAttachment methods
- `internal/cli/mail_messages.go` (+325 lines) - Search, thread, and attachment commands
- `internal/cli/cli.go` (+10 lines) - Command tree updates for search/thread/attachments

## Dependencies

**Requires:**
- Phase 04 Plan 01: MailClient infrastructure, mail types, folder resolution, MessageListRow pattern

**Provides:**
- SearchQuery builder for Phase 5 (if needed for sent mail search)
- Complete mail read functionality (search, threads, attachments)
- Pattern for binary file downloads (reusable for future file operations)

**Affects:**
- None (Phase 04 complete - no remaining mail read plans)

## Self-Check: PASSED

**Files exist:**
- FOUND: internal/zoho/search.go
- FOUND: internal/zoho/mail_client.go (modified)
- FOUND: internal/cli/mail_messages.go (modified)
- FOUND: internal/cli/cli.go (modified)

**Commits exist:**
- FOUND: 4baa5d1
- FOUND: 93962fd

**Binary works:**
- FOUND: ./zoh mail messages search --help shows all search flags
- FOUND: ./zoh mail messages thread --help shows thread-id arg
- FOUND: ./zoh mail attachments list --help shows message-id arg
- FOUND: ./zoh mail attachments download --help shows all required flags
