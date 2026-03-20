package arouter

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
)

// WalletSigner signs messages to prove wallet ownership.
type WalletSigner interface {
	Address() string
	SignMessage(message []byte) ([]byte, error)
}

// EvmWalletSigner signs messages using an ECDSA private key (EIP-191 personal_sign).
type EvmWalletSigner struct {
	key  *ecdsa.PrivateKey
	addr common.Address
}

func NewEvmWalletSigner(key *ecdsa.PrivateKey) *EvmWalletSigner {
	return &EvmWalletSigner{key: key, addr: crypto.PubkeyToAddress(key.PublicKey)}
}

func (s *EvmWalletSigner) Address() string { return s.addr.Hex() }

func (s *EvmWalletSigner) SignMessage(message []byte) ([]byte, error) {
	prefixed := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256([]byte(prefixed))
	sig, err := crypto.Sign(hash, s.key)
	if err != nil {
		return nil, err
	}
	sig[64] += 27 // EIP-155 v value
	return sig, nil
}

// SolanaWalletSigner signs messages using an ed25519 private key.
type SolanaWalletSigner struct {
	key     ed25519.PrivateKey
	address string
}

func NewSolanaWalletSigner(key ed25519.PrivateKey) *SolanaWalletSigner {
	pub := key.Public().(ed25519.PublicKey)
	return &SolanaWalletSigner{key: key, address: base58.Encode(pub)}
}

func (s *SolanaWalletSigner) Address() string { return s.address }

func (s *SolanaWalletSigner) SignMessage(message []byte) ([]byte, error) {
	return ed25519.Sign(s.key, message), nil
}

// SIWxAuthResult contains the result of a SIWx authentication.
type SIWxAuthResult struct {
	JWT      string `json:"jwt"`
	TenantID string `json:"tenant_id"`
	KeyID    string `json:"key_id"`
}

// SIWxOptions configures the SIWx authentication.
type SIWxOptions struct {
	// ChainID overrides the default chain ID. For EVM defaults to "8453" (Base).
	ChainID string
	// Statement is the human-readable text shown during signing.
	Statement string
}

// AuthenticateWithSIWx performs SIWx (Sign-In-With-X) authentication
// against the ARouter gateway. It signs a CAIP-122 message with the
// provided wallet signer and calls POST /v1/x402/auth to obtain a wallet JWT.
//
//	signer := arouter.NewEvmWalletSigner(privateKey)
//	result, err := arouter.AuthenticateWithSIWx(ctx, "https://api.arouter.ai", signer, nil)
//	client := arouter.NewClient("https://api.arouter.ai", result.JWT)
func AuthenticateWithSIWx(ctx context.Context, baseURL string, signer WalletSigner, opts *SIWxOptions) (*SIWxAuthResult, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	domain := parsed.Host // includes port for non-standard ports

	chainType := "evm"
	if !strings.HasPrefix(signer.Address(), "0x") {
		chainType = "solana"
	}

	message := createSIWxMessage(domain, signer.Address(), chainType, opts)

	sig, err := signer.SignMessage([]byte(message))
	if err != nil {
		return nil, fmt.Errorf("signing SIWx message: %w", err)
	}

	var sigStr string
	if chainType == "evm" {
		sigStr = "0x" + hex.EncodeToString(sig)
	} else {
		sigStr = base64.StdEncoding.EncodeToString(sig)
	}

	payload, _ := json.Marshal(map[string]string{
		"message":   message,
		"signature": sigStr,
	})
	header := base64.StdEncoding.EncodeToString(payload)

	authURL := strings.TrimRight(baseURL, "/") + "/v1/x402/auth"
	req, err := http.NewRequestWithContext(ctx, "POST", authURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("SIGN-IN-WITH-X", header)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SIWx request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SIWx auth failed (%d): %s", resp.StatusCode, string(body))
	}

	var result SIWxAuthResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing SIWx response: %w", err)
	}
	return &result, nil
}

func createSIWxMessage(domain, address, chainType string, opts *SIWxOptions) string {
	chainLabel := "Ethereum"
	if chainType == "solana" {
		chainLabel = "Solana"
	}

	statement := "Sign in to ARouter with your wallet"
	chainID := "8453"
	if chainType == "solana" {
		chainID = "5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp"
	}
	if opts != nil {
		if opts.Statement != "" {
			statement = opts.Statement
		}
		if opts.ChainID != "" {
			chainID = opts.ChainID
		}
	}

	nonce := generateNonce()
	issuedAt := time.Now().UTC().Format(time.RFC3339)

	return fmt.Sprintf(`%s wants you to sign in with your %s account:
%s

%s

URI: https://%s/v1/x402/auth
Version: 1
Chain ID: %s
Nonce: %s
Issued At: %s`, domain, chainLabel, address, statement, domain, chainID, nonce, issuedAt)
}

func generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
