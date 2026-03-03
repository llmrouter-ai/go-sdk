package arouter

import (
	"errors"
	"fmt"
)

// Sentinel errors for common API failure modes.
var (
	ErrUnauthorized  = errors.New("arouter: unauthorized")
	ErrForbidden     = errors.New("arouter: forbidden")
	ErrNotFound      = errors.New("arouter: not found")
	ErrRateLimited   = errors.New("arouter: rate limited")
	ErrQuotaExceeded = errors.New("arouter: quota exceeded")
	ErrBadRequest    = errors.New("arouter: bad request")
	ErrServerError   = errors.New("arouter: server error")
)

// APIError is returned when the ARouter API responds with a non-2xx status.
type APIError struct {
	StatusCode int    `json:"status_code"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("arouter: %d %s: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("arouter: %d: %s", e.StatusCode, e.Message)
}

// Unwrap maps the APIError to its corresponding sentinel error so callers can
// use errors.Is for common cases.
func (e *APIError) Unwrap() error {
	switch e.StatusCode {
	case 400:
		return ErrBadRequest
	case 401:
		return ErrUnauthorized
	case 403:
		return ErrForbidden
	case 404:
		return ErrNotFound
	case 429:
		if e.Code == "quota_exceeded" {
			return ErrQuotaExceeded
		}
		return ErrRateLimited
	default:
		if e.StatusCode >= 500 {
			return ErrServerError
		}
		return nil
	}
}
