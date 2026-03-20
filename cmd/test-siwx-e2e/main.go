package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"time"

	arouter "github.com/arouter-ai/arouter-go"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	baseURL := envOr("AROUTER_BASE_URL", "http://localhost:19080")

	var key *ecdsa.PrivateKey
	var err error
	if hexKey := os.Getenv("EVM_PRIVATE_KEY"); hexKey != "" {
		key, err = crypto.HexToECDSA(trim0x(hexKey))
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid EVM_PRIVATE_KEY: %v\n", err)
			os.Exit(1)
		}
	} else {
		key, _ = crypto.GenerateKey()
	}

	signer := arouter.NewEvmWalletSigner(key)
	fmt.Printf("Testing against: %s\n", baseURL)
	fmt.Printf("Wallet: %s\n", signer.Address())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	auth, err := arouter.AuthenticateWithSIWx(ctx, baseURL, signer, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SIWx auth failed: %v\n", err)
		os.Exit(1)
	}
	if auth.JWT == "" {
		fmt.Fprintln(os.Stderr, "SIWx auth returned empty jwt")
		os.Exit(1)
	}

	fmt.Printf("JWT: %s...\n", truncate(auth.JWT, 24))
	fmt.Printf("Tenant: %s\n", auth.TenantID)

	client := arouter.NewClient(baseURL, auth.JWT, arouter.WithTimeout(30*time.Second))
	_, err = client.ChatCompletion(ctx, &arouter.ChatCompletionRequest{
		Model:    "test",
		Messages: []arouter.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		fmt.Fprintln(os.Stderr, "expected insufficient credits error")
		os.Exit(1)
	}
	fmt.Printf("Chat result: %v\n", err)
	fmt.Println("Go SIWx E2E PASS")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func trim0x(v string) string {
	if len(v) >= 2 && v[:2] == "0x" {
		return v[2:]
	}
	return v
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
