// End-to-end test: Solana wallet auth + x402 payment on Solana Devnet.
// Requires: services running with X402_ENABLED=true X402_NETWORKS containing solana:EtWTRABZaYq6iMfeYKouRu166VU2xqa1
//
// Run:   SOL_PRIVATE_KEY=<base58_64byte_keypair> go run ./cmd/test-solana-e2e
package main

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"os"
	"time"

	arouter "github.com/arouter-ai/arouter-go"
	"github.com/mr-tron/base58"
)

func main() {
	baseURL := envOr("AROUTER_BASE_URL", "http://localhost:18080")
	keyB58 := os.Getenv("SOL_PRIVATE_KEY")
	if keyB58 == "" {
		fmt.Fprintln(os.Stderr, "SOL_PRIVATE_KEY environment variable is required (base58 encoded 64-byte keypair)")
		os.Exit(1)
	}

	keyBytes, err := base58.Decode(keyB58)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid base58 key: %v\n", err)
		os.Exit(1)
	}
	if len(keyBytes) != ed25519.PrivateKeySize {
		fmt.Fprintf(os.Stderr, "invalid key length: %d, expected %d\n", len(keyBytes), ed25519.PrivateKeySize)
		os.Exit(1)
	}

	privKey := ed25519.PrivateKey(keyBytes)
	pubKey := privKey.Public().(ed25519.PublicKey)
	addr := base58.Encode(pubKey)

	fmt.Printf("Solana wallet: %s\n", addr)
	fmt.Printf("Base URL:      %s\n\n", baseURL)

	// Solana wallet auth + x402 auto-payment
	client := arouter.NewClient(baseURL, "",
		arouter.WithX402SolanaPayment(privKey),
		arouter.WithTimeout(90*time.Second),
	)

	ctx := context.Background()

	fmt.Println("=== Chat Completion (Solana wallet auth + x402 auto-payment) ===")
	fmt.Println("Flow: wallet auth → 402 → x402 SDK signs USDC payment → settle → 200")
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
