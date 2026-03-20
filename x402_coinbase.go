package arouter

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"

	x402sdk "github.com/coinbase/x402/go"
	x402http "github.com/coinbase/x402/go/http"
	evmclient "github.com/coinbase/x402/go/mechanisms/evm/exact/client"
	svmclient "github.com/coinbase/x402/go/mechanisms/svm/exact/client"
	evmsigners "github.com/coinbase/x402/go/signers/evm"
	svmsigners "github.com/coinbase/x402/go/signers/svm"
)

// WithX402CoinbasePayment configures the client with Coinbase's official x402 SDK
// for automatic on-chain USDC payment on EVM networks (Base, Ethereum, etc.).
//
// On the first request the gateway returns 402, the x402 SDK signs payment
// and retries. The response includes a wallet JWT in PAYMENT-RESPONSE
// which is cached and used as Bearer token for all subsequent requests.
//
//	key, _ := crypto.HexToECDSA("your-private-key-hex")
//	client := arouter.NewClient(baseURL, "",
//	    arouter.WithX402CoinbasePayment(key),
//	)
func WithX402CoinbasePayment(privateKey *ecdsa.PrivateKey) Option {
	return func(c *Client) {
		keyHex := hex.EncodeToString(crypto.FromECDSA(privateKey))
		evmSigner, err := evmsigners.NewClientSignerFromPrivateKey(keyHex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "arouter: WARNING: x402 EVM payment signer init failed: %v\n", err)
			return
		}

		x402Client := x402sdk.Newx402Client().
			Register("eip155:*", evmclient.NewExactEvmScheme(evmSigner, nil))

		c.httpClient = x402http.WrapHTTPClientWithPayment(
			c.httpClient,
			x402http.Newx402HTTPClient(x402Client),
		)
		wrapWithJWTCache(c, NewEvmWalletSigner(privateKey), nil)
	}
}

// WithX402CoinbasePaymentFromHex is a convenience wrapper accepting a hex private key string.
func WithX402CoinbasePaymentFromHex(hexKey string) Option {
	return func(c *Client) {
		hexKey = strings.TrimPrefix(hexKey, "0x")
		key, err := crypto.HexToECDSA(hexKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "arouter: WARNING: invalid EVM private key: %v\n", err)
			return
		}
		WithX402CoinbasePayment(key)(c)
	}
}

// WithX402SolanaPayment configures the client with Coinbase's x402 SDK
// for automatic on-chain SPL token payment on Solana networks.
//
//	solKey := ed25519.NewKeyFromSeed(seed)
//	client := arouter.NewClient(baseURL, "",
//	    arouter.WithX402SolanaPayment(solKey),
//	)
func WithX402SolanaPayment(privateKey ed25519.PrivateKey) Option {
	return func(c *Client) {
		b58Key := base58.Encode(privateKey)
		svmSigner, err := svmsigners.NewClientSignerFromPrivateKey(b58Key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "arouter: WARNING: x402 Solana payment signer init failed: %v\n", err)
			return
		}

		x402Client := x402sdk.Newx402Client().
			Register("solana:*", svmclient.NewExactSvmScheme(svmSigner, nil))

		c.httpClient = x402http.WrapHTTPClientWithPayment(
			c.httpClient,
			x402http.Newx402HTTPClient(x402Client),
		)
		wrapWithJWTCache(c, NewSolanaWalletSigner(privateKey), nil)
	}
}

// WithX402DualChainPayment configures both EVM and Solana x402 payment in one call.
// The authSigner is the wallet used for SIWx JWT renewal on 401 — it MUST match the
// wallet that was used for first registration. Pass the EVM signer if the first payment
// was on EVM, or the Solana signer if the first payment was on Solana.
func WithX402DualChainPayment(evmKey *ecdsa.PrivateKey, solKey ed25519.PrivateKey, authSigner WalletSigner) Option {
	return func(c *Client) {
		keyHex := hex.EncodeToString(crypto.FromECDSA(evmKey))
		evmSigner, err := evmsigners.NewClientSignerFromPrivateKey(keyHex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "arouter: WARNING: x402 EVM signer init failed: %v\n", err)
			return
		}

		b58Key := base58.Encode(solKey)
		svmSigner, err := svmsigners.NewClientSignerFromPrivateKey(b58Key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "arouter: WARNING: x402 Solana signer init failed: %v\n", err)
			return
		}

		x402Client := x402sdk.Newx402Client().
			Register("eip155:*", evmclient.NewExactEvmScheme(evmSigner, nil)).
			Register("solana:*", svmclient.NewExactSvmScheme(svmSigner, nil))

		c.httpClient = x402http.WrapHTTPClientWithPayment(
			c.httpClient,
			x402http.Newx402HTTPClient(x402Client),
		)
		wrapWithJWTCache(c, authSigner, nil)
	}
}

// wrapWithJWTCache wraps the client's HTTP transport to cache the wallet JWT
// from PAYMENT-RESPONSE and inject it as Bearer token on subsequent requests.
func wrapWithJWTCache(c *Client, signer WalletSigner, opts *SIWxOptions) {
	base := c.httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	c.httpClient.Transport = &jwtCachingTransport{base: base, baseURL: c.baseURL, signer: signer, siwxOptions: opts}
}

type jwtCachingTransport struct {
	base        http.RoundTripper
	baseURL     string
	signer      WalletSigner
	siwxOptions *SIWxOptions
	jwt         string
	mu          sync.Mutex
}

func (t *jwtCachingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	cachedJWT := t.jwt
	t.mu.Unlock()

	// Snapshot body before first RoundTrip so we can replay on 401 retry.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("jwtCachingTransport: read body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	if cachedJWT != "" {
		auth := req.Header.Get("Authorization")
		if auth == "" || auth == "Bearer" || auth == "Bearer " {
			req = req.Clone(req.Context())
			req.Header.Set("Authorization", "Bearer "+cachedJWT)
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if paymentResponse := resp.Header.Get("PAYMENT-RESPONSE"); paymentResponse != "" {
		if decoded, err := base64.StdEncoding.DecodeString(paymentResponse); err == nil {
			var payload struct {
				JWT string `json:"jwt"`
			}
			if json.Unmarshal(decoded, &payload) == nil && payload.JWT != "" {
				t.mu.Lock()
				t.jwt = payload.JWT
				t.mu.Unlock()
			}
		}
	}

	if resp.StatusCode == http.StatusUnauthorized && t.signer != nil {
		resp.Body.Close()
		result, siwxErr := AuthenticateWithSIWx(req.Context(), t.baseURL, t.signer, t.siwxOptions)
		if siwxErr == nil && result.JWT != "" {
			t.mu.Lock()
			t.jwt = result.JWT
			t.mu.Unlock()
			retryReq := req.Clone(req.Context())
			retryReq.Header.Set("Authorization", "Bearer "+result.JWT)
			if bodyBytes != nil {
				retryReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
			return t.base.RoundTrip(retryReq)
		}
	}

	return resp, err
}
