package payment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var identifierPartPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// NewPaymentIdempotencyKey creates a stable payment identifier for one local
// order. Repeating the same order must reuse this value.
func NewPaymentIdempotencyKey(orderID string) (IdempotencyKey, error) {
	return newScopedIdempotencyKey("pay", orderID)
}

// NewRefundIdempotencyKey creates a stable refund identifier. refundID should
// be a platform-generated refund request ID, not a provider transaction ID.
func NewRefundIdempotencyKey(paymentID, refundID string) (IdempotencyKey, error) {
	if err := validateIdentifierPart(paymentID); err != nil {
		return IdempotencyKey{}, fmt.Errorf("%w: payment id", err)
	}
	return newScopedIdempotencyKey("refund", paymentID+"_"+refundID)
}

// NewCallbackIdempotencyKey creates a deduplication key for a provider event.
func NewCallbackIdempotencyKey(eventID string) (IdempotencyKey, error) {
	return newScopedIdempotencyKey("callback", eventID)
}

func newScopedIdempotencyKey(scope, subject string) (IdempotencyKey, error) {
	if err := validateIdentifierPart(scope); err != nil {
		return IdempotencyKey{}, fmt.Errorf("%w: scope", err)
	}
	if err := validateIdentifierPart(subject); err != nil {
		return IdempotencyKey{}, fmt.Errorf("%w: subject", err)
	}
	value := scope + "_" + subject
	if len(value) > 128 {
		digest := sha256.Sum256([]byte(value))
		value = scope + "_" + hex.EncodeToString(digest[:])
	}
	return IdempotencyKey{Value: value}, nil
}

func validateIdentifierPart(value string) error {
	if !identifierPartPattern.MatchString(value) {
		return ErrInvalidRequest
	}
	return nil
}

// RequestFingerprint produces a stable hash that callers can persist next to
// an idempotency key to reject reuse with different business parameters.
func RequestFingerprint(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("%w: fingerprint: %v", ErrInvalidRequest, err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func NormalizeIdempotencyKey(value string) (IdempotencyKey, error) {
	original := value
	value = strings.TrimSpace(value)
	if len(value) < 8 || len(value) > 128 || original != value {
		return IdempotencyKey{}, ErrInvalidRequest
	}
	return IdempotencyKey{Value: value}, nil
}
