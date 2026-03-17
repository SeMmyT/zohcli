package zoho

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// antispamCheckInfo holds API mapping for an antispam check
type antispamCheckInfo struct {
	GetMode     string // query param for GET
	PutMode     string // "mode" field for PUT
	ResponseKey string // key in response data
	BodyKey     string // "option" or "dmarcOption" for PUT body
}

var antispamChecks = map[string]antispamCheckInfo{
	"spf": {
		GetMode:     "getSPFFailOption",
		PutMode:     "updateSPFFailOption",
		ResponseKey: "spfFail",
		BodyKey:     "option",
	},
	"dkim": {
		GetMode:     "getDKIMFailOption",
		PutMode:     "updateDKIMFailOption",
		ResponseKey: "dkimFail",
		BodyKey:     "option",
	},
	"dmarc": {
		GetMode:     "getDMARCFailOption",
		PutMode:     "updateDMARCFailOption",
		ResponseKey: "dmarcFail",
		BodyKey:     "dmarcOption",
	},
	"cousin-domain": {
		GetMode:     "getCousinDomainOption",
		PutMode:     "updateCousinDomainOption",
		ResponseKey: "cdFail",
		BodyKey:     "dmarcOption",
	},
}

// GetAntispamOption fetches the current antispam action for the given check.
func (ac *AdminClient) GetAntispamOption(ctx context.Context, check string) (string, error) {
	info, ok := antispamChecks[check]
	if !ok {
		return "", fmt.Errorf("unknown antispam check: %s", check)
	}

	path := fmt.Sprintf("/api/organization/%d/antispam/options?mode=%s", ac.zoid, info.GetMode)
	resp, err := ac.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ac.parseErrorResponse(resp)
	}

	// Parse as generic map since each check has a different response key
	var result struct {
		Status struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
		} `json:"status"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if result.Status.Code != 200 {
		return "", fmt.Errorf("API error: %s (code %d)", result.Status.Description, result.Status.Code)
	}

	val, ok := result.Data[info.ResponseKey]
	if !ok {
		return "", fmt.Errorf("response missing key %q", info.ResponseKey)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("unexpected type for key %q: %T", info.ResponseKey, val)
	}

	return str, nil
}

// SetAntispamOption updates the antispam action for the given check.
func (ac *AdminClient) SetAntispamOption(ctx context.Context, check, option string) error {
	info, ok := antispamChecks[check]
	if !ok {
		return fmt.Errorf("unknown antispam check: %s", check)
	}

	path := fmt.Sprintf("/api/organization/%d/antispam/options", ac.zoid)

	reqBody := map[string]string{
		"mode":      info.PutMode,
		info.BodyKey: option,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := ac.client.DoMail(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ac.parseErrorResponse(resp)
	}

	return nil
}
