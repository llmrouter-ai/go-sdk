package arouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

// Client is the ARouter SDK client.
//
// Initialize with just a base URL and API key — no tenant ID, JWT, or
// login credentials needed. The server identifies your tenant from the key.
//
//	client := arouter.NewClient("https://api.arouter.io", "lr_live_xxx")
//
// The client provides three groups of methods:
//
//	LLM:    ChatCompletion, ChatCompletionStream, CreateEmbedding, ListModels, ProxyRequest
//	Keys:   CreateKey, ListKeys, UpdateKey, DeleteKey
//	Usage:  GetUsageSummary, GetUsageTimeSeries
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new ARouter client.
//
//	baseURL is the root URL of the ARouter gateway (e.g. "https://api.arouter.io").
//	apiKey  is your API key (lr_live_xxx) or management key (lr_mgmt_xxx).
func NewClient(baseURL, apiKey string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// newRequest builds an authenticated *http.Request.
func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	url := c.baseURL + path

	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("arouter: marshal request: %w", err)
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// do executes a request and decodes the JSON response into dst.
func (c *Client) do(req *http.Request, dst any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("arouter: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseErrorResponse(resp)
	}

	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return fmt.Errorf("arouter: decode response: %w", err)
		}
	}
	return nil
}

// parseErrorResponse reads an error response body and returns an *APIError.
func parseErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	apiErr := &APIError{StatusCode: resp.StatusCode}

	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) == nil && envelope.Error.Message != "" {
		apiErr.Code = envelope.Error.Code
		apiErr.Message = envelope.Error.Message
		return apiErr
	}

	// Fallback: try flat structure.
	_ = json.Unmarshal(body, apiErr)
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}
	return apiErr
}
