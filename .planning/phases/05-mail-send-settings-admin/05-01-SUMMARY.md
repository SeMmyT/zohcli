---
phase: 05-mail-send-settings-admin
plan: 01
subsystem: mail-send
tags: [mail, send, compose, reply, forward, attachments]
dependency_graph:
  requires: [mail-client-infrastructure]
  provides: [send-email-api, compose-cli, reply-cli, forward-cli, attachment-upload]
  affects: [mail-operations]
tech_stack:
  added: []
  patterns: [two-step-attachment-upload, binary-content-type, action-based-routing]
key_files:
  created:
    - internal/zoho/mail_send.go
    - internal/cli/mail_send.go
  modified:
    - internal/zoho/mail_types.go
    - internal/cli/cli.go
decisions:
  - "Attachment upload uses application/octet-stream Content-Type (not multipart/form-data)"
  - "Two-step attachment workflow: upload first, reference in send request"
  - "Action field in request determines operation type (reply/replyall/forward)"
  - "Reply-all combines original ToAddress and CcAddress into new CcAddress field"
  - "resolveFolderID helper provides folder name/ID flexibility (matches existing pattern)"
metrics:
  duration_minutes: 2
  tasks_completed: 2
  files_created: 2
  files_modified: 2
  commits: 2
  completed_date: 2026-02-14
---

# Phase 5 Plan 1: Mail Send Operations Summary

**One-liner:** Email send operations with compose, reply, reply-all, forward, and file attachment upload via two-step workflow

## What Was Built

Implemented complete email send functionality for the Zoho Mail CLI, enabling users to compose new emails, reply to messages (single or all recipients), forward messages, and attach local files to any send operation.

**Core Components:**

1. **MailClient Send API** (internal/zoho/mail_send.go):
   - `SendEmail` - Send new email message
   - `ReplyToEmail` - Reply to original sender
   - `ReplyAllToEmail` - Reply to all recipients
   - `ForwardEmail` - Forward message to new recipients
   - `UploadAttachment` - Upload file and return reference for send request

2. **Send Request Types** (internal/zoho/mail_types.go):
   - `SendEmailRequest` - Request body with to/cc/bcc, subject, content, format, action, attachments
   - `AttachmentReference` - Uploaded attachment metadata (storeName, attachmentName, attachmentPath)
   - `AttachmentUploadResponse` - Upload API response wrapper
   - `SendEmailResponse` - Send API response wrapper

3. **CLI Send Commands** (internal/cli/mail_send.go):
   - `zoh mail send compose` - Compose and send new email
   - `zoh mail send reply` - Reply to message (with --all for reply-all)
   - `zoh mail send forward` - Forward message to new recipient
   - `resolveFolderID` - Folder name/ID resolution helper

## Task Breakdown

| Task | Name                                                        | Commit  | Files Modified                                               |
| ---- | ----------------------------------------------------------- | ------- | ------------------------------------------------------------ |
| 1    | Send types, attachment upload, and MailClient send methods  | 088bc44 | internal/zoho/mail_send.go, internal/zoho/mail_types.go      |
| 2    | Send CLI commands (compose, reply, forward)                 | 4cd9392 | internal/cli/mail_send.go, internal/cli/cli.go               |

## Key Technical Details

**Attachment Upload Pattern:**
- Uses `application/octet-stream` Content-Type (not `multipart/form-data`)
- Bypasses `DoMail` helper to avoid automatic `application/json` header
- Directly uses `mc.client.httpClient.Do` with manual request construction
- Returns `AttachmentReference` for inclusion in send request

**Send Request Architecture:**
- Single `SendEmailRequest` type used for all operations
- `Action` field determines operation: `reply`, `replyall`, `forward`
- `MailFormat` field specifies `html` or `plaintext`
- Attachments array holds uploaded attachment references

**Reply-All Logic:**
- Fetches original message metadata via `GetMessageMetadata`
- Combines `ToAddress` and `CcAddress` from original message
- Sets combined list as `CcAddress` in reply request
- ToAddress always set to original sender (`FromAddress`)

**CLI User Experience:**
- All send commands support `--attach` repeatable flag for multiple files
- `--html` flag switches from plaintext (default) to HTML mode
- Folder resolution accepts both folder name and folder ID
- Confirmation messages printed to stderr after successful send

## Deviations from Plan

None - plan executed exactly as written.

## Verification Results

All verification steps passed:

1. Build: `go build ./...` - SUCCESS
2. Vet: `go vet ./...` - SUCCESS
3. Help text verification:
   - `./zoh mail send --help` shows compose, reply, forward subcommands
   - `./zoh mail send compose --help` shows --to, --subject, --body, --html, --cc, --bcc, --attach
   - `./zoh mail send reply --help` shows message-id arg and --folder, --body, --html, --attach, --all
   - `./zoh mail send forward --help` shows message-id arg and --folder, --to, --body, --html, --attach
4. Code inspection confirms `Content-Type: application/octet-stream` for attachment upload

## Success Criteria

- [x] MailClient has complete send API: SendEmail, ReplyToEmail, ReplyAllToEmail, ForwardEmail, UploadAttachment
- [x] CLI commands for compose, reply (with --all for reply-all), and forward are registered and show correct help
- [x] Attachment upload uses two-step workflow (upload first, reference in send request)
- [x] Content-Type is application/octet-stream for attachment uploads (NOT multipart/form-data)
- [x] All send types defined in mail_types.go following existing response wrapper pattern

## Self-Check: PASSED

**Files created:**
- internal/zoho/mail_send.go - FOUND
- internal/cli/mail_send.go - FOUND

**Files modified:**
- internal/zoho/mail_types.go - FOUND
- internal/cli/cli.go - FOUND

**Commits:**
- 088bc44 - FOUND (feat(05-01): add MailClient send methods and types)
- 4cd9392 - FOUND (feat(05-01): add send CLI commands)
