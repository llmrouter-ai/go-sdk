package arouter

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// x402 protocol headers
const (
	x402HeaderPaymentRequired  = "PAYMENT-REQUIRED"
	x402HeaderPaymentSignature = "PAYMENT-SIGNATURE"
	x402HeaderPaymentResponse  = "PAYMENT-RESPONSE"
)

// ErrInsufficientCredits is returned when the server requires payment and the
// client has no x402 wallet configured (or payment fails).
var ErrInsufficientCredits = errors.New("arouter: insufficient credits (402)")

// X402Signer signs payment payloads for the x402 protocol.
// Implement this interface to provide custom signing logic for different
// blockchain networks (EVM, Solana, etc.).
type X402Signer interface {
	// SignPayment receives the decoded PaymentRequired object and returns
	// a signed PaymentPayload (JSON bytes) ready for Base64 encoding.
	SignPayment(ctx context.Context, paymentRequired json.RawMessage) ([]byte, error)
}

// X402Option configures the x402 payment behavior.
type X402Option func(*x402Config)

type x402Config struct {
	signer     X402Signer
	maxRetries int
}

// WithX402MaxRetries sets the maximum number of payment retry attempts (default 1).
func WithX402MaxRetries(n int) X402Option {
	return func(cfg *x402Config) { cfg.maxRetries = n }
}

// WithX402Signer enables x402 automatic payment on 402 responses using a
// custom signer implementation. The signer handles blockchain-specific
// signing logic.
//
//	client := arouter.NewClient(baseURL, apiKey,
//	    arouter.WithX402Signer(myEvmSigner),
//	)
//
// When the server responds with 402 + PAYMENT-REQUIRED header, the client
// automatically signs a payment and retries the request with the
// PAYMENT-SIGNATURE header.
func WithX402Signer(signer X402Signer, opts ...X402Option) Option {
	return func(c *Client) {
		cfg := &x402Config{signer: signer, maxRetries: 1}
		for _, o := range opts {
			o(cfg)
		}
		c.httpClient = wrapWithX402(c.httpClient, cfg)
	}
}

// EvmPrivateKeySigner is a convenience X402Signer that uses a raw ECDSA
// private key to sign EVM-compatible x402 payments.
//
// For production use, consider implementing X402Signer with a proper
// key management solution (hardware wallet, KMS, etc.).
type EvmPrivateKeySigner struct {
	PrivateKey *ecdsa.PrivateKey
}

// SignPayment builds and signs an EVM payment payload.
// This is a placeholder implementation — in production, use the official
// x402 Go client SDK (github.com/coinbase/x402/go) for proper signing.
func (s *EvmPrivateKeySigner) SignPayment(_ context.Context, paymentRequired json.RawMessage) ([]byte, error) {
	_ = paymentRequired
	return nil, fmt.Errorf("EvmPrivateKeySigner: use the x402 Go SDK (github.com/coinbase/x402/go) for production signing")
}

// ────────────────────────────────────────────────────────────────────────────
// x402Transport — HTTP transport wrapper
// ────────────────────────────────────────────────────────────────────────────

func wrapWithX402(client *http.Client, cfg *x402Config) *http.Client {
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &http.Client{
		Transport:     &x402Transport{base: transport, cfg: cfg},
		Timeout:       client.Timeout,
		CheckRedirect: client.CheckRedirect,
		Jar:           client.Jar,
	}
}

type x402Transport struct {
	base http.RoundTripper
	cfg  *x402Config
}

func (t *x402Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if resp.StatusCode != http.StatusPaymentRequired {
		return resp, nil
	}

	payReqHeader := resp.Header.Get(x402HeaderPaymentRequired)
	if payReqHeader == "" {
		return resp, nil
	}

	if t.cfg.signer == nil {
		return resp, nil
	}

	// Drain and close the 402 response body
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	payReqBytes, err := base64.StdEncoding.DecodeString(payReqHeader)
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		return resp, nil
	}

	for attempt := 0; attempt < t.cfg.maxRetries; attempt++ {
		signedPayload, signErr := t.cfg.signer.SignPayment(req.Context(), payReqBytes)
		if signErr != nil {
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			return resp, fmt.Errorf("arouter: x402 sign payment: %w", signErr)
		}

		retryReq := req.Clone(req.Context())
		retryReq.Header.Set(x402HeaderPaymentSignature, base64.StdEncoding.EncodeToString(signedPayload))

		if req.Body != nil && req.GetBody != nil {
			retryReq.Body, _ = req.GetBody()
		}

		retryResp, retryErr := t.base.RoundTrip(retryReq)
		if retryErr != nil {
			return nil, retryErr
		}

		if retryResp.StatusCode != http.StatusPaymentRequired {
			return retryResp, nil
		}

		io.ReadAll(retryResp.Body)
		retryResp.Body.Close()
	}

	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return resp, nil
}
