package zoho

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"

	"github.com/semmy-space/zoh/internal/config"
)

// AdminClient wraps the Zoho Client with admin-specific functionality
type AdminClient struct {
	client *Client
	zoid   int64 // Cached organization ID
}

// NewAdminClient creates a new AdminClient with the given config and token source
// It automatically resolves and caches the organization ID
func NewAdminClient(cfg *config.Config, tokenSource oauth2.TokenSource) (*AdminClient, error) {
	client, err := NewClient(cfg, tokenSource)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	ac := &AdminClient{
		client: client,
	}

	// Resolve organization ID
	ctx := context.Background()
	zoid, err := ac.getOrganizationID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get organization ID: %w", err)
	}
	ac.zoid = zoid

	return ac, nil
}

// getOrganizationID fetches the organization ID from the Zoho API
func (ac *AdminClient) getOrganizationID(ctx context.Context) (int64, error) {
	resp, err := ac.client.Do(ctx, http.MethodGet, "/api/organization/", nil)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, ac.parseErrorResponse(resp)
	}

	var orgResp OrgResponse
	if err := json.NewDecoder(resp.Body).Decode(&orgResp); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	if orgResp.Status.Code != 200 {
		return 0, fmt.Errorf("API error: %s (code %d)", orgResp.Status.Description, orgResp.Status.Code)
	}

	return orgResp.Data.OrganizationID, nil
}

// ListUsers fetches a list of users with pagination
func (ac *AdminClient) ListUsers(ctx context.Context, start, limit int) ([]User, error) {
	path := fmt.Sprintf("/api/organization/%d/accounts?start=%d&limit=%d", ac.zoid, start, limit)
	resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ac.parseErrorResponse(resp)
	}

	var userResp UserListResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if userResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", userResp.Status.Description, userResp.Status.Code)
	}

	return userResp.Data, nil
}

// GetUser fetches a single user by account ID
func (ac *AdminClient) GetUser(ctx context.Context, accountID int64) (*User, error) {
	path := fmt.Sprintf("/api/organization/%d/accounts/%d", ac.zoid, accountID)
	resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ac.parseErrorResponse(resp)
	}

	var userResp UserDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if userResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", userResp.Status.Description, userResp.Status.Code)
	}

	return &userResp.Data, nil
}

// GetUserByEmail fetches a user by email address
// This iterates through all users until a match is found
func (ac *AdminClient) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	// Create a page iterator to search through all users
	iterator := NewPageIterator(func(start, limit int) ([]User, error) {
		return ac.ListUsers(ctx, start, limit)
	}, 50)

	// Fetch all users (this will paginate automatically)
	users, err := iterator.FetchAll()
	if err != nil {
		return nil, fmt.Errorf("fetch users: %w", err)
	}

	// Search for matching email
	for _, user := range users {
		if user.EmailAddress == email {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("user not found: %s", email)
}

// parseErrorResponse attempts to parse an error response from the Zoho API
func (ac *AdminClient) parseErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error response", resp.StatusCode)
	}

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		// If we can't parse the error, return the raw body
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// If we successfully parsed an APIError, use its Error() method
	return fmt.Errorf("HTTP %d: %w", resp.StatusCode, &apiErr)
}

// GetUserByIdentifier is a helper that accepts either a ZUID or email address
func (ac *AdminClient) GetUserByIdentifier(ctx context.Context, identifier string) (*User, error) {
	// Try to parse as int64 (ZUID)
	if zuid, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		return ac.GetUser(ctx, zuid)
	}

	// Otherwise, treat as email
	return ac.GetUserByEmail(ctx, identifier)
}

// CreateUser creates a new user in the organization
func (ac *AdminClient) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	path := fmt.Sprintf("/api/organization/%d/accounts", ac.zoid)

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

	var userResp UserDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if userResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", userResp.Status.Description, userResp.Status.Code)
	}

	return &userResp.Data, nil
}

// UpdateUserRole changes a user's role in the organization
func (ac *AdminClient) UpdateUserRole(ctx context.Context, zuid int64, newRole string) error {
	path := fmt.Sprintf("/api/organization/%d/accounts", ac.zoid)

	req := UpdateUserRequest{
		Mode:    "changeRole",
		ZUID:    zuid,
		NewRole: newRole,
	}

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

// EnableUser activates a user account
func (ac *AdminClient) EnableUser(ctx context.Context, zuid int64) error {
	path := fmt.Sprintf("/api/organization/%d/accounts", ac.zoid)

	req := UpdateUserRequest{
		Mode: "enableUser",
		ZUID: zuid,
	}

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

// DisableUser deactivates a user account with specified options
func (ac *AdminClient) DisableUser(ctx context.Context, zuid int64, opts DisableUserOpts) error {
	path := fmt.Sprintf("/api/organization/%d/accounts", ac.zoid)

	// Build request body with mode, zuid, and option fields
	reqBody := map[string]interface{}{
		"mode": "disableUser",
		"zuid": zuid,
	}

	if opts.BlockIncoming {
		reqBody["blockIncoming"] = true
	}
	if opts.RemoveMailForward {
		reqBody["removeMailForward"] = true
	}
	if opts.RemoveGroupMembership {
		reqBody["removeGroupMembership"] = true
	}
	if opts.RemoveAlias {
		reqBody["removeAlias"] = true
	}

	body, err := json.Marshal(reqBody)
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

// DeleteUser permanently deletes a user from the organization
func (ac *AdminClient) DeleteUser(ctx context.Context, zuid int64) error {
	path := fmt.Sprintf("/api/organization/%d/accounts/%d", ac.zoid, zuid)

	resp, err := ac.client.Do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ac.parseErrorResponse(resp)
	}

	return nil
}

// ListGroups fetches a list of groups with pagination
func (ac *AdminClient) ListGroups(ctx context.Context, start, limit int) ([]Group, error) {
	path := fmt.Sprintf("/api/organization/%d/groups?start=%d&limit=%d", ac.zoid, start, limit)
	resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ac.parseErrorResponse(resp)
	}

	var groupResp GroupListResponse
	if err := json.NewDecoder(resp.Body).Decode(&groupResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if groupResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", groupResp.Status.Description, groupResp.Status.Code)
	}

	return groupResp.Data, nil
}

// GetGroup fetches a single group by ZGID
func (ac *AdminClient) GetGroup(ctx context.Context, zgid int64) (*Group, error) {
	path := fmt.Sprintf("/api/organization/%d/groups/%d", ac.zoid, zgid)
	resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ac.parseErrorResponse(resp)
	}

	var groupResp GroupDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&groupResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if groupResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", groupResp.Status.Description, groupResp.Status.Code)
	}

	return &groupResp.Data, nil
}

// GetGroupMembers fetches the member list for a group
func (ac *AdminClient) GetGroupMembers(ctx context.Context, zgid int64) ([]GroupMember, error) {
	path := fmt.Sprintf("/api/organization/%d/groups/%d/members", ac.zoid, zgid)
	resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ac.parseErrorResponse(resp)
	}

	var membersResp GroupMembersResponse
	if err := json.NewDecoder(resp.Body).Decode(&membersResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if membersResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", membersResp.Status.Description, membersResp.Status.Code)
	}

	return membersResp.Data, nil
}

// GetGroupByEmail fetches a group by email address
// This iterates through all groups until a match is found
func (ac *AdminClient) GetGroupByEmail(ctx context.Context, email string) (*Group, error) {
	// Create a page iterator to search through all groups
	iterator := NewPageIterator(func(start, limit int) ([]Group, error) {
		return ac.ListGroups(ctx, start, limit)
	}, 50)

	// Fetch all groups (this will paginate automatically)
	groups, err := iterator.FetchAll()
	if err != nil {
		return nil, fmt.Errorf("fetch groups: %w", err)
	}

	// Search for matching email
	for _, group := range groups {
		if group.GroupEmailAddress == email {
			return &group, nil
		}
	}

	return nil, fmt.Errorf("group not found: %s", email)
}

// CreateGroup creates a new group
func (ac *AdminClient) CreateGroup(ctx context.Context, req CreateGroupRequest) (*Group, error) {
	path := fmt.Sprintf("/api/organization/%d/groups", ac.zoid)

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

	var groupResp GroupDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&groupResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if groupResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", groupResp.Status.Description, groupResp.Status.Code)
	}

	return &groupResp.Data, nil
}

// UpdateGroup updates a group's name and/or description
func (ac *AdminClient) UpdateGroup(ctx context.Context, zgid int64, name, description string) error {
	path := fmt.Sprintf("/api/organization/%d/groups/%d", ac.zoid, zgid)

	// Build request body with only non-empty fields
	reqBody := map[string]interface{}{
		"mode": "updateMailGroup",
	}
	if name != "" {
		reqBody["groupName"] = name
	}
	if description != "" {
		reqBody["description"] = description
	}

	body, err := json.Marshal(reqBody)
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

// DeleteGroup permanently deletes a group
func (ac *AdminClient) DeleteGroup(ctx context.Context, zgid int64) error {
	path := fmt.Sprintf("/api/organization/%d/groups/%d", ac.zoid, zgid)
	resp, err := ac.client.Do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ac.parseErrorResponse(resp)
	}

	return nil
}

// AddGroupMembers adds members to a group with batching for large member lists
func (ac *AdminClient) AddGroupMembers(ctx context.Context, zgid int64, members []GroupMemberToAdd) error {
	path := fmt.Sprintf("/api/organization/%d/groups/%d", ac.zoid, zgid)

	// Batch members in groups of 50 to stay within API limits
	batchSize := 50
	for i := 0; i < len(members); i += batchSize {
		end := i + batchSize
		if end > len(members) {
			end = len(members)
		}

		batch := members[i:end]
		req := AddGroupMembersRequest{
			Mode:                "addMailGroupMember",
			MailGroupMemberList: batch,
		}

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
	}

	return nil
}

// RemoveGroupMembers removes members from a group
func (ac *AdminClient) RemoveGroupMembers(ctx context.Context, zgid int64, members []GroupMemberToRemove) error {
	path := fmt.Sprintf("/api/organization/%d/groups/%d", ac.zoid, zgid)

	req := RemoveGroupMembersRequest{
		Mode:                "removeMailGroupMember",
		MailGroupMemberList: members,
	}

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
