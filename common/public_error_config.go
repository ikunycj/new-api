package common

import (
	"errors"
	"strings"
	"sync/atomic"
)

// Option key persisted in the options table. The public error projection is
// opt-in so an upgrade keeps the historical response bodies until an operator
// explicitly turns normalization on.
const PublicErrorModeOptionKey = "PublicErrorMode"

// Projection modes for client-facing error responses.
//
// The internal error record (channel id/name, cause chain, failure scope,
// action, error_ref) is always kept as-is; only the bytes written to the
// client are affected.
const (
	// PublicErrorModePassthrough keeps the legacy behaviour: the response body
	// carries the internal classification and the upstream message verbatim.
	PublicErrorModePassthrough = "passthrough"
	// PublicErrorModeNormalized folds upstream operational state (exhausted
	// balance, empty account pool, credential failures) into generic 5xx bodies
	// and strips every internal field, while still telling the caller about
	// problems that are genuinely theirs (bad request, own quota, content
	// policy).
	PublicErrorModeNormalized = "normalized"
)

// ErrInvalidPublicErrorMode is returned for an unrecognised option value.
var ErrInvalidPublicErrorMode = errors.New("PublicErrorMode must be passthrough or normalized")

// DefaultPublicErrorMode deliberately preserves the existing contract. A
// malformed or unknown option value also falls back here, so a bad config can
// never start rewriting production responses.
const DefaultPublicErrorMode = PublicErrorModePassthrough

var publicErrorMode atomic.Value

func init() {
	publicErrorMode.Store(DefaultPublicErrorMode)
}

// PublicErrorMode returns the active projection mode.
func PublicErrorMode() string {
	mode, _ := publicErrorMode.Load().(string)
	if strings.EqualFold(strings.TrimSpace(mode), PublicErrorModeNormalized) {
		return PublicErrorModeNormalized
	}
	return PublicErrorModePassthrough
}

// SetPublicErrorMode stores the projection mode. Any value other than
// "normalized" is treated as passthrough.
func SetPublicErrorMode(mode string) {
	if strings.EqualFold(strings.TrimSpace(mode), PublicErrorModeNormalized) {
		publicErrorMode.Store(PublicErrorModeNormalized)
		return
	}
	publicErrorMode.Store(PublicErrorModePassthrough)
}

// IsPublicErrorNormalized reports whether client-facing errors must be
// projected. A false result means "write the legacy body".
func IsPublicErrorNormalized() bool {
	return PublicErrorMode() == PublicErrorModeNormalized
}

// NormalizePublicErrorMode validates and canonicalizes an option value.
func NormalizePublicErrorMode(value string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(value))
	switch mode {
	case PublicErrorModeNormalized:
		return PublicErrorModeNormalized, nil
	case PublicErrorModePassthrough:
		return PublicErrorModePassthrough, nil
	default:
		return "", ErrInvalidPublicErrorMode
	}
}
