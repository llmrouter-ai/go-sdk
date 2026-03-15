// End-to-end test for x402 wallet auth + payment flow.
// Run with: go run ./cmd/test-e2e
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	arouter "github.com/arouter-ai/arouter-go"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	baseURL := envOr("AROUTER_BASE_URL", "http://localhost:18080")
	keyHex := envOr("AGENT_PRIVATE_KEY", "3b48d05d86c9b7f044d5230eeb9397e0a90ccff8e02ae9f93e0a988dea5e9d8b")

	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid private key: %v\n", err)
		os.Exit(1)
	}
	privateKey, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid ECDSA key: %v\n", err)
		os.Exit(1)
	}

	signer := arouter.NewEvmWalletSigner(privateKey)
	fmt.Printf("Agent wallet: %s\n", signer.Address())
	fmt.Printf("Base URL: %s\n\n", baseURL)

	client := arouter.NewClient(baseURL, "",
		arouter.WithWalletAuth(signer),
		arouter.WithTimeout(30*time.Second),
	)

	ctx := context.Background()

	fmt.Println("=== Chat Completion (wallet auth, no API key) ===")
	resp, err := client.ChatCompletion(ctx, &arouter.ChatCompletionRequest{
		Model:    "openrouter/auto",
		Messages: []arouter.Message{{Role: "user", Content: "Say hi in exactly 3 words"}},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\n(Expected: 402 if no credits, or success if credits available)")
	} else {
		fmt.Printf("Response: %s\n", resp.Choices[0].Message.Content)
		fmt.Printf("Model: %s\n", resp.Model)
		fmt.Printf("Usage: %d input, %d output tokens\n", resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
