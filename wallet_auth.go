package arouter

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
)

const walletAuthHeader = "X-Wallet-Auth"

// WalletSigner signs messages to prove wallet ownership.
type WalletSigner interface {
	// Address returns the wallet address (e.g. "0x...").
	Address() string
	// SignMessage signs a message and returns the 65-byte signature.
	SignMessage(message []byte) ([]byte, error)
}

// WithWalletAuth enables wallet-based authentication for x402 zero-registration.
//
// Every request will include an X-Wallet-Auth header with a signed message
// proving wallet ownership. No API key is needed — the wallet IS the identity.
//
//	signer := arouter.NewEvmWalletSigner(privateKey)
//	client := arouter.NewClient(baseURL, "",
//	    arouter.WithWalletAuth(signer),
//	    arouter.WithX402Signer(signer), // for automatic payment when balance is 0
//	)
func WithWalletAuth(signer WalletSigner) Option {
	return func(c *Client) {
		c.httpClient = wrapWithWalletAuth(c.httpClient, signer)
	}
}

// EvmWalletSigner implements both WalletSigner and X402Signer using an
// ECDSA private key. For production, consider using a KMS-backed signer.
type EvmWalletSigner struct {
	key  *ecdsa.PrivateKey
	addr common.Address
}

// NewEvmWalletSigner creates a signer from an ECDSA private key.
func NewEvmWalletSigner(key *ecdsa.PrivateKey) *EvmWalletSigner {
	return &EvmWalletSigner{
		key:  key,
		addr: crypto.PubkeyToAddress(key.PublicKey),
	}
}

func (s *EvmWalletSigner) Address() string {
	return s.addr.Hex()
}

func (s *EvmWalletSigner) SignMessage(message []byte) ([]byte, error) {
	prefixed := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256([]byte(prefixed))
	sig, err := crypto.Sign(hash, s.key)
	if err != nil {
		return nil, err
	}
	sig[64] += 27 // Ethereum convention: v = 27 or 28
	return sig, nil
}

// SolanaWalletSigner implements WalletSigner using an ed25519 private key.
type SolanaWalletSigner struct {
	key ed25519.PrivateKey
}

// NewSolanaWalletSigner creates a signer from an ed25519 private key.
func NewSolanaWalletSigner(key ed25519.PrivateKey) *SolanaWalletSigner {
	return &SolanaWalletSigner{key: key}
}

func (s *SolanaWalletSigner) Address() string {
	return base58.Encode(s.key.Public().(ed25519.PublicKey))
}

func (s *SolanaWalletSigner) SignMessage(message []byte) ([]byte, error) {
	return ed25519.Sign(s.key, message), nil
}

// walletAuthTransport wraps an HTTP transport to inject X-Wallet-Auth on every request.
type walletAuthTransport struct {
	base   http.RoundTripper
	signer WalletSigner
}

func wrapWithWalletAuth(client *http.Client, signer WalletSigner) *http.Client {
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &http.Client{
		Transport:     &walletAuthTransport{base: transport, signer: signer},
		Timeout:       client.Timeout,
		CheckRedirect: client.CheckRedirect,
		Jar:           client.Jar,
	}
}

func (t *walletAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Don't override if a real Bearer token is already set (API key user)
	if auth := req.Header.Get("Authorization"); auth != "" && auth != "Bearer " && auth != "Bearer" {
		return t.base.RoundTrip(req)
	}

	ts := time.Now().Unix()

	// Compute body hash for replay protection
	bodyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if req.Body != nil && req.GetBody != nil {
		bodyReader, _ := req.GetBody()
		if bodyReader != nil {
			bodyBytes, _ := io.ReadAll(bodyReader)
			if len(bodyBytes) > 0 {
				h := sha256.Sum256(bodyBytes)
				bodyHash = hex.EncodeToString(h[:])
			}
		}
	} else if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		if len(bodyBytes) > 0 {
			h := sha256.Sum256(bodyBytes)
			bodyHash = hex.EncodeToString(h[:])
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	msg := fmt.Sprintf("arouter:%d:%s:%s:%s", ts, req.Method, req.URL.Path, bodyHash)

	sig, err := t.signer.SignMessage([]byte(msg))
	if err != nil {
		return nil, fmt.Errorf("arouter: wallet auth sign: %w", err)
	}

	req = req.Clone(req.Context())
	addr := t.signer.Address()
	var sigStr string
	if strings.HasPrefix(addr, "0x") {
		sigStr = "0x" + hex.EncodeToString(sig)
	} else {
		sigStr = base58.Encode(sig)
	}
	req.Header.Set(walletAuthHeader, fmt.Sprintf("%s:%d:%s", addr, ts, sigStr))
	if req.Header.Get("Authorization") == "Bearer " || req.Header.Get("Authorization") == "Bearer" {
		req.Header.Del("Authorization")
	}

	return t.base.RoundTrip(req)
}
