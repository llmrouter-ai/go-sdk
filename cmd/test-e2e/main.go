// End-to-end test: wallet auth + x402 payment on Base Sepolia.
// Requires: services running from worktree with X402_ENABLED=true X402_NETWORKS=eip155:84532
//
// Run:   AGENT_PRIVATE_KEY=xxx go run ./cmd/test-e2e
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	arouter "github.com/arouter-ai/arouter-go"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	baseURL := envOr("AROUTER_BASE_URL", "http://localhost:18080")
	keyHex := os.Getenv("AGENT_PRIVATE_KEY")
	if keyHex == "" {
		fmt.Fprintln(os.Stderr, "AGENT_PRIVATE_KEY environment variable is required")
		os.Exit(1)
	}

	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid key: %v\n", err)
		os.Exit(1)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	fmt.Printf("Agent wallet: %s\n", addr.Hex())
	fmt.Printf("Base URL:     %s\n\n", baseURL)

	// One-liner: wallet auth + automatic x402 payment
	client := arouter.NewClient(baseURL, "",
		arouter.WithX402CoinbasePayment(key),
		arouter.WithTimeout(60*time.Second),
	)

	ctx := context.Background()

	fmt.Println("=== Chat Completion (wallet auth + x402 auto-payment) ===")
	fmt.Println("If no credits: 402 → x402 SDK signs USDC payment → retries → 200")
	fmt.Println()

	resp, err := client.ChatCompletion(ctx, &arouter.ChatCompletionRequest{
		Model:    "openrouter/auto",
		Messages: []arouter.Message{{Role: "user", Content: "Say hi in exactly 3 words"}},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Response: %s\n", resp.Choices[0].Message.Content)
	fmt.Printf("Model:    %s\n", resp.Model)
	fmt.Printf("Tokens:   %d input, %d output\n", resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
