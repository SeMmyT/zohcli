# Phase 5: Mail -- Send, Settings & Admin - Research

**Researched:** 2026-02-14
**Domain:** Zoho Mail API (send operations, account settings, admin policies)
**Confidence:** MEDIUM-HIGH

## Summary

Phase 5 implements mail composition/sending, account settings management, and admin policy configuration through the **Zoho Mail API**. This phase builds on Phase 4's read-only MailClient by adding write operations for sending email (compose, reply, forward), uploading attachments, managing signatures/vacation replies, and configuring organization-level spam controls and mail policies.

**Critical findings:**
- **Two-step attachment workflow**: Upload attachments to file store FIRST (returns storeName/path), THEN include those references in send email request. Not inline multipart like traditional email APIs.
- **Content-Type gotcha for attachments**: Must use `application/octet-stream` for binary uploads, NOT `multipart/form-data` (community-reported blocker). Official docs misleading.
- **Reply/Forward use same endpoint**: `POST /messages/{messageId}` with `action` parameter ("reply", "replyall", "forward") distinguishes operation type.
- **Mode-based API pattern**: Settings endpoints use `mode` parameter in request body to multiplex operations (e.g., `mode: "addVacationReply"`, `mode: "updateDisplayName"`).
- **Admin vs User endpoints**: Most settings/admin operations have dual endpoints (admin auth with zoid/zuid, user auth with accountId only).
- **Spam control uses enum categories**: Single PUT endpoint handles all allowlist/blocklist types via `spamCategory` field (whiteListEmail, spamDomain, quarantineIP, etc.).

**Primary recommendation:** Extend MailClient with send methods using two-step attachment flow, implement mode-based request builders for settings operations, create AdminPolicyClient for organization-level spam/retention controls, and add comprehensive Content-Type handling for binary vs JSON payloads.

## Standard Stack

### Core (Already in go.mod)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| mime/multipart | stdlib | Multipart form builder | Required for raw file uploads with proper boundaries |
| io | stdlib | Stream handling | Binary file reading for attachment uploads |
| os | stdlib | File operations | Reading attachment files from disk |
| net/http | stdlib | HTTP client | POST/PUT requests for send/settings APIs |
| encoding/json | stdlib | JSON marshaling | Request/response format for most operations |
| golang.org/x/oauth2 | v0.35.0 | OAuth2 auth flow | Required for Zoho API authentication |

### Supporting (Already in go.mod)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/alecthomas/kong | v1.14.0 | CLI command parsing | Struct tags for all send/settings commands |
| github.com/charmbracelet/lipgloss/v2 | v2.0.0-beta1 | Rich terminal output | Styled output formatting |
| github.com/rodaine/table | v1.3.0 | Table rendering | List output for signatures, policies |

### New Dependencies Required
**None.** All required libraries are already in the project from Phases 1-4. Standard library's `mime/multipart` handles attachment uploads natively.

**Installation:**
No new dependencies needed. Existing stack sufficient.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── zoho/
│   ├── client.go              # Existing: region-aware HTTP client
│   ├── mail_client.go         # EXTEND: add send/settings methods
│   ├── mail_send.go           # NEW: send operation request builders
│   ├── mail_settings.go       # NEW: account settings (signatures, vacation)
│   ├── admin_policy_client.go # NEW: org-level policies (spam, retention)
│   ├── mail_types.go          # EXTEND: add send/settings types
│   └── attachment_upload.go   # NEW: two-step attachment upload
├── cli/
│   ├── mail_send.go           # NEW: compose/reply/forward commands
│   ├── mail_settings.go       # NEW: signature/vacation/display name commands
│   └── mail_admin.go          # NEW: spam/retention/logs commands
└── output/
    └── formatter.go           # Existing: Formatter interface
```

### Pattern 1: Two-Step Attachment Upload

**What:** Upload files to Zoho file store first, then reference them in send email request.

**When to use:** For all email sending operations that include attachments (compose, reply, forward).

**Example:**
```go
// internal/zoho/attachment_upload.go (new file)
package zoho

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
)

// AttachmentUploadResponse from POST /messages/attachments
type AttachmentUploadResponse struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data struct {
        StoreName      string `json:"storeName"`
        AttachmentName string `json:"attachmentName"`
        AttachmentPath string `json:"attachmentPath"`
    } `json:"data"`
}

// UploadAttachment uploads a file and returns references for email send
func (mc *MailClient) UploadAttachment(ctx context.Context, filePath string) (*AttachmentReference, error) {
    // Open file
    file, err := os.Open(filePath)
    if err != nil {
        return nil, fmt.Errorf("open file: %w", err)
    }
    defer file.Close()

    // Extract filename from path
    fileName := filepath.Base(filePath)

    // Build request URL with fileName query param
    path := fmt.Sprintf("/api/accounts/%s/messages/attachments?fileName=%s",
        mc.accountID, url.QueryEscape(fileName))

    // Create request with custom Content-Type
    urlStr := mc.client.region.MailBase + path
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, file)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    // CRITICAL: Use application/octet-stream, NOT multipart/form-data
    req.Header.Set("Content-Type", "application/octet-stream")

    // Execute via HTTP client (goes through OAuth2 + rate limit transports)
    resp, err := mc.client.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, mc.parseErrorResponse(resp)
    }

    var uploadResp AttachmentUploadResponse
    if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    if uploadResp.Status.Code != 200 {
        return nil, fmt.Errorf("API error: %s (code %d)",
            uploadResp.Status.Description, uploadResp.Status.Code)
    }

    return &AttachmentReference{
        StoreName:      uploadResp.Data.StoreName,
        AttachmentName: uploadResp.Data.AttachmentName,
        AttachmentPath: uploadResp.Data.AttachmentPath,
    }, nil
}

// AttachmentReference holds uploaded attachment metadata for send requests
type AttachmentReference struct {
    StoreName      string `json:"storeName"`
    AttachmentName string `json:"attachmentName"`
    AttachmentPath string `json:"attachmentPath"`
}
```
**Source:** Derived from [POST - Upload attachment](https://www.zoho.com/mail/help/api/post-upload-attachments.html) and [Zoho Mail API: How to Upload an Attachment](https://pebblesrox.wordpress.com/2021/03/28/zoho-mail-api-how-to-upload-an-attachment/)

### Pattern 2: Send Email with Action-Based Routing

**What:** Use single endpoint with `action` parameter to distinguish compose vs reply vs forward.

**When to use:** For all email sending operations.

**Example:**
```go
// internal/zoho/mail_send.go (new file)
package zoho

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

// SendEmailRequest represents email composition parameters
type SendEmailRequest struct {
    FromAddress string                 `json:"fromAddress"`
    ToAddress   string                 `json:"toAddress"`
    CcAddress   string                 `json:"ccAddress,omitempty"`
    BccAddress  string                 `json:"bccAddress,omitempty"`
    Subject     string                 `json:"subject"`
    Content     string                 `json:"content"`
    MailFormat  string                 `json:"mailFormat,omitempty"` // "html" or "plaintext"
    Action      string                 `json:"action,omitempty"`     // "reply", "replyall", "forward"
    Attachments []AttachmentReference  `json:"attachments,omitempty"`
}

// SendEmail sends a new email (compose)
func (mc *MailClient) SendEmail(ctx context.Context, req *SendEmailRequest) error {
    path := fmt.Sprintf("/api/accounts/%s/messages", mc.accountID)
    return mc.sendEmailRequest(ctx, path, req)
}

// ReplyToEmail sends a reply to an existing message
func (mc *MailClient) ReplyToEmail(ctx context.Context, messageID string, req *SendEmailRequest) error {
    req.Action = "reply"
    path := fmt.Sprintf("/api/accounts/%s/messages/%s", mc.accountID, messageID)
    return mc.sendEmailRequest(ctx, path, req)
}

// ReplyAllToEmail sends a reply-all to an existing message
func (mc *MailClient) ReplyAllToEmail(ctx context.Context, messageID string, req *SendEmailRequest) error {
    req.Action = "replyall"
    path := fmt.Sprintf("/api/accounts/%s/messages/%s", mc.accountID, messageID)
    return mc.sendEmailRequest(ctx, path, req)
}

// ForwardEmail forwards an existing message
func (mc *MailClient) ForwardEmail(ctx context.Context, messageID string, req *SendEmailRequest) error {
    req.Action = "forward"
    path := fmt.Sprintf("/api/accounts/%s/messages/%s", mc.accountID, messageID)
    return mc.sendEmailRequest(ctx, path, req)
}

// sendEmailRequest is the common send logic
func (mc *MailClient) sendEmailRequest(ctx context.Context, path string, req *SendEmailRequest) error {
    // Marshal request body
    body, err := json.Marshal(req)
    if err != nil {
        return fmt.Errorf("marshal request: %w", err)
    }

    resp, err := mc.client.DoMail(ctx, http.MethodPost, path, bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return mc.parseErrorResponse(resp)
    }

    var sendResp struct {
        Status struct {
            Code        int    `json:"code"`
            Description string `json:"description"`
        } `json:"status"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&sendResp); err != nil {
        return fmt.Errorf("decode response: %w", err)
    }

    if sendResp.Status.Code != 200 {
        return fmt.Errorf("API error: %s (code %d)",
            sendResp.Status.Description, sendResp.Status.Code)
    }

    return nil
}
```
**Source:** Derived from [POST - Send an Email](https://www.zoho.com/mail/help/api/post-send-an-email.html) and [POST - Send reply to an email](https://www.zoho.com/mail/help/api/post-reply-to-an-email.html)

### Pattern 3: Mode-Based Settings Management

**What:** Use `mode` parameter in PUT request body to multiplex different settings operations on same endpoint.

**When to use:** For signature, vacation, display name, and forwarding settings.

**Example:**
```go
// internal/zoho/mail_settings.go (new file)
package zoho

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

// AddVacationReply configures vacation auto-reply
func (mc *MailClient) AddVacationReply(ctx context.Context, vacation *VacationReply) error {
    req := map[string]interface{}{
        "mode": "addVacationReply",
        "vacationResponse": map[string]interface{}{
            "fromDate":   vacation.FromDate,   // MM/DD/YYYY HH:MM:SS
            "toDate":     vacation.ToDate,     // MM/DD/YYYY HH:MM:SS
            "sendingInt": vacation.SendingInt, // int
            "subject":    vacation.Subject,
            "content":    vacation.Content,
            "sendTo":     vacation.SendTo, // "all", "contacts", "org", etc.
        },
    }

    return mc.updateAccountSettings(ctx, req)
}

// UpdateDisplayName updates user's display name
func (mc *MailClient) UpdateDisplayName(ctx context.Context, displayName string) error {
    req := map[string]interface{}{
        "mode":        "updateDisplayName",
        "displayName": displayName,
    }

    return mc.updateAccountSettings(ctx, req)
}

// UpdateDisplayNameAndEmail updates both display name and email
func (mc *MailClient) UpdateDisplayNameAndEmail(ctx context.Context, displayName, emailAddress string) error {
    req := map[string]interface{}{
        "mode":         "displaynameemailupdate",
        "displayName":  displayName,
        "emailAddress": emailAddress,
    }

    return mc.updateAccountSettings(ctx, req)
}

// updateAccountSettings is the common settings update logic
func (mc *MailClient) updateAccountSettings(ctx context.Context, reqBody map[string]interface{}) error {
    path := fmt.Sprintf("/api/accounts/%s", mc.accountID)

    body, err := json.Marshal(reqBody)
    if err != nil {
        return fmt.Errorf("marshal request: %w", err)
    }

    resp, err := mc.client.DoMail(ctx, http.MethodPut, path, bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return mc.parseErrorResponse(resp)
    }

    var settingsResp struct {
        Status struct {
            Code        int    `json:"code"`
            Description string `json:"description"`
        } `json:"status"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&settingsResp); err != nil {
        return fmt.Errorf("decode response: %w", err)
    }

    if settingsResp.Status.Code != 200 {
        return fmt.Errorf("API error: %s (code %d)",
            settingsResp.Status.Description, settingsResp.Status.Code)
    }

    return nil
}

// VacationReply holds vacation auto-reply configuration
type VacationReply struct {
    FromDate   string // MM/DD/YYYY HH:MM:SS
    ToDate     string // MM/DD/YYYY HH:MM:SS
    SendingInt int    // Reply interval
    Subject    string
    Content    string
    SendTo     string // "all", "contacts", "noncontacts", "org", etc.
}
```
**Source:** Derived from [PUT - Add vacation reply](https://www.zoho.com/mail/help/api/put-add-vacation-reply.html) and [PUT - Update display name and email](https://www.zoho.com/mail/help/api/put-update-display-name-and-email-address.html)

### Pattern 4: Signature Management with POST/GET/PUT/DELETE

**What:** Full CRUD operations on email signatures with alias assignment.

**When to use:** For managing user email signatures.

**Example:**
```go
// AddSignature creates a new email signature
func (mc *MailClient) AddSignature(ctx context.Context, sig *Signature) (string, error) {
    req := map[string]interface{}{
        "name":     sig.Name,
        "content":  sig.Content,
        "position": sig.Position, // 0 = below quoted, 1 = above quoted
    }

    if sig.AssignUsers != "" {
        req["assignUsers"] = sig.AssignUsers // comma-separated email addresses
    }

    body, err := json.Marshal(req)
    if err != nil {
        return "", fmt.Errorf("marshal request: %w", err)
    }

    path := "/api/accounts/signature"
    resp, err := mc.client.DoMail(ctx, http.MethodPost, path, bytes.NewReader(body))
    if err != nil {
        return "", fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
        return "", mc.parseErrorResponse(resp)
    }

    var sigResp struct {
        Status struct {
            Code        int    `json:"code"`
            Description string `json:"description"`
        } `json:"status"`
        Data struct {
            ID   string `json:"id"`
            Name string `json:"name"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&sigResp); err != nil {
        return "", fmt.Errorf("decode response: %w", err)
    }

    return sigResp.Data.ID, nil
}

// ListSignatures retrieves all signatures
func (mc *MailClient) ListSignatures(ctx context.Context) ([]Signature, error) {
    path := "/api/accounts/signature"
    resp, err := mc.client.DoMail(ctx, http.MethodGet, path, nil)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, mc.parseErrorResponse(resp)
    }

    var sigResp struct {
        Status struct {
            Code        int    `json:"code"`
            Description string `json:"description"`
        } `json:"status"`
        Data []Signature `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&sigResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    return sigResp.Data, nil
}

// Signature represents an email signature
type Signature struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Content     string `json:"content"`
    Position    int    `json:"position"`    // 0 or 1
    AssignUsers string `json:"assignUsers"` // comma-separated
}
```
**Source:** Derived from [Signature API](https://www.zoho.com/mail/help/api/signature-api.html) and [POST - Add user signature](https://www.zoho.com/mail/help/api/add-user-signature.html)

### Pattern 5: Organization-Level Spam Control

**What:** Manage allowlists/blocklists for email addresses, domains, and IPs via single enum-based endpoint.

**When to use:** For admin-level spam policy configuration.

**Example:**
```go
// internal/zoho/admin_policy_client.go (new file)
package zoho

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

// AdminPolicyClient wraps Client for organization-level policies
type AdminPolicyClient struct {
    client *Client
    zoid   string // Cached organization ID
}

// NewAdminPolicyClient creates policy client with cached zoid
func NewAdminPolicyClient(cfg *config.Config, tokenSource oauth2.TokenSource) (*AdminPolicyClient, error) {
    client, err := NewClient(cfg, tokenSource)
    if err != nil {
        return nil, err
    }

    apc := &AdminPolicyClient{client: client}

    // Fetch organization ID (same as AdminClient pattern)
    ctx := context.Background()
    zoid, err := apc.getOrganizationID(ctx)
    if err != nil {
        return nil, err
    }
    apc.zoid = zoid

    return apc, nil
}

// SpamCategory represents allowlist/blocklist types
type SpamCategory string

const (
    // Email address categories
    WhiteListEmail          SpamCategory = "whiteListEmail"
    SpamEmail               SpamCategory = "spamEmail"
    RejectEmail             SpamCategory = "rejectEmail"
    QuarantineEmail         SpamCategory = "quarantineEmail"
    TrustedEmail            SpamCategory = "trustedEmail"
    RecipientSpamEmail      SpamCategory = "recipientSpamEmail"
    RecipientRejectEmail    SpamCategory = "recipientRejectEmail"
    RecipientQuarantineEmail SpamCategory = "recipientQuarantineEmail"

    // Domain categories
    WhiteListDomain  SpamCategory = "whiteListDomain"
    SpamDomain       SpamCategory = "spamDomain"
    RejectDomain     SpamCategory = "rejectDomain"
    QuarantineDomain SpamCategory = "quarantineDomain"
    TrustedDomain    SpamCategory = "trustedDomain"
    SpamTLD          SpamCategory = "spamTLD"
    RejectTLD        SpamCategory = "rejectTLD"
    QuarantineTLD    SpamCategory = "quarantineTLD"

    // IP address categories
    WhiteListIP  SpamCategory = "whiteListIP"
    SpamIP       SpamCategory = "spamIP"
    RejectIP     SpamCategory = "rejectIP"
    QuarantineIP SpamCategory = "quarantineIP"
)

// UpdateSpamList updates allowlist/blocklist entries
func (apc *AdminPolicyClient) UpdateSpamList(ctx context.Context, category SpamCategory, values []string) error {
    req := map[string]interface{}{
        "spamCategory": string(category),
        "Value":        values,
    }

    body, err := json.Marshal(req)
    if err != nil {
        return fmt.Errorf("marshal request: %w", err)
    }

    path := fmt.Sprintf("/api/organization/%s/antispam/data", apc.zoid)
    resp, err := apc.client.DoMail(ctx, http.MethodPut, path, bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return apc.parseErrorResponse(resp)
    }

    var spamResp struct {
        Status struct {
            Code        int    `json:"code"`
            Description string `json:"description"`
        } `json:"status"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&spamResp); err != nil {
        return fmt.Errorf("decode response: %w", err)
    }

    if spamResp.Status.Code != 200 {
        return fmt.Errorf("API error: %s (code %d)",
            spamResp.Status.Description, spamResp.Status.Code)
    }

    return nil
}
```
**Source:** Derived from [PUT - Add/Update Spam Listing Info](https://www.zoho.com/mail/help/api/put-org-spam-info.html)

### Anti-Patterns to Avoid

- **Using multipart/form-data for attachments**: Zoho Mail rejects this format. Use `application/octet-stream` with binary body.
- **Inline attachment upload during send**: Upload attachments FIRST, get references, THEN send email. Not a single multipart operation.
- **Hardcoding mode strings**: Define constants for mode values to avoid typos (e.g., "addVacationReply" vs "addvacationreply").
- **Setting Content-Type globally**: Client.doRequest sets `application/json` by default. Override for binary uploads.
- **Assuming reply-all uses separate endpoint**: All reply variants use same endpoint with different `action` parameter.
- **Fetching signatures per-user in loops**: Signature API operates at account level, returns all signatures at once.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Multipart form encoding | Manual boundary generation | mime/multipart package | Handles RFC 2046 compliance, boundary uniqueness, header formatting |
| Attachment content detection | Custom MIME type guessing | http.DetectContentType or lookup tables | Standard library handles common types correctly |
| Date format for vacation | Custom time parsing | time.Parse with "01/02/2006 15:04:05" layout | Matches Zoho's MM/DD/YYYY HH:MM:SS format |
| Mode validation | Runtime string checks | Const enums + switch validation | Compile-time safety, clear error messages |
| Spam category validation | String matching | SpamCategory type + const enums | Type safety, IDE autocomplete |
| Signature position | Magic numbers | const (PositionBelow=0, PositionAbove=1) | Self-documenting code |

**Key insight:** Two-step attachment workflow is **fundamentally different** from traditional SMTP/email APIs. Must upload first, then reference. Content-Type gotcha (`application/octet-stream` not `multipart/form-data`) is a **documented community blocker**. Mode-based settings API requires careful request builder abstraction to avoid string typos.

## Common Pitfalls

### Pitfall 1: Wrong Content-Type for Attachment Upload

**What goes wrong:** 400 Bad Request or attachment upload fails silently.

**Why it happens:** Official documentation shows `Content-Type: application/json` in curl examples with `--form` flag, which is misleading. Community reports spending hours debugging this.

**How to avoid:**
- ALWAYS use `Content-Type: application/octet-stream` for binary file uploads
- Override Client.doRequest's default Content-Type for attachment endpoints
- Pass fileName as query parameter, NOT in multipart field
- Send raw file bytes in request body, NOT form-encoded

**Warning signs:** "Invalid request" errors, 400 responses, attachments not appearing in sent email

**Source:** [Zoho Mail API: How to Upload an Attachment](https://pebblesrox.wordpress.com/2021/03/28/zoho-mail-api-how-to-upload-an-attachment/)

### Pitfall 2: Forgetting Two-Step Upload Workflow

**What goes wrong:** Attempting to send email with file paths instead of storeName references, resulting in missing attachments.

**Why it happens:** Assuming inline attachment upload like traditional email APIs.

**How to avoid:**
- Upload attachments FIRST: `POST /messages/attachments` → returns storeName/path
- Store references in `[]AttachmentReference` slice
- Include references in send email request body
- Document two-step flow in CLI help text

**Warning signs:** Email sends successfully but has no attachments, "attachment not found" errors

### Pitfall 3: Mode String Typos in Settings Requests

**What goes wrong:** 400 errors with "invalid mode" or operation silently fails.

**Why it happens:** Settings API uses mode strings like "addVacationReply", "updateDisplayName", "displaynameemailupdate" (inconsistent casing).

**How to avoid:**
- Define const strings for all mode values
- Create typed request builders (AddVacationReplyRequest, UpdateDisplayNameRequest)
- Validate mode in client methods before API call
- Unit test request marshaling to catch typos early

**Warning signs:** 400 Bad Request, "mode not recognized", settings update silently ignored

### Pitfall 4: Reply/Forward Recipient Handling

**What goes wrong:** Reply-all doesn't include all original recipients, forward loses To addresses.

**Why it happens:** Assuming API auto-populates recipients based on action parameter.

**How to avoid:**
- For reply: set toAddress to original sender
- For replyall: set toAddress to original sender + ccAddress to original recipients
- For forward: require explicit toAddress from user (no auto-population)
- GetMessageMetadata first to extract original recipients

**Warning signs:** Reply-all only replies to sender, forward has empty recipient list

### Pitfall 5: Vacation Reply Date Format Mismatch

**What goes wrong:** Vacation auto-reply not activated, "invalid date" errors.

**Why it happens:** API requires `MM/DD/YYYY HH:MM:SS` format, not RFC3339 or ISO8601.

**How to avoid:**
- Use `time.Parse` with layout `"01/02/2006 15:04:05"` (Go's reference time)
- Validate date format in CLI command before API call
- Provide clear error message for wrong format
- Document format in help text

**Warning signs:** 400 errors, vacation reply not activating on expected date

### Pitfall 6: Signature Assignment vs Alias Confusion

**What goes wrong:** Signature not appearing for expected email address/alias.

**Why it happens:** Unclear whether assignUsers accepts aliases or only primary addresses.

**How to avoid:**
- Test with both primary addresses and aliases during implementation
- Document supported formats in CLI help
- Consider fetching account details first to validate addresses
- Provide clear feedback on which addresses were assigned

**Warning signs:** Signature created but not showing in compose UI, silent assignment failures

### Pitfall 7: Spam Category Enum Typos

**What goes wrong:** Blocklist/allowlist updates fail with "invalid category" errors.

**Why it happens:** 16 different category strings (whiteListEmail, spamDomain, quarantineIP, etc.) with inconsistent casing.

**How to avoid:**
- Define SpamCategory type with const enums
- Validate category before API call
- Provide autocomplete-friendly CLI flags (map user-friendly names to enum values)
- Unit test all category constants against API responses

**Warning signs:** 400 Bad Request, "spamCategory not recognized", updates silently ignored

### Pitfall 8: Missing OAuth Scopes for Write Operations

**What goes wrong:** 403 Forbidden when attempting to send email or update settings.

**Why it happens:** Phase 4 only requested READ scopes, write operations need additional permissions.

**How to avoid:**
- Request `ZohoMail.messages.ALL` or `ZohoMail.messages.CREATE` for sending
- Request `ZohoMail.accounts` for settings management
- Request `ZohoMail.organization.spam.ALL` or `.UPDATE` for spam controls
- Document all required scopes in auth flow
- Provide clear error message when scope insufficient

**Warning signs:** 403 responses for send/settings endpoints while read endpoints work

## Code Examples

Verified patterns from official sources:

### Send Email with Attachments (Two-Step Flow)

```go
// Example: Send email with file attachments
func sendEmailWithAttachments(mc *MailClient, to, subject, body string, files []string) error {
    ctx := context.Background()

    // Step 1: Upload attachments
    var attachments []AttachmentReference
    for _, filePath := range files {
        ref, err := mc.UploadAttachment(ctx, filePath)
        if err != nil {
            return fmt.Errorf("upload %s: %w", filePath, err)
        }
        attachments = append(attachments, *ref)
    }

    // Step 2: Send email with attachment references
    req := &SendEmailRequest{
        FromAddress: "sender@example.com", // Must be authenticated account
        ToAddress:   to,
        Subject:     subject,
        Content:     body,
        MailFormat:  "html",
        Attachments: attachments,
    }

    if err := mc.SendEmail(ctx, req); err != nil {
        return fmt.Errorf("send email: %w", err)
    }

    return nil
}
```
**Source:** [POST - Upload attachment](https://www.zoho.com/mail/help/api/post-upload-attachments.html) + [POST - Send an email with Attachments](https://www.zoho.com/mail/help/api/post-send-email-attachment.html)

### Reply All to Message

```go
// Example: Reply-all with quoted original message
func replyAllToMessage(mc *MailClient, folderID, messageID, replyBody string) error {
    ctx := context.Background()

    // Fetch original message metadata for recipients
    metadata, err := mc.GetMessageMetadata(ctx, folderID, messageID)
    if err != nil {
        return err
    }

    // Build recipient list (original sender + all recipients)
    ccAddresses := metadata.CcAddress
    // Add original To addresses to CC (if not already in CC)
    // Note: API response may have comma-separated strings or arrays

    req := &SendEmailRequest{
        FromAddress: "user@example.com",
        ToAddress:   metadata.FromAddress, // Original sender
        CcAddress:   ccAddresses,          // All original recipients
        Subject:     "Re: " + metadata.Subject,
        Content:     replyBody,
        MailFormat:  "html",
    }

    if err := mc.ReplyAllToEmail(ctx, messageID, req); err != nil {
        return err
    }

    return nil
}
```
**Source:** [POST - Send reply to an email](https://www.zoho.com/mail/help/api/post-reply-to-an-email.html)

### Configure Vacation Auto-Reply

```go
// Example: Set vacation reply for date range
func setVacationReply(mc *MailClient, start, end time.Time, message string) error {
    ctx := context.Background()

    vacation := &VacationReply{
        FromDate:   start.Format("01/02/2006 15:04:05"), // MM/DD/YYYY HH:MM:SS
        ToDate:     end.Format("01/02/2006 15:04:05"),
        SendingInt: 1440, // Reply interval in minutes (1440 = once per day)
        Subject:    "Out of Office",
        Content:    message,
        SendTo:     "all", // or "contacts", "org", "nonOrgAll", etc.
    }

    if err := mc.AddVacationReply(ctx, vacation); err != nil {
        return err
    }

    fmt.Fprintln(os.Stderr, "Vacation auto-reply enabled")
    return nil
}
```
**Source:** [PUT - Add vacation reply](https://www.zoho.com/mail/help/api/put-add-vacation-reply.html)

### Manage Email Signatures

```go
// Example: Create signature and assign to aliases
func createSignature(mc *MailClient, name, content string, aliases []string) (string, error) {
    ctx := context.Background()

    sig := &Signature{
        Name:        name,
        Content:     content,
        Position:    0, // Below quoted content
        AssignUsers: strings.Join(aliases, ","),
    }

    sigID, err := mc.AddSignature(ctx, sig)
    if err != nil {
        return "", err
    }

    fmt.Fprintf(os.Stderr, "Created signature %q (ID: %s)\n", name, sigID)
    return sigID, nil
}
```
**Source:** [POST - Add user signature](https://www.zoho.com/mail/help/api/add-user-signature.html)

### Update Spam Control Lists

```go
// Example: Add domains to blocklist
func blockDomains(apc *AdminPolicyClient, domains []string) error {
    ctx := context.Background()

    // Add to spam domain list (quarantine)
    if err := apc.UpdateSpamList(ctx, QuarantineDomain, domains); err != nil {
        return fmt.Errorf("update spam list: %w", err)
    }

    fmt.Fprintf(os.Stderr, "Added %d domains to quarantine list\n", len(domains))
    return nil
}

// Example: Add email addresses to allowlist
func allowEmails(apc *AdminPolicyClient, emails []string) error {
    ctx := context.Background()

    if err := apc.UpdateSpamList(ctx, WhiteListEmail, emails); err != nil {
        return err
    }

    fmt.Fprintf(os.Stderr, "Added %d emails to allowlist\n", len(emails))
    return nil
}
```
**Source:** [PUT - Add/Update Spam Listing Info](https://www.zoho.com/mail/help/api/put-org-spam-info.html)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Inline multipart attachments | Two-step upload (file store first) | Current API design | Simpler API but requires workflow change |
| Dedicated reply/forward endpoints | Action parameter routing | Current API design | Single endpoint, less REST-ful but functional |
| Individual setting endpoints | Mode-based multiplexing | Current API design | Reduces endpoint count, requires careful mode management |
| Separate allowlist/blocklist APIs | Single enum-based spam control | Current API design | Unified interface, 16 category enums to manage |

**Deprecated/outdated:**
- **Multipart/form-data for attachments**: API explicitly rejects this despite official curl examples being misleading.
- **Auto-recipient population for reply/forward**: API requires explicit recipient specification in all cases.

## Open Questions

1. **Does Zoho support batch attachment upload?**
   - What we know: Upload endpoint accepts one file at a time with fileName query param
   - What's unclear: Whether multiple files can be uploaded in single multipart request
   - Recommendation: Implement sequential uploads in parallel with rate limiting; test multipart during implementation

2. **How are signature aliases resolved?**
   - What we know: assignUsers accepts comma-separated email addresses
   - What's unclear: Whether this supports aliases or only primary addresses, how assignment failures are reported
   - Recommendation: Test with both primary and alias addresses; document behavior in client method comments

3. **What are exact forwarding restriction capabilities?**
   - What we know: GET endpoint retrieves forward restrictions, returns restrictionID/name/count
   - What's unclear: How to CREATE/UPDATE/DELETE restrictions (only list API documented)
   - Recommendation: Research Admin Console UI behavior; may need additional API endpoints not yet documented

4. **Are there limits on vacation reply frequency?**
   - What we know: sendingInt parameter controls reply interval
   - What's unclear: Minimum/maximum allowed values, whether 0 means "reply once only"
   - Recommendation: Test with various interval values; document observed limits

5. **Can we retrieve current spam control lists?**
   - What we know: PUT endpoint updates lists with spamCategory + values
   - What's unclear: Whether GET endpoint exists to retrieve current lists, or if we can only overwrite
   - Recommendation: Test GET on same endpoint path; may need to track state client-side if no retrieval API

6. **What happens to pending scheduled emails on send failure?**
   - What we know: isSchedule parameter enables deferred sending
   - What's unclear: Error handling for scheduled sends, how to list/cancel scheduled emails
   - Recommendation: v1 skip scheduling feature; v2 research scheduled email management APIs

7. **Do attachment uploads persist across sessions?**
   - What we know: Upload returns storeName/path references
   - What's unclear: How long uploaded files remain in file store, whether they're cleaned up if email not sent
   - Recommendation: Upload and send immediately; avoid long delays between steps

## Sources

### Primary (HIGH confidence)
- [POST - Send an Email](https://www.zoho.com/mail/help/api/post-send-an-email.html) - Send email endpoint
- [POST - Upload attachment](https://www.zoho.com/mail/help/api/post-upload-attachments.html) - Attachment upload endpoint
- [POST - Send reply to an email](https://www.zoho.com/mail/help/api/post-reply-to-an-email.html) - Reply operations
- [PUT - Add vacation reply](https://www.zoho.com/mail/help/api/put-add-vacation-reply.html) - Vacation auto-reply
- [Signature API](https://www.zoho.com/mail/help/api/signature-api.html) - Signature management overview
- [POST - Add user signature](https://www.zoho.com/mail/help/api/add-user-signature.html) - Create signature
- [GET - Get User Signature](https://www.zoho.com/mail/help/api/get-user-signature.html) - List signatures
- [PUT - Update display name and email](https://www.zoho.com/mail/help/api/put-update-display-name-and-email-address.html) - Display name settings
- [PUT - Add send mail details](https://www.zoho.com/mail/help/api/put-to-add-send-mail-details.html) - Sender aliases
- [PUT - Add/Update Spam Listing Info](https://www.zoho.com/mail/help/api/put-org-spam-info.html) - Spam control
- [Logs API](https://www.zoho.com/mail/help/api/logs-api.html) - Delivery logs overview
- [GET - Forward restrictions](https://www.zoho.com/mail/help/api/get-forward-restrictions.html) - Forwarding policy

### Secondary (MEDIUM confidence)
- [Zoho Mail API: How to Upload an Attachment](https://pebblesrox.wordpress.com/2021/03/28/zoho-mail-api-how-to-upload-an-attachment/) - Community-documented Content-Type gotcha
- [Email Forwarding Rules](https://www.zoho.com/mail/help/adminconsole/email-forwarding-rules.html) - UI documentation for forwarding
- [Spam Control Lists](https://www.zoho.com/mail/help/adminconsole/spam-control-lists.html) - UI documentation for allowlist/blocklist
- [Audit logs in Admin Reports](https://www.zoho.com/mail/help/adminconsole/log-reports.html) - UI documentation for delivery logs

### Tertiary (LOW confidence - requires verification)
- Community discussion: Multipart/form-data rejection - needs direct API testing
- Reply-all recipient handling - unclear from docs, needs implementation testing
- Forwarding restriction CRUD operations - only GET documented, CREATE/UPDATE/DELETE need investigation

## Metadata

**Confidence breakdown:**
- Send operations: MEDIUM-HIGH - API well-documented, two-step workflow confirmed by community
- Attachment upload: HIGH - Community blog verified Content-Type requirement, workflow clear
- Signatures: MEDIUM - API documented, alias assignment needs testing
- Vacation reply: HIGH - API documented with all parameters
- Display name/aliases: MEDIUM - Multiple endpoints exist, unclear which is canonical
- Spam control: MEDIUM - PUT endpoint documented, unclear if GET exists for retrieval
- Forwarding/retention: LOW - Only list endpoint documented, update/create operations unclear
- Delivery logs: MEDIUM - API exists but parameters/filtering options sparse in docs

**Research date:** 2026-02-14
**Valid until:** 2026-03-16 (30 days - Zoho Mail API is stable, infrequent breaking changes expected)
