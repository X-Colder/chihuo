package payment

import (
	"context"
	"strings"
)

// MerchantCredentialSource keeps provider secrets outside the payment domain
// package. A production adapter should load these values from a secret manager
// or environment-backed implementation at request time.
type MerchantCredentialSource interface {
	Load(ctx context.Context) (MerchantCredentials, error)
}

type MerchantCredentials struct {
	AppID                 string
	MerchantID            string
	CertificateSerial     string
	APIV3Key              []byte
	MerchantPrivateKeyPEM []byte
}

type CallbackSignatureVerifier interface {
	Verify(ctx context.Context, request CallbackRequest) error
}

type ProviderOptions struct {
	AppID             string
	MerchantID        string
	APIBaseURL        string
	CredentialSource  MerchantCredentialSource
	SignatureVerifier CallbackSignatureVerifier
}

func (o ProviderOptions) ValidateForProduction() error {
	if strings.TrimSpace(o.AppID) == "" ||
		strings.TrimSpace(o.MerchantID) == "" ||
		strings.TrimSpace(o.APIBaseURL) == "" ||
		o.CredentialSource == nil ||
		o.SignatureVerifier == nil {
		return ErrCredentialUnavailable
	}
	return nil
}
