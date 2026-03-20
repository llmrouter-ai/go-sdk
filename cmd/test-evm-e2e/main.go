package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"strings"
	"time"

	arouter "github.com/arouter-ai/arouter-go"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	baseURL := envOr("AROUTER_BASE_URL", "http://localhost:19080")
	hexKey := os.Getenv("EVM_PRIVATE_KEY")
	if hexKey == "" {
		fmt.Fprintln(os.Stderr, "EVM_PRIVATE_KEY environment variable is required")
		os.Exit(1)
	}

	key, err := parseHexKey(hexKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid EVM private key: %v\n", err)
		os.Exit(1)
	}

	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
	fmt.Printf("Testing against: %s\n", baseURL)
	fmt.Printf("Wallet: %s\n", addr)

	client := arouter.NewClient(
		baseURL,
		"",
		arouter.WithX402CoinbasePayment(key),
		arouter.WithTimeout(90*time.Second),
	)

	resp, err := client.ChatCompletion(context.Background(), &arouter.ChatCompletionRequest{
		Model:    "openrouter/auto",
		Messages: []arouter.Message{{Role: "user", Content: "Say hi in exactly 3 words"}},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Go EVM x402 FAIL: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Go EVM x402 PASS")
	fmt.Printf("Response: %s\n", resp.Choices[0].Message.Content)
}

func parseHexKey(v string) (*ecdsa.PrivateKey, error) {
	v = strings.TrimPrefix(v, "0x")
	return crypto.HexToECDSA(v)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
