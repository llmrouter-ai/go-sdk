# ARouter Go SDK

Official Go client for the [ARouter](https://github.com/arouter-ai) API gateway — one API key, every LLM provider.

## Installation

```bash
go get github.com/arouter-ai/arouter-go
```

> Requires Go 1.21+. Zero external dependencies.

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	arouter "github.com/arouter-ai/arouter-go"
)

func main() {
	client := arouter.NewClient("https://api.arouter.io", "lr_live_xxx")

	resp, err := client.ChatCompletion(context.Background(), &arouter.ChatCompletionRequest{
		Model: "openrouter/anthropic/claude-sonnet-4",
		Messages: []arouter.Message{
			{Role: "user", Content: "Hello!"},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Choices[0].Message.Content)
}
```

## API Overview

| Category | Methods |
|----------|---------|
| **LLM** | `ChatCompletion`, `ChatCompletionStream`, `CreateEmbedding`, `ListModels`, `ProxyRequest` |
| **Keys** | `CreateKey`, `ListKeys`, `UpdateKey`, `DeleteKey` |
| **Usage** | `GetUsageSummary`, `GetUsageTimeSeries` |

## Streaming

```go
stream, err := client.ChatCompletionStream(ctx, &arouter.ChatCompletionRequest{
	Model:    "openrouter/anthropic/claude-sonnet-4",
	Messages: []arouter.Message{{Role: "user", Content: "Tell me a story"}},
})
if err != nil {
	log.Fatal(err)
}
defer stream.Close()

for {
	chunk, err := stream.Recv()
	if err == arouter.ErrStreamDone {
		break
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(chunk.Choices[0].Delta.Content)
}
```

## Key Management

Management keys (`lr_mgmt_`) can create and manage API keys with provider/model restrictions, rate limits, and quotas — no dashboard login required.

```go
mgmtClient := arouter.NewClient("https://api.arouter.io", "lr_mgmt_xxx")

key, err := mgmtClient.CreateKey(ctx, &arouter.CreateKeyRequest{
	Name:             "worker-1",
	AllowedProviders: []string{"openai", "anthropic"},
	AllowedModels:    []string{"gpt-4o", "claude-sonnet-4-20250514"},
	Limit:            float64Ptr(150),
	LimitReset:       "monthly",
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("API Key:", key.Key) // lr_live_xxx

// List all keys
keys, _ := mgmtClient.ListKeys(ctx, nil)
for _, k := range keys.Data {
	fmt.Println(k.Hash, k.Name)
}

// Update a key
mgmtClient.UpdateKey(ctx, key.Data.Hash, &arouter.UpdateKeyRequest{
	Disabled: boolPtr(true),
})

// Delete a key
mgmtClient.DeleteKey(ctx, key.Data.Hash)
```

## Embeddings

```go
resp, err := client.CreateEmbedding(ctx, &arouter.EmbeddingRequest{
	Model: "openai/text-embedding-3-small",
	Input: "Hello, world",
})
```

## List Models

```go
models, err := client.ListModels(ctx)
for _, m := range models.Data {
	fmt.Printf("%s (by %s)\n", m.ID, m.OwnedBy)
}
```

## Usage Analytics

```go
summary, err := client.GetUsageSummary(ctx, &arouter.UsageQuery{
	StartTime: "2025-01-01T00:00:00Z",
	EndTime:   "2025-01-31T23:59:59Z",
})
fmt.Println(summary) // Requests: 1234 | Tokens: 56789 | Cost: $1.23
```

## Provider Proxy

Forward raw requests to any provider endpoint:

```go
body := strings.NewReader(`{"input": "hello", "model": "text-embedding-3-small"}`)
resp, err := client.ProxyRequest(ctx, "openai", "v1/embeddings", body)
if err != nil {
	log.Fatal(err)
}
defer resp.Body.Close()
```

## Client Options

```go
client := arouter.NewClient(baseURL, apiKey,
	arouter.WithTimeout(60 * time.Second),
	arouter.WithHTTPClient(customHTTPClient),
)
```

## Error Handling

All API errors are returned as `*arouter.APIError` and support `errors.Is` matching:

```go
_, err := client.ChatCompletion(ctx, req)
if errors.Is(err, arouter.ErrRateLimited) {
	// back off and retry
}
if errors.Is(err, arouter.ErrQuotaExceeded) {
	// quota exhausted
}

var apiErr *arouter.APIError
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.StatusCode, apiErr.Code, apiErr.Message)
}
```

Sentinel errors: `ErrUnauthorized` · `ErrForbidden` · `ErrNotFound` · `ErrRateLimited` · `ErrQuotaExceeded` · `ErrBadRequest` · `ErrServerError`

## License

MIT
