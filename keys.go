package arouter

import (
	"context"
	"fmt"
	"net/http"
)

// --- Key Management API (aligned with OpenRouter) ---
//
// These methods require a Management Key (lr_mgmt_) for authentication.
// Regular API keys (lr_live_) are for LLM calls only.
//
// Endpoint: /api/v1/keys

// CreateKey creates a new regular API key.
//
//	resp, err := client.CreateKey(ctx, &arouter.CreateKeyRequest{
//	    Name: "my-service",
//	    Limit: float64Ptr(150),
//	    LimitReset: "monthly",
//	})
//	fmt.Println(resp.Key) // lr_live_xxx — use this to make LLM calls
func (c *Client) CreateKey(ctx context.Context, req *CreateKeyRequest) (*CreateKeyResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/api/v1/keys", req)
	if err != nil {
		return nil, err
	}

	var resp CreateKeyResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListKeys lists API keys managed by the current management key.
func (c *Client) ListKeys(ctx context.Context, opts *ListKeysOptions) (*ListKeysResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodGet, "/api/v1/keys", nil)
	if err != nil {
		return nil, err
	}

	if opts != nil {
		q := httpReq.URL.Query()
		if opts.PageSize > 0 {
			q.Set("page_size", fmt.Sprintf("%d", opts.PageSize))
		}
		if opts.PageToken != "" {
			q.Set("page_token", opts.PageToken)
		}
		if opts.Offset > 0 {
			q.Set("offset", fmt.Sprintf("%d", opts.Offset))
		}
		if opts.IncludeDisabled {
			q.Set("include_disabled", "true")
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	var resp ListKeysResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateKey updates an API key by its hash.
func (c *Client) UpdateKey(ctx context.Context, hash string, req *UpdateKeyRequest) (*UpdateKeyResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPatch, "/api/v1/keys/"+hash, req)
	if err != nil {
		return nil, err
	}

	var resp UpdateKeyResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteKey deletes an API key by its hash.
func (c *Client) DeleteKey(ctx context.Context, hash string) error {
	httpReq, err := c.newRequest(ctx, http.MethodDelete, "/api/v1/keys/"+hash, nil)
	if err != nil {
		return err
	}
	return c.do(httpReq, nil)
}
