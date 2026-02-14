package zoho

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListSignatures fetches all email signatures for the primary account
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

	var sigResp SignatureListResponse
	if err := json.NewDecoder(resp.Body).Decode(&sigResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if sigResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", sigResp.Status.Description, sigResp.Status.Code)
	}

	return sigResp.Data, nil
}

// AddSignature creates a new email signature and returns the signature ID
func (mc *MailClient) AddSignature(ctx context.Context, sig *Signature) (string, error) {
	// Build request body
	reqBody := map[string]interface{}{
		"name":     sig.Name,
		"content":  sig.Content,
		"position": sig.Position,
	}

	// Add optional assignUsers field if specified
	if sig.AssignUsers != "" {
		reqBody["assignUsers"] = sig.AssignUsers
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	path := "/api/accounts/signature"
	resp, err := mc.client.DoMail(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", mc.parseErrorResponse(resp)
	}

	var sigCreateResp SignatureCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&sigCreateResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if sigCreateResp.Status.Code != 200 {
		return "", fmt.Errorf("API error: %s (code %d)", sigCreateResp.Status.Description, sigCreateResp.Status.Code)
	}

	return sigCreateResp.Data.ID, nil
}

// GetAccountDetails fetches account details including vacation reply and forwarding settings
func (mc *MailClient) GetAccountDetails(ctx context.Context) (*AccountDetails, error) {
	path := fmt.Sprintf("/api/accounts/%s", mc.accountID)
	resp, err := mc.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mc.parseErrorResponse(resp)
	}

	var accountResp AccountDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&accountResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if accountResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", accountResp.Status.Description, accountResp.Status.Code)
	}

	return &accountResp.Data, nil
}

// AddVacationReply enables vacation auto-reply with specified settings
func (mc *MailClient) AddVacationReply(ctx context.Context, vacation *VacationReply) error {
	reqBody := map[string]interface{}{
		"mode": "addVacationReply",
		"vacationResponse": map[string]interface{}{
			"fromDate":   vacation.FromDate,
			"toDate":     vacation.ToDate,
			"sendingInt": vacation.SendingInt,
			"subject":    vacation.Subject,
			"content":    vacation.Content,
			"sendTo":     vacation.SendTo,
		},
	}

	return mc.updateAccountSettings(ctx, reqBody)
}

// DisableVacationReply disables vacation auto-reply
func (mc *MailClient) DisableVacationReply(ctx context.Context) error {
	reqBody := map[string]interface{}{
		"mode": "disableVacationReply",
	}

	return mc.updateAccountSettings(ctx, reqBody)
}

// UpdateDisplayName updates the account display name
func (mc *MailClient) UpdateDisplayName(ctx context.Context, displayName string) error {
	reqBody := map[string]interface{}{
		"mode":        "updateDisplayName",
		"displayName": displayName,
	}

	return mc.updateAccountSettings(ctx, reqBody)
}

// updateAccountSettings is a private helper for mode-based PUT operations
func (mc *MailClient) updateAccountSettings(ctx context.Context, reqBody map[string]interface{}) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	path := fmt.Sprintf("/api/accounts/%s", mc.accountID)
	resp, err := mc.client.DoMail(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mc.parseErrorResponse(resp)
	}

	// Parse standard status response
	var statusResp struct {
		Status struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
		} `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if statusResp.Status.Code != 200 {
		return fmt.Errorf("API error: %s (code %d)", statusResp.Status.Description, statusResp.Status.Code)
	}

	return nil
}
