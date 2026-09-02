package payment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

type NoopProvider struct {
	CallbackEventParser CallbackEventParser
}

var _ PaymentProvider = (*NoopProvider)(nil)
var _ CallbackEventParser = (*NoopProvider)(nil)

func NewNoopProvider() *NoopProvider {
	return &NoopProvider{
		CallbackEventParser: JSONCallbackEventParser{AllowUnsigned: true},
	}
}

func (p *NoopProvider) CreateJSAPIPrepay(ctx context.Context, req CreateJSAPIPrepayRequest) (PrepayResult, error) {
	if err := validateContext(ctx); err != nil {
		return PrepayResult{}, err
	}
	if err := validateCreateRequest(req, time.Now().UTC()); err != nil {
		return PrepayResult{}, err
	}
	return PrepayResult{
		Provider:      "noop",
		OutTradeNo:    req.OutTradeNo,
		PrepayID:      "noop_" + shortDigest(req.OutTradeNo),
		PaymentStatus: PaymentPrepayCreated,
		PaymentParams: JSAPIPaymentParams{
			AppID:     "noop",
			TimeStamp: strconv.FormatInt(time.Now().Unix(), 10),
			NonceStr:  "noop_" + shortDigest(req.IdempotencyKey.String()),
			Package:   "prepay_id=noop_" + shortDigest(req.OutTradeNo),
			SignType:  "NOOP",
			PaySign:   "",
		},
		ExpiresAt: req.ExpireAt,
	}, nil
}

func (p *NoopProvider) QueryPayment(ctx context.Context, req QueryPaymentRequest) (PaymentResult, error) {
	if err := validateContext(ctx); err != nil {
		return PaymentResult{}, err
	}
	if err := validateQueryPaymentRequest(req); err != nil {
		return PaymentResult{}, err
	}
	return PaymentResult{
		Provider:   "noop",
		OutTradeNo: req.OutTradeNo,
		Status:     ProviderPaymentNotPay,
		PaidAmount: Money{Currency: DefaultCurrency},
	}, nil
}

func (p *NoopProvider) ClosePayment(ctx context.Context, req ClosePaymentRequest) (ClosePaymentResult, error) {
	if err := validateContext(ctx); err != nil {
		return ClosePaymentResult{}, err
	}
	if err := validateCloseRequest(req); err != nil {
		return ClosePaymentResult{}, err
	}
	now := time.Now().UTC()
	return ClosePaymentResult{
		Provider:   "noop",
		OutTradeNo: req.OutTradeNo,
		Status:     ProviderPaymentClosed,
		ClosedAt:   &now,
	}, nil
}

func (p *NoopProvider) ApplyRefund(ctx context.Context, req ApplyRefundRequest) (RefundResult, error) {
	if err := validateContext(ctx); err != nil {
		return RefundResult{}, err
	}
	if err := validateRefundRequest(req); err != nil {
		return RefundResult{}, err
	}
	return RefundResult{
		Provider:     "noop",
		OutRefundNo:  req.OutRefundNo,
		OutTradeNo:   req.OutTradeNo,
		RefundID:     "noop_refund_" + shortDigest(req.OutRefundNo),
		Status:       ProviderRefundProcessing,
		TotalAmount:  req.TotalAmount.normalized(),
		RefundAmount: req.RefundAmount.normalized(),
	}, nil
}

func (p *NoopProvider) QueryRefund(ctx context.Context, req QueryRefundRequest) (RefundResult, error) {
	if err := validateContext(ctx); err != nil {
		return RefundResult{}, err
	}
	if err := validateQueryRefundRequest(req); err != nil {
		return RefundResult{}, err
	}
	return RefundResult{
		Provider: "noop",
		Status:   ProviderRefundProcessing,
	}, nil
}

func (p *NoopProvider) ParseCallback(ctx context.Context, request CallbackRequest) (CallbackEvent, error) {
	parser := p.CallbackEventParser
	if parser == nil {
		parser = JSONCallbackEventParser{AllowUnsigned: true}
	}
	return parser.ParseCallback(ctx, request)
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalidRequest
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func validateCreateRequest(req CreateJSAPIPrepayRequest, now time.Time) error {
	if !req.IdempotencyKey.Valid() ||
		req.OutTradeNo == "" ||
		req.OrderID == "" ||
		req.OpenID == "" ||
		req.Description == "" ||
		req.NotifyURL == "" ||
		req.ExpireAt.IsZero() ||
		!req.ExpireAt.After(now) {
		return ErrInvalidRequest
	}
	return req.Amount.normalized().Validate()
}

func validateQueryPaymentRequest(req QueryPaymentRequest) error {
	if req.OutTradeNo == "" && req.TransactionID == "" {
		return ErrInvalidRequest
	}
	return nil
}

func validateCloseRequest(req ClosePaymentRequest) error {
	if !req.IdempotencyKey.Valid() || req.OutTradeNo == "" {
		return ErrInvalidRequest
	}
	return nil
}

func validateRefundRequest(req ApplyRefundRequest) error {
	if !req.IdempotencyKey.Valid() ||
		req.OutRefundNo == "" ||
		req.OutTradeNo == "" {
		return ErrInvalidRequest
	}
	total := req.TotalAmount.normalized()
	refund := req.RefundAmount.normalized()
	if err := total.Validate(); err != nil {
		return err
	}
	if err := refund.Validate(); err != nil {
		return err
	}
	if total.Currency != refund.Currency || refund.AmountCents > total.AmountCents {
		return ErrInvalidRequest
	}
	return nil
}

func validateQueryRefundRequest(req QueryRefundRequest) error {
	if req.OutRefundNo == "" && req.TransactionID == "" {
		return ErrInvalidRequest
	}
	return nil
}

func shortDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])[:16]
}
