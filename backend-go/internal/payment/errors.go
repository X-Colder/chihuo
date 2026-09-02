package payment

import "errors"

var (
	ErrInvalidRequest        = errors.New("payment: invalid request")
	ErrInvalidTransition     = errors.New("payment: invalid state transition")
	ErrIdempotencyConflict   = errors.New("payment: idempotency conflict")
	ErrNotFound              = errors.New("payment: resource not found")
	ErrAlreadyFinalized      = errors.New("payment: resource already finalized")
	ErrSignatureVerifier     = errors.New("payment: callback signature verifier is required")
	ErrInvalidCallback       = errors.New("payment: invalid callback")
	ErrUnsupportedCallback   = errors.New("payment: unsupported callback event")
	ErrProviderUnavailable   = errors.New("payment: provider unavailable")
	ErrCredentialUnavailable = errors.New("payment: merchant credentials unavailable")
)

type InvalidTransitionError struct {
	Aggregate string
	From      string
	To        string
}

func (e *InvalidTransitionError) Error() string {
	return e.Aggregate + ": cannot transition from " + e.From + " to " + e.To
}

func (e *InvalidTransitionError) Unwrap() error {
	return ErrInvalidTransition
}
