# Phase 3: Admin -- Domains & Audit - Research

**Researched:** 2026-02-14
**Domain:** Zoho Mail API (domain management and audit/security operations)
**Confidence:** MEDIUM

## Summary

Phase 3 implements domain management with DNS verification and audit/security log access through the **Zoho Mail API**. The API provides RESTful endpoints for domain CRUD operations, DNS record verification (TXT, CNAME, HTML), and comprehensive audit logging (admin actions, login history, SMTP logs). All operations continue using the existing Mail API infrastructure established in Phase 2.

**Critical Finding:** Domain and audit operations use **different pagination patterns**. Domain APIs use simple offset pagination (start/limit) like users/groups, but audit/log APIs use **cursor-based pagination** (scrollId, pageKey/prevKey) with bidirectional traversal. Login history data is limited to 90 days retention.

**Key architectural challenge:** The audit APIs use **Unix milliseconds timestamps** for date filtering (fromTime/toTime), requiring conversion between Go's `time.Time` and int64 milliseconds. SMTP logs use a complex dual-cursor pagination system (pageKey for forward, prevKey for backward) with multiple search criteria options.

**Primary recommendation:** Extend the existing AdminClient with domain and audit methods, implement cursor-based pagination abstraction alongside the existing PageIterator, add time conversion helpers for millisecond timestamps, and handle the diverse response formats for different log types.

## Standard Stack

### Core (Already in go.mod)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| encoding/json | stdlib | JSON marshaling | API request/response format |
| time | stdlib | Date/time handling | Unix millisecond conversions for audit logs |
| golang.org/x/oauth2 | v0.35.0 | OAuth2 auth flow | Required for Zoho API authentication |
| golang.org/x/time/rate | v0.14.0 | Rate limiting | Already used for 30 req/min budget |
| github.com/cenkalti/backoff/v4 | v4.3.0 | Exponential backoff retry | Already handles 429 responses |

### Supporting (Already in go.mod)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/alecthomas/kong | v1.14.0 | CLI command parsing | Struct tags for all commands |
| github.com/charmbracelet/lipgloss/v2 | v2.0.0-beta1 | Rich terminal output | Styled output formatting |
| github.com/rodaine/table | v1.3.0 | Table rendering | List output for domains and logs |

### New Dependencies Required
**None.** All required libraries are already in the project from Phases 1 and 2. The standard library's `time` package handles millisecond conversions natively since Go 1.17.

**Installation:**
No new dependencies needed. Existing stack sufficient.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── zoho/
│   ├── client.go           # Existing: region-aware HTTP client
│   ├── ratelimit.go        # Existing: 429 retry logic
│   ├── admin_client.go     # EXTEND: add domain & audit methods
│   ├── pagination.go       # EXTEND: add cursor-based iterator
│   ├── types.go            # EXTEND: add domain & audit types
│   └── timeutil.go         # NEW: millisecond conversion helpers
├── cli/
│   ├── admin_domains.go    # NEW: domain commands (list, get, add, verify, update)
│   └── admin_audit.go      # NEW: audit commands (logs, sessions, security)
└── output/
    ├── formatter.go        # Existing: Formatter interface
    └── table.go            # Existing: table rendering
```

### Pattern 1: Extend AdminClient with Domain Methods
**What:** Add domain-specific methods to the existing AdminClient struct, reusing the cached zoid.
**When to use:** For all domain operations to maintain consistency with Phase 2 user/group patterns.
**Example:**
```go
// internal/zoho/admin_client.go (extend existing struct)

// ListDomains fetches all domains in the organization
func (ac *AdminClient) ListDomains(ctx context.Context) ([]Domain, error) {
    path := fmt.Sprintf("/api/organization/%d/domains", ac.zoid)
    resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, ac.parseErrorResponse(resp)
    }

    var domainResp DomainListResponse
    if err := json.NewDecoder(resp.Body).Decode(&domainResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    if domainResp.Status.Code != 200 {
        return nil, fmt.Errorf("API error: %s (code %d)",
            domainResp.Status.Description, domainResp.Status.Code)
    }

    return domainResp.Data, nil
}

// AddDomain adds a new domain to the organization
func (ac *AdminClient) AddDomain(ctx context.Context, domainName string) (*Domain, error) {
    path := fmt.Sprintf("/api/organization/%d/domains", ac.zoid)

    req := AddDomainRequest{DomainName: domainName}
    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }

    resp, err := ac.client.Do(ctx, http.MethodPost, path, bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        return nil, ac.parseErrorResponse(resp)
    }

    var domainResp DomainDetailResponse
    if err := json.NewDecoder(resp.Body).Decode(&domainResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    return &domainResp.Data, nil
}

// VerifyDomain triggers domain verification using specified method
func (ac *AdminClient) VerifyDomain(ctx context.Context, domainName, method string) error {
    path := fmt.Sprintf("/api/organization/%d/domains/%s", ac.zoid, domainName)

    req := VerifyDomainRequest{Mode: method}
    body, err := json.Marshal(req)
    if err != nil {
        return fmt.Errorf("marshal request: %w", err)
    }

    resp, err := ac.client.Do(ctx, http.MethodPut, path, bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return ac.parseErrorResponse(resp)
    }

    return nil
}
```

### Pattern 2: Cursor-Based Pagination Iterator
**What:** New pagination abstraction for cursor-based APIs (audit logs, login history) that tracks scroll/page keys.
**When to use:** For all log and audit list operations that use cursor pagination instead of offset pagination.
**Example:**
```go
// internal/zoho/pagination.go (add alongside existing PageIterator)

// CursorIterator provides cursor-based pagination for Zoho log APIs
type CursorIterator[T any] struct {
    fetchFunc   func(cursor string) ([]T, string, error) // Returns items, nextCursor, error
    currentPage []T
    cursor      string
    done        bool
    index       int
}

// NewCursorIterator creates a cursor-based iterator
func NewCursorIterator[T any](fetchFunc func(cursor string) ([]T, string, error)) *CursorIterator[T] {
    return &CursorIterator[T]{
        fetchFunc: fetchFunc,
        cursor:    "", // Empty for first request
        done:      false,
    }
}

// FetchAll retrieves all pages until cursor is exhausted
func (ci *CursorIterator[T]) FetchAll() ([]T, error) {
    var results []T

    for !ci.done {
        items, nextCursor, err := ci.fetchFunc(ci.cursor)
        if err != nil {
            return nil, err
        }

        results = append(results, items...)

        if nextCursor == "" || len(items) == 0 {
            ci.done = true
            break
        }

        ci.cursor = nextCursor
    }

    return results, nil
}

// Next returns the next item (for streaming iteration)
func (ci *CursorIterator[T]) Next() (T, bool, error) {
    if ci.index < len(ci.currentPage) {
        item := ci.currentPage[ci.index]
        ci.index++
        return item, true, nil
    }

    if ci.done {
        var zero T
        return zero, false, nil
    }

    // Fetch next page
    items, nextCursor, err := ci.fetchFunc(ci.cursor)
    if err != nil {
        var zero T
        return zero, false, err
    }

    if len(items) == 0 || nextCursor == "" {
        ci.done = true
        var zero T
        return zero, false, nil
    }

    ci.currentPage = items
    ci.cursor = nextCursor
    ci.index = 0

    return ci.Next()
}
```

### Pattern 3: Time Conversion Helpers
**What:** Utility functions for converting between Go's `time.Time` and Unix milliseconds (int64).
**When to use:** For all audit and log API calls that require millisecond timestamps.
**Example:**
```go
// internal/zoho/timeutil.go (new file)
package zoho

import "time"

// ToUnixMillis converts time.Time to Unix milliseconds (int64)
func ToUnixMillis(t time.Time) int64 {
    return t.UnixMilli()
}

// FromUnixMillis converts Unix milliseconds (int64) to time.Time
func FromUnixMillis(ms int64) time.Time {
    return time.UnixMilli(ms)
}

// FormatMillisTimestamp formats a millisecond timestamp for display
func FormatMillisTimestamp(ms int64) string {
    return FromUnixMillis(ms).Format(time.RFC3339)
}
```

### Pattern 4: Response Type Definitions for Domain & Audit
**What:** Define Go structs matching API responses with proper JSON tags for domains and audit logs.
**When to use:** For all domain and audit API request/response structures.
**Example:**
```go
// internal/zoho/types.go (extend existing file)

// Domain represents a Zoho Mail domain
type Domain struct {
    DomainName              string   `json:"domainName"`
    DomainID                string   `json:"domainId"`
    VerificationStatus      bool     `json:"verificationStatus"`
    DKIMStatus              bool     `json:"dkimstatus"`
    SPFStatus               bool     `json:"spfstatus"`
    MXStatus                string   `json:"mxstatus"`
    VerifiedDate            int64    `json:"verifiedDate"`
    MailHostingEnabled      bool     `json:"mailHostingEnabled"`
    IsDomainAlias           bool     `json:"isDomainAlias"`
    IsExpired               bool     `json:"isExpired"`
    Primary                 bool     `json:"primary"`
    DKIMDetailList          []DKIM   `json:"dkimDetailList"`
    CNAMEVerificationCode   string   `json:"CNAMEVerificationCode"`
    HTMLVerificationCode    string   `json:"HTMLVerificationCode"`
}

// DomainListResponse from GET /api/organization/{zoid}/domains
type DomainListResponse struct {
    Status APIStatus `json:"status"`
    Data   []Domain  `json:"data"`
}

// DomainDetailResponse from POST/GET single domain
type DomainDetailResponse struct {
    Status APIStatus `json:"status"`
    Data   Domain    `json:"data"`
}

// AddDomainRequest for POST /api/organization/{zoid}/domains
type AddDomainRequest struct {
    DomainName string `json:"domainName"`
}

// VerifyDomainRequest for PUT /api/organization/{zoid}/domains/{name}
type VerifyDomainRequest struct {
    Mode string `json:"mode"` // verifyDomainByTXT, verifyDomainByCName, verifyDomainByHTML
}

// AuditLog represents an admin action audit record
type AuditLog struct {
    SubCategory    string                 `json:"subCategory"`
    Data           map[string]interface{} `json:"data"`
    Type           string                 `json:"type"`
    RequestTime    int64                  `json:"requestTime"` // Unix milliseconds
    PerformedBy    string                 `json:"performedBy"`
    AuditLogType   string                 `json:"auditLogType"`
    ClientIP       string                 `json:"clientIp"`
    MainCategory   string                 `json:"mainCategory"`
    OperationType  string                 `json:"operationType"`
    PerformedOn    string                 `json:"performedOn"`
    Category       string                 `json:"category"`
    Operation      string                 `json:"operation"`
}

// AuditLogResponse from GET /api/organization/{zoid}/activity
type AuditLogResponse struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data struct {
        Audit         []AuditLog `json:"audit"`
        LastIndexTime string     `json:"lastIndexTime"`
        LastEntityID  string     `json:"lastEntityId"`
    } `json:"data"`
}

// LoginHistoryEntry represents a login event
type LoginHistoryEntry struct {
    UserID       int64  `json:"userId"`
    EmailAddress string `json:"emailAddress"`
    IPAddress    string `json:"ipAddress"`
    LoginTime    int64  `json:"loginTime"` // Unix milliseconds
    Status       string `json:"status"`
    AccessType   string `json:"accessType"` // web, mobile
    ClientInfo   string `json:"clientInfo"`
}

// LoginHistoryResponse from GET /api/organization/{zoid}/accounts/reports/loginHistory
type LoginHistoryResponse struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data struct {
        LoginHistory []LoginHistoryEntry `json:"loginHistory"`
        ScrollID     string              `json:"scrollId"`
    } `json:"data"`
}

// SMTPLogEntry represents an SMTP transaction log
type SMTPLogEntry struct {
    MessageID     string   `json:"messageId"`
    FromAddress   string   `json:"fromAddr"`
    ToAddresses   []string `json:"toAddr"`
    Subject       string   `json:"subject"`
    TransactionID string   `json:"transactionId"`
    Timestamp     int64    `json:"timestamp"`
    Status        string   `json:"status"`
}

// SMTPLogResponse from POST /api/organization/{zoid}/smtplogs
type SMTPLogResponse struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data struct {
        HasNext       bool           `json:"hnxt"`
        HasPrevious   bool           `json:"hasPrevious"`
        PageKey       string         `json:"pagekey"`
        PagePrevKey   string         `json:"pagePrevKey"`
        Response      []SMTPLogEntry `json:"response"`
    } `json:"data"`
}
```

### Pattern 5: Audit Log Fetch with Date Range
**What:** Helper method wrapping cursor pagination for audit logs with time range filtering.
**When to use:** For CLI commands that need to fetch audit logs within a date range.
**Example:**
```go
// GetAuditLogs fetches audit logs within a time range
func (ac *AdminClient) GetAuditLogs(ctx context.Context, startTime, endTime time.Time, searchKey string) ([]AuditLog, error) {
    startMillis := ToUnixMillis(startTime)
    endMillis := ToUnixMillis(endTime)

    var allLogs []AuditLog
    lastEntityID := ""
    lastIndexTime := ""

    for {
        path := fmt.Sprintf("/api/organization/%d/activity?startTime=%d&endTime=%d&limit=100",
            ac.zoid, startMillis, endMillis)

        if searchKey != "" {
            path += fmt.Sprintf("&searchKey=%s", url.QueryEscape(searchKey))
        }
        if lastEntityID != "" {
            path += fmt.Sprintf("&lastEntityId=%s&lastIndexTime=%s",
                url.QueryEscape(lastEntityID), url.QueryEscape(lastIndexTime))
        }

        resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
        if err != nil {
            return nil, fmt.Errorf("request failed: %w", err)
        }
        defer resp.Body.Close()

        var auditResp AuditLogResponse
        if err := json.NewDecoder(resp.Body).Decode(&auditResp); err != nil {
            return nil, fmt.Errorf("decode response: %w", err)
        }

        allLogs = append(allLogs, auditResp.Data.Audit...)

        // Check if more results exist
        if auditResp.Data.LastEntityID == "" || len(auditResp.Data.Audit) == 0 {
            break
        }

        lastEntityID = auditResp.Data.LastEntityID
        lastIndexTime = auditResp.Data.LastIndexTime
    }

    return allLogs, nil
}
```

### Anti-Patterns to Avoid
- **Hardcoding verification methods:** Domain verification supports three methods (TXT, CNAME, HTML); always validate the method parameter against allowed values.
- **Ignoring 90-day retention limit:** Login history API only retains 90 days of data; CLI should validate date ranges before making requests.
- **Manual time arithmetic:** Use `time.Time` methods and the timeutil helpers; don't manipulate milliseconds manually.
- **Fetching all logs without limits:** Audit logs can be large; implement batch size limits and streaming iteration, not full FetchAll() by default.
- **Confusing SMTP log pagination:** SMTP logs use dual cursors (pageKey/prevKey); don't assume standard cursor pattern.
- **Mixing domain operations:** Domain endpoints use different `mode` parameters for different operations on the same URL; always set mode correctly.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Time conversions | Manual millisecond math | time.UnixMilli() / t.UnixMilli() (Go 1.17+) | Built-in since Go 1.17, handles edge cases, timezone-aware |
| Cursor pagination | Custom state tracking | CursorIterator pattern (see above) | Encapsulates cursor management, bidirectional traversal, reusable |
| Date range parsing | Custom parser for CLI flags | time.Parse with RFC3339 or custom format | Standard library handles timezones, validates format |
| DNS record formatting | String concatenation for verification codes | Display as-is from API response | API returns properly formatted TXT/CNAME values ready for DNS |
| Log filtering | Client-side filtering after fetch | API searchKey parameter where available | Reduces bandwidth, faster queries, server-side optimization |
| Retry logic | Custom backoff for audit APIs | Existing RateLimitTransport with backoff/v4 | Already handles 429s, exponential backoff, jitter |

**Key insight:** Zoho's audit APIs have **90-day retention** and **diverse pagination patterns**. Don't assume all list operations work the same—domain lists use offset pagination, audit logs use cursor pagination, and SMTP logs use bidirectional cursors.

## Common Pitfalls

### Pitfall 1: 90-Day Login History Retention Limit
**What goes wrong:** Requesting login history older than 90 days returns empty results or errors.
**Why it happens:** Zoho only retains login history for 90 days; this is a hard API limitation.
**How to avoid:**
- Validate date ranges before making API requests
- Show clear error message when user requests data older than 90 days
- Document the 90-day limit in CLI help text
**Warning signs:** Empty responses for valid date ranges; "no data available" errors.

### Pitfall 2: Millisecond Timestamp Conversion Errors
**What goes wrong:** Incorrect time zone handling or precision loss when converting between time.Time and int64 milliseconds.
**Why it happens:** Mixing seconds and milliseconds, or not handling UTC/local timezone conversions.
**How to avoid:**
- Always use `time.UnixMilli()` and `t.UnixMilli()` (Go 1.17+)
- Store and transmit timestamps in UTC
- Convert to local timezone only for display
- Use the timeutil helpers for consistency
**Warning signs:** Times off by 1000x (seconds vs milliseconds), timezone-related discrepancies.

### Pitfall 3: Domain Verification Mode Confusion
**What goes wrong:** Verification fails because wrong `mode` value sent in PUT request.
**Why it happens:** Domain verification, enabling hosting, setting primary, and updating DKIM all use PUT to the same endpoint with different `mode` values.
**How to avoid:**
- Define separate request structs for each operation (VerifyDomainRequest, EnableHostingRequest, etc.)
- Validate mode parameter against allowed values before request
- Document which mode values are valid for each operation
**Warning signs:** 400 Bad Request errors on domain PUT operations; "invalid mode" in error messages.

### Pitfall 4: SMTP Log Pagination Complexity
**What goes wrong:** Infinite loops or missing results when paginating SMTP logs with dual-cursor system.
**Why it happens:** SMTP logs use both `pageKey` (forward) and `prevKey` (backward) with boolean flags `isNext` and `isPrevious`.
**How to avoid:**
- Initial request: set `isNext=false`, `isPrevious=false`, both keys empty
- Forward pagination: set `isNext=true`, use `pageKey` from previous response
- Backward pagination: set `isPrevious=true`, use `pagePrevKey` from previous response
- Check `hasNext` and `hasPrevious` flags to detect end-of-data
**Warning signs:** Duplicate log entries, skipped entries, pagination loops.

### Pitfall 5: Audit Log Search Criteria Misuse
**What goes wrong:** Search returns empty results despite matching data existing.
**Why it happens:** `searchKey` parameter behavior varies by API endpoint (audit vs login history vs SMTP).
**How to avoid:**
- For audit logs: searchKey filters by category, sub-category, operation type, performer, subject
- For login history: Use `mode` parameter for filter type (loginActivity, failedLoginActivity, etc.)
- For SMTP logs: Use `searchCriteria` (messageId, fromAddr, toAddr) with matching `searchKey`
- Don't assume searchKey works the same across all log APIs
**Warning signs:** Empty results when data should exist; unclear filtering behavior.

### Pitfall 6: Domain List vs Detail Response Differences
**What goes wrong:** Accessing fields that only exist in detail responses, causing nil pointer errors.
**Why it happens:** GET /domains (list) returns fewer fields than GET /domains/{name} (detail).
**How to avoid:**
- List response has basic fields: domainName, domainId, verificationStatus, verification codes
- Detail response adds: DKIM details, catch-all settings, hosting status
- Check which fields are needed before choosing list vs detail endpoint
- Use separate type definitions if field sets diverge significantly
**Warning signs:** Nil pointer errors when accessing domain fields; missing data in CLI output.

### Pitfall 7: Missing OAuth Scopes for Logs
**What goes wrong:** 403 Forbidden errors when fetching audit or SMTP logs.
**Why it happens:** Different log types require different OAuth scopes.
**How to avoid:**
- Audit logs: `ZohoMail.organization.audit.READ` or `.ALL`
- Login history: `ZohoMail.organization.accounts.READ` or `.ALL`
- SMTP logs: `ZohoMail.partner.organization.READ` or `.ALL`
- Request all necessary scopes during Phase 3 implementation
- Document scope requirements in auth/scopes.go
**Warning signs:** 403 responses for log endpoints; "insufficient scope" errors.

## Code Examples

Verified patterns from official sources:

### Fetch All Domains
```go
// GET /api/organization/{zoid}/domains
// Required scope: ZohoMail.organization.domains.READ or .ALL
// Returns: DomainListResponse with array of Domain objects
// No pagination - returns all domains in single response

func (ac *AdminClient) ListDomains(ctx context.Context) ([]Domain, error) {
    path := fmt.Sprintf("/api/organization/%d/domains", ac.zoid)
    resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, ac.parseErrorResponse(resp)
    }

    var domainResp DomainListResponse
    if err := json.NewDecoder(resp.Body).Decode(&domainResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    if domainResp.Status.Code != 200 {
        return nil, fmt.Errorf("API error: %s (code %d)",
            domainResp.Status.Description, domainResp.Status.Code)
    }

    return domainResp.Data, nil
}
```

### Add Domain
```go
// POST /api/organization/{zoid}/domains
// Required scope: ZohoMail.organization.domains.CREATE or .ALL
// Request body: {"domainName": "example.com"}
// Response: HTTP 201 Created with domain details including verification codes

func (ac *AdminClient) AddDomain(ctx context.Context, domainName string) (*Domain, error) {
    path := fmt.Sprintf("/api/organization/%d/domains", ac.zoid)

    req := AddDomainRequest{DomainName: domainName}
    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }

    resp, err := ac.client.Do(ctx, http.MethodPost, path, bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        return nil, ac.parseErrorResponse(resp)
    }

    var domainResp DomainDetailResponse
    if err := json.NewDecoder(resp.Body).Decode(&domainResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    return &domainResp.Data, nil
}
```

### Verify Domain
```go
// PUT /api/organization/{zoid}/domains/{domainName}
// Required scope: ZohoMail.organization.domains.UPDATE or .ALL
// Request body: {"mode": "verifyDomainByTXT"} or "verifyDomainByCName" or "verifyDomainByHTML"
// Response: HTTP 200 OK with {"status": {"code": 200}, "data": {"status": true}}

func (ac *AdminClient) VerifyDomain(ctx context.Context, domainName, method string) error {
    // Validate method parameter
    validMethods := map[string]bool{
        "verifyDomainByTXT":   true,
        "verifyDomainByCName": true,
        "verifyDomainByHTML":  true,
    }
    if !validMethods[method] {
        return fmt.Errorf("invalid verification method: %s", method)
    }

    path := fmt.Sprintf("/api/organization/%d/domains/%s", ac.zoid, domainName)

    req := VerifyDomainRequest{Mode: method}
    body, err := json.Marshal(req)
    if err != nil {
        return fmt.Errorf("marshal request: %w", err)
    }

    resp, err := ac.client.Do(ctx, http.MethodPut, path, bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return ac.parseErrorResponse(resp)
    }

    return nil
}
```

### Fetch Audit Logs with Cursor Pagination
```go
// GET /api/organization/{zoid}/activity?startTime={ms}&endTime={ms}&limit={n}&lastEntityId={id}&lastIndexTime={time}
// Required scope: ZohoMail.organization.audit.READ or .ALL
// Pagination: cursor-based using lastEntityId and lastIndexTime from response
// Times in Unix milliseconds

func (ac *AdminClient) GetAuditLogs(ctx context.Context, startTime, endTime time.Time) ([]AuditLog, error) {
    startMillis := startTime.UnixMilli()
    endMillis := endTime.UnixMilli()

    var allLogs []AuditLog
    lastEntityID := ""
    lastIndexTime := ""

    for {
        path := fmt.Sprintf("/api/organization/%d/activity?startTime=%d&endTime=%d&limit=100",
            ac.zoid, startMillis, endMillis)

        if lastEntityID != "" {
            path += fmt.Sprintf("&lastEntityId=%s&lastIndexTime=%s", lastEntityID, lastIndexTime)
        }

        resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
        if err != nil {
            return nil, fmt.Errorf("request failed: %w", err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            return nil, ac.parseErrorResponse(resp)
        }

        var auditResp AuditLogResponse
        if err := json.NewDecoder(resp.Body).Decode(&auditResp); err != nil {
            return nil, fmt.Errorf("decode response: %w", err)
        }

        if auditResp.Status.Code != 200 {
            return nil, fmt.Errorf("API error: %s (code %d)",
                auditResp.Status.Description, auditResp.Status.Code)
        }

        allLogs = append(allLogs, auditResp.Data.Audit...)

        // Check if more results exist
        if auditResp.Data.LastEntityID == "" || len(auditResp.Data.Audit) == 0 {
            break
        }

        lastEntityID = auditResp.Data.LastEntityID
        lastIndexTime = auditResp.Data.LastIndexTime
    }

    return allLogs, nil
}
```

### Fetch Login History with Scroll Pagination
```go
// GET /api/organization/{zoid}/accounts/reports/loginHistory?mode={mode}&fromTime={ms}&toTime={ms}&batchSize={n}&scrollId={id}
// Required scope: ZohoMail.organization.accounts.READ or .ALL
// Pagination: scrollId cursor, 90-day retention limit
// mode: loginActivity, failedLoginActivity, protocolLoginActivity, failedProtocolLoginActivity

func (ac *AdminClient) GetLoginHistory(ctx context.Context, mode string, startTime, endTime time.Time) ([]LoginHistoryEntry, error) {
    // Validate 90-day limit
    now := time.Now()
    ninetyDaysAgo := now.AddDate(0, 0, -90)
    if startTime.Before(ninetyDaysAgo) {
        return nil, fmt.Errorf("login history only available for last 90 days")
    }

    startMillis := startTime.UnixMilli()
    endMillis := endTime.UnixMilli()

    var allEntries []LoginHistoryEntry
    scrollID := ""

    for {
        path := fmt.Sprintf("/api/organization/%d/accounts/reports/loginHistory?mode=%s&fromTime=%d&toTime=%d&batchSize=100",
            ac.zoid, mode, startMillis, endMillis)

        if scrollID != "" {
            path += fmt.Sprintf("&scrollId=%s", scrollID)
        }

        resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
        if err != nil {
            return nil, fmt.Errorf("request failed: %w", err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            return nil, ac.parseErrorResponse(resp)
        }

        var loginResp LoginHistoryResponse
        if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
            return nil, fmt.Errorf("decode response: %w", err)
        }

        allEntries = append(allEntries, loginResp.Data.LoginHistory...)

        // Check if more results exist
        if loginResp.Data.ScrollID == "" || len(loginResp.Data.LoginHistory) == 0 {
            break
        }

        scrollID = loginResp.Data.ScrollID
    }

    return allEntries, nil
}
```

### Fetch SMTP Logs with Dual-Cursor Pagination
```go
// POST /api/organization/{zoid}/smtplogs
// Required scope: ZohoMail.partner.organization.READ or .ALL
// Pagination: dual-cursor (pageKey forward, prevKey backward)
// searchCriteria: messageId, fromAddr, toAddr

func (ac *AdminClient) GetSMTPLogs(ctx context.Context, startTime, endTime time.Time, searchCriteria, searchKey string) ([]SMTPLogEntry, error) {
    startMillis := startTime.UnixMilli()
    endMillis := endTime.UnixMilli()

    var allLogs []SMTPLogEntry

    // Initial request
    req := map[string]interface{}{
        "fromDateTime":   startMillis,
        "toDateTime":     endMillis,
        "searchCriteria": searchCriteria,
        "searchKey":      searchKey,
        "limit":          100,
        "isNext":         false,
        "isPrevious":     false,
        "pageKey":        "",
        "prevKey":        "",
    }

    for {
        body, err := json.Marshal(req)
        if err != nil {
            return nil, fmt.Errorf("marshal request: %w", err)
        }

        path := fmt.Sprintf("/api/organization/%d/smtplogs", ac.zoid)
        resp, err := ac.client.Do(ctx, http.MethodPost, path, bytes.NewReader(body))
        if err != nil {
            return nil, fmt.Errorf("request failed: %w", err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            return nil, ac.parseErrorResponse(resp)
        }

        var smtpResp SMTPLogResponse
        if err := json.NewDecoder(resp.Body).Decode(&smtpResp); err != nil {
            return nil, fmt.Errorf("decode response: %w", err)
        }

        allLogs = append(allLogs, smtpResp.Data.Response...)

        // Check if more results exist
        if !smtpResp.Data.HasNext || len(smtpResp.Data.Response) == 0 {
            break
        }

        // Prepare next request
        req["isNext"] = true
        req["pageKey"] = smtpResp.Data.PageKey
    }

    return allLogs, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| time.Unix(seconds, 0) | time.UnixMilli(ms) | Go 1.17 (2021) | Native millisecond support eliminates manual conversion |
| Custom cursor state | Generic CursorIterator[T] | Go 1.18 generics (2022) | Type-safe pagination abstraction |
| Manual scroll tracking | Structured pagination types | Current best practice (2026) | Cleaner code, reduced bugs |
| Client-side log filtering | API searchKey/searchCriteria | Zoho API design | Faster queries, reduced bandwidth |

**Deprecated/outdated:**
- **Manual millisecond conversion:** Use `time.UnixMilli()` and `t.UnixMilli()` (Go 1.17+) instead of `time.Unix(0, ms*1e6)`.
- **Unlimited log fetching:** Implement batch limits and streaming; don't fetch entire audit history in single call.
- **Hardcoded verification codes:** Display codes from API response; don't attempt to generate or validate DNS records client-side.

## Open Questions

1. **Are there API endpoints for security policy configuration (2FA, password policies)?**
   - What we know: Admin Console UI allows configuring 2FA enforcement, password policies, session timeout.
   - What's unclear: Whether these settings are exposed via read/write API endpoints or only via web UI.
   - Recommendation: Research API documentation further, test with curl during 03-02 implementation. If no API exists, document limitation and print redirect message to web console.

2. **Can we list active user sessions/devices via API?**
   - What we know: Admin Console shows "Session history" with active sessions per user.
   - What's unclear: No explicit API endpoint found in documentation for listing sessions.
   - Recommendation: Check Users API and Accounts API for session-related endpoints during curl testing. May be under user detail response or separate endpoint.

3. **What is the rate limit for audit log APIs?**
   - What we know: Standard Zoho Mail API has 30 req/min limit.
   - What's unclear: Whether audit/log APIs have separate limits or use the same global limit.
   - Recommendation: Assume same 30 req/min limit, rely on existing RateLimitTransport. Monitor 429 responses during testing.

4. **Do domain operations count against the same rate limit as admin ops?**
   - What we know: All operations use Zoho Mail API with same OAuth token.
   - What's unclear: Whether domains, audit logs, and user management share the same rate limit bucket.
   - Recommendation: Assume shared limit, implement conservative batch sizes. Document actual behavior in code comments after testing.

5. **What fields exist in domain detail response that aren't in list response?**
   - What we know: List endpoint documented, detail endpoint documented separately.
   - What's unclear: Complete field-by-field comparison of what's returned by each.
   - Recommendation: Test both endpoints with curl, create separate type definitions if field sets diverge significantly.

## Sources

### Primary (HIGH confidence)
- [GET - Fetch All Domains](https://www.zoho.com/mail/help/api/get-all-domains.html) - Domain list endpoint
- [POST - Add Domain to Organization](https://www.zoho.com/mail/help/api/post-add-domain-to-org.html) - Domain creation
- [PUT - Verify Domain](https://www.zoho.com/mail/help/api/put-verify-domain.html) - Domain verification methods
- [Domain API](https://www.zoho.com/mail/help/api/domain-api.html) - Complete domain API overview
- [Retrieve Activity Log](https://www.zoho.com/mail/help/api/get-activity-log.html) - Audit log endpoint
- [GET - Get login history](https://www.zoho.com/mail/help/api/get-login-history.html) - Login history endpoint
- [POST - Get SMTP logs](https://www.zoho.com/mail/help/api/get-smtp-logs.html) - SMTP log endpoint
- [Logs API Details](https://www.zoho.com/mail/help/api/logs-api.html) - Overview of log APIs
- [Zoho Mail API Index](https://www.zoho.com/mail/help/api/) - Complete API reference

### Secondary (MEDIUM confidence)
- [Domain Verification](https://www.zoho.com/mail/help/adminconsole/domain-verification.html) - Verification process overview
- [Audit logs in Admin Reports](https://www.zoho.com/mail/help/adminconsole/log-reports.html) - Audit log categories
- [TFA - Two Factor Authentication](https://www.zoho.com/mail/help/adminconsole/two-factor-authentication.html) - 2FA settings
- [Organization Security - Admin Console](https://www.zoho.com/mail/help/adminconsole/security.html) - Security policy overview
- [DKIM Configuration](https://www.zoho.com/mail/help/adminconsole/dkim-configuration.html) - DNS record setup
- [time package - time - Go Packages](https://pkg.go.dev/time) - Go time package documentation
- [How to Convert Milliseconds to time.Time in Golang](https://leapcell.io/blog/how-to-convert-milliseconds-to-time-in-golang) - Time conversion patterns

### Tertiary (LOW confidence - requires verification)
- WebSearch results on session management APIs - No explicit endpoint found, needs curl verification
- WebSearch results on security policy APIs - Web UI confirmed, API endpoints not documented
- Community discussions on log retention periods - 90-day limit mentioned but needs official confirmation

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already integrated and verified in Phases 1 and 2
- Domain APIs: HIGH - Verified via official Zoho Mail API documentation
- Audit/Log APIs: MEDIUM - Pagination patterns documented but cursor behavior needs testing
- Time conversion: HIGH - Go 1.17+ native millisecond support, standard library
- Security policy APIs: LOW - Web UI exists but API endpoints unclear, requires verification
- Pitfalls: MEDIUM - Based on API documentation analysis and Go best practices

**Research date:** 2026-02-14
**Valid until:** 2026-03-16 (30 days - Zoho Mail API is stable, infrequent breaking changes expected)
