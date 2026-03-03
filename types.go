package arouter

import "encoding/json"

// ==================== LLM Types ====================

// Message represents a chat message in the OpenAI-compatible format.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents a function call requested by the model.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction contains the name and arguments of a tool call.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool describes a function the model may call.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction is the function definition within a Tool.
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ResponseFormat constrains the output format of the model.
type ResponseFormat struct {
	Type string `json:"type"`
}

// ChatCompletionRequest is the request payload for chat completions.
type ChatCompletionRequest struct {
	Model            string          `json:"model"`
	Messages         []Message       `json:"messages"`
	Temperature      *float64        `json:"temperature,omitempty"`
	TopP             *float64        `json:"top_p,omitempty"`
	N                *int            `json:"n,omitempty"`
	Stream           bool            `json:"stream,omitempty"`
	Stop             []string        `json:"stop,omitempty"`
	MaxTokens        *int            `json:"max_tokens,omitempty"`
	PresencePenalty  *float64        `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64        `json:"frequency_penalty,omitempty"`
	Tools            []Tool          `json:"tools,omitempty"`
	ToolChoice       any             `json:"tool_choice,omitempty"`
	ResponseFormat   *ResponseFormat `json:"response_format,omitempty"`
	User             string          `json:"user,omitempty"`
	Extra            map[string]any  `json:"-"`
}

// ChatCompletionResponse is the response from a non-streaming chat completion.
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice represents a single completion choice.
type Choice struct {
	Index        int      `json:"index"`
	Message      *Message `json:"message,omitempty"`
	Delta        *Message `json:"delta,omitempty"`
	FinishReason *string  `json:"finish_reason,omitempty"`
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionChunk is a single chunk from a streaming chat completion.
type ChatCompletionChunk struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// ==================== Key Management Types (aligned with OpenRouter) ====================

// CreateKeyRequest is the payload for creating an API key via the management API.
type CreateKeyRequest struct {
	Name             string   `json:"name"`
	Limit            *float64 `json:"limit,omitempty"`
	LimitReset       string   `json:"limit_reset,omitempty"`
	ExpiresAt        *string  `json:"expires_at,omitempty"`
	AllowedProviders []string `json:"allowed_providers,omitempty"`
	AllowedModels    []string `json:"allowed_models,omitempty"`
}

// CreateKeyResponse is returned when an API key is created.
type CreateKeyResponse struct {
	Data   KeyObject `json:"data"`
	Key    string    `json:"key"`
}

// KeyObject represents a key in API responses (aligned with OpenRouter).
type KeyObject struct {
	Hash             string          `json:"hash"`
	Name             string          `json:"name"`
	Label            string          `json:"label,omitempty"`
	KeyType          string          `json:"key_type"`
	Disabled         bool            `json:"disabled"`
	Limit            *float64        `json:"limit"`
	LimitRemaining   *float64        `json:"limit_remaining"`
	LimitReset       string          `json:"limit_reset,omitempty"`
	AllowedProviders []string        `json:"allowed_providers,omitempty"`
	AllowedModels    []string        `json:"allowed_models,omitempty"`
	Usage            float64         `json:"usage"`
	UsageDaily       float64         `json:"usage_daily"`
	UsageWeekly      float64         `json:"usage_weekly"`
	UsageMonthly     float64         `json:"usage_monthly"`
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        *string         `json:"updated_at"`
	ExpiresAt        *string         `json:"expires_at,omitempty"`
}

// UpdateKeyRequest is the payload for updating an API key.
type UpdateKeyRequest struct {
	Name             *string  `json:"name,omitempty"`
	Disabled         *bool    `json:"disabled,omitempty"`
	Limit            *float64 `json:"limit,omitempty"`
	LimitReset       *string  `json:"limit_reset,omitempty"`
	AllowedProviders []string `json:"allowed_providers,omitempty"`
	AllowedModels    []string `json:"allowed_models,omitempty"`
}

// UpdateKeyResponse is returned when an API key is updated.
type UpdateKeyResponse struct {
	Data KeyObject `json:"data"`
}

// ListKeysOptions contains query parameters for listing keys.
type ListKeysOptions struct {
	PageSize        int    `json:"page_size,omitempty"`
	PageToken       string `json:"page_token,omitempty"`
	Offset          int    `json:"offset,omitempty"`
	IncludeDisabled bool   `json:"include_disabled,omitempty"`
}

// ListKeysResponse is the paginated response for key listing.
type ListKeysResponse struct {
	Data []KeyObject `json:"data"`
}

// DeleteKeyResponse is returned when a key is deleted.
type DeleteKeyResponse struct {
	Data struct {
		Deleted bool `json:"deleted"`
	} `json:"data"`
}

// RateLimitConfig configures per-key rate limits.
type RateLimitConfig struct {
	RequestsPerMinute int32 `json:"requests_per_minute,omitempty"`
	RequestsPerDay    int32 `json:"requests_per_day,omitempty"`
	TokensPerMinute   int32 `json:"tokens_per_minute,omitempty"`
}

// ==================== Embedding Types ====================

// EmbeddingRequest is the request payload for creating embeddings.
type EmbeddingRequest struct {
	Model          string `json:"model"`
	Input          any    `json:"input"`
	EncodingFormat string `json:"encoding_format,omitempty"`
}

// EmbeddingResponse is the response from the embeddings endpoint.
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  EmbeddingUsage  `json:"usage"`
}

// EmbeddingData is a single embedding vector.
type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

// EmbeddingUsage tracks token consumption for embeddings.
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ==================== Model Types ====================

// ModelListResponse is the response from the list models endpoint.
type ModelListResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// ModelInfo describes a single model.
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ==================== Usage & Analytics Types ====================

// UsageQuery contains query parameters for usage API calls.
type UsageQuery struct {
	StartTime   string
	EndTime     string
	ProviderID  string
	Model       string
	KeyID       string
	Granularity string
}

// UsageSummary is the response from the usage summary endpoint.
type UsageSummary struct {
	TotalRequests     int64           `json:"total_requests"`
	TotalInputTokens  int64           `json:"total_input_tokens"`
	TotalOutputTokens int64           `json:"total_output_tokens"`
	TotalTokens       int64           `json:"total_tokens"`
	EstimatedCostUSD  float64         `json:"estimated_cost_usd"`
	ByProvider        []ProviderUsage `json:"by_provider,omitempty"`
	ByModel           []ModelUsage    `json:"by_model,omitempty"`
}

// ProviderUsage is per-provider usage breakdown.
type ProviderUsage struct {
	ProviderID       string  `json:"provider_id"`
	Requests         int64   `json:"requests"`
	InputTokens      int64   `json:"input_tokens"`
	OutputTokens     int64   `json:"output_tokens"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
}

// ModelUsage is per-model usage breakdown.
type ModelUsage struct {
	ProviderID       string  `json:"provider_id"`
	Model            string  `json:"model"`
	Requests         int64   `json:"requests"`
	InputTokens      int64   `json:"input_tokens"`
	OutputTokens     int64   `json:"output_tokens"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
}

// UsageTimeSeries is the response from the usage timeseries endpoint.
type UsageTimeSeries struct {
	DataPoints []UsageDataPoint `json:"data_points"`
}

// UsageDataPoint is a single data point in the usage time series.
type UsageDataPoint struct {
	Timestamp        string  `json:"timestamp"`
	Requests         int64   `json:"requests"`
	InputTokens      int64   `json:"input_tokens"`
	OutputTokens     int64   `json:"output_tokens"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
}

// Legacy types kept for backward compat in internal usage
type APIKeyInfo struct {
	ID               string          `json:"id"`
	Prefix           string          `json:"prefix,omitempty"`
	Name             string          `json:"name"`
	KeyType          string          `json:"key_type,omitempty"`
	Disabled         bool            `json:"disabled,omitempty"`
	AllowedProviders []string        `json:"allowed_providers,omitempty"`
	AllowedModels    []string        `json:"allowed_models,omitempty"`
	RateLimit        *RateLimitConfig `json:"rate_limit,omitempty"`
	ExpiresAt        *string         `json:"expires_at,omitempty"`
	CreatedAt        json.RawMessage `json:"created_at,omitempty"`
}
