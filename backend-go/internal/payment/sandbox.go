package payment

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type SandboxOption func(*SandboxProvider)

func WithSandboxClock(now func() time.Time) SandboxOption {
	return func(provider *SandboxProvider) {
		if now != nil {
			provider.now = now
		}
	}
}

type SandboxProvider struct {
	mu                  sync.Mutex
	now                 func() time.Time
	sequence            uint64
	payments            map[string]*sandboxPayment
	refunds             map[string]*sandboxRefund
	idempotency         map[string]sandboxIdempotencyRecord
	CallbackEventParser CallbackEventParser
}

var _ PaymentProvider = (*SandboxProvider)(nil)
var _ CallbackEventParser = (*SandboxProvider)(nil)

type sandboxPayment struct {
	request       CreateJSAPIPrepayRequest
	fingerprint   string
	prepay        PrepayResult
	transactionID string
	status        ProviderPaymentStatus
	paidAt        *time.Time
}

type sandboxRefund struct {
	request     ApplyRefundRequest
	fingerprint string
	result      RefundResult
}

type sandboxIdempotencyRecord struct {
	fingerprint string
	kind        string
	key         string
}

func NewSandboxProvider(options ...SandboxOption) *SandboxProvider {
	provider := &SandboxProvider{
		now:                 func() time.Time { return time.Now().UTC() },
		payments:            make(map[string]*sandboxPayment),
		refunds:             make(map[string]*sandboxRefund),
		idempotency:         make(map[string]sandboxIdempotencyRecord),
		CallbackEventParser: JSONCallbackEventParser{AllowUnsigned: true},
	}
	for _, option := range options {
		option(provider)
	}
	return provider
}

func (p *SandboxProvider) CreateJSAPIPrepay(ctx context.Context, req CreateJSAPIPrepayRequest) (PrepayResult, error) {
	if err := validateContext(ctx); err != nil {
		return PrepayResult{}, err
	}
	if err := validateCreateRequest(req, p.now()); err != nil {
		return PrepayResult{}, err
	}
	fingerprint, err := RequestFingerprint(req)
	if err != nil {
		return PrepayResult{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.checkIdempotency(req.IdempotencyKey, fingerprint, "payment"); err != nil {
		return PrepayResult{}, err
	}
	if existing, ok := p.payments[req.OutTradeNo]; ok {
		if existing.fingerprint != fingerprint {
			return PrepayResult{}, ErrIdempotencyConflict
		}
		return existing.prepay, nil
	}
	p.sequence++
	prepayID := fmt.Sprintf("sandbox_prepay_%d", p.sequence)
	payResult := PrepayResult{
		Provider:      "sandbox",
		OutTradeNo:    req.OutTradeNo,
		PrepayID:      prepayID,
		PaymentStatus: PaymentPrepayCreated,
		PaymentParams: JSAPIPaymentParams{
			AppID:     "sandbox",
			TimeStamp: fmt.Sprintf("%d", p.now().Unix()),
			NonceStr:  fmt.Sprintf("sandbox_nonce_%d", p.sequence),
			Package:   "prepay_id=" + prepayID,
			SignType:  "SANDBOX",
		},
		ExpiresAt: req.ExpireAt,
	}
	p.payments[req.OutTradeNo] = &sandboxPayment{
		request:     req,
		fingerprint: fingerprint,
		prepay:      payResult,
		status:      ProviderPaymentNotPay,
	}
	p.rememberIdempotency(req.IdempotencyKey, fingerprint, "payment")
	return payResult, nil
}

func (p *SandboxProvider) QueryPayment(ctx context.Context, req QueryPaymentRequest) (PaymentResult, error) {
	if err := validateContext(ctx); err != nil {
		return PaymentResult{}, err
	}
	if err := validateQueryPaymentRequest(req); err != nil {
		return PaymentResult{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	payment, ok := p.findPayment(req.OutTradeNo, req.TransactionID)
	if !ok {
		return PaymentResult{}, ErrNotFound
	}
	result := PaymentResult{
		Provider:      "sandbox",
		OutTradeNo:    payment.request.OutTradeNo,
		TransactionID: payment.transactionID,
		Status:        payment.status,
		PaidAmount:    Money{Currency: payment.request.Amount.normalized().Currency},
		PaidAt:        cloneTime(payment.paidAt),
	}
	if payment.status == ProviderPaymentSuccess || payment.status == ProviderPaymentRefunded {
		result.PaidAmount = payment.request.Amount.normalized()
	}
	return result, nil
}

func (p *SandboxProvider) ClosePayment(ctx context.Context, req ClosePaymentRequest) (ClosePaymentResult, error) {
	if err := validateContext(ctx); err != nil {
		return ClosePaymentResult{}, err
	}
	if err := validateCloseRequest(req); err != nil {
		return ClosePaymentResult{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	payment, ok := p.payments[req.OutTradeNo]
	if !ok {
		return ClosePaymentResult{}, ErrNotFound
	}
	if payment.status == ProviderPaymentSuccess || payment.status == ProviderPaymentRefunded {
		return ClosePaymentResult{}, ErrAlreadyFinalized
	}
	if payment.status == ProviderPaymentClosed {
		closedAt := p.now()
		return ClosePaymentResult{
			Provider:   "sandbox",
			OutTradeNo: req.OutTradeNo,
			Status:     ProviderPaymentClosed,
			ClosedAt:   &closedAt,
		}, nil
	}
	payment.status = ProviderPaymentClosed
	closedAt := p.now()
	return ClosePaymentResult{
		Provider:   "sandbox",
		OutTradeNo: req.OutTradeNo,
		Status:     ProviderPaymentClosed,
		ClosedAt:   &closedAt,
	}, nil
}

func (p *SandboxProvider) ApplyRefund(ctx context.Context, req ApplyRefundRequest) (RefundResult, error) {
	if err := validateContext(ctx); err != nil {
		return RefundResult{}, err
	}
	if err := validateRefundRequest(req); err != nil {
		return RefundResult{}, err
	}
	fingerprint, err := RequestFingerprint(req)
	if err != nil {
		return RefundResult{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.checkIdempotency(req.IdempotencyKey, fingerprint, "refund"); err != nil {
		return RefundResult{}, err
	}
	payment, ok := p.payments[req.OutTradeNo]
	if !ok {
		return RefundResult{}, ErrNotFound
	}
	if payment.status != ProviderPaymentSuccess && payment.status != ProviderPaymentRefunded {
		return RefundResult{}, fmt.Errorf("%w: payment is not successful", ErrInvalidRequest)
	}
	if existing, ok := p.refunds[req.OutRefundNo]; ok {
		if existing.fingerprint != fingerprint {
			return RefundResult{}, ErrIdempotencyConflict
		}
		return existing.result, nil
	}
	p.sequence++
	result := RefundResult{
		Provider:     "sandbox",
		OutRefundNo:  req.OutRefundNo,
		OutTradeNo:   req.OutTradeNo,
		RefundID:     fmt.Sprintf("sandbox_refund_%d", p.sequence),
		Status:       ProviderRefundProcessing,
		TotalAmount:  req.TotalAmount.normalized(),
		RefundAmount: req.RefundAmount.normalized(),
	}
	p.refunds[req.OutRefundNo] = &sandboxRefund{
		request:     req,
		fingerprint: fingerprint,
		result:      result,
	}
	p.rememberIdempotency(req.IdempotencyKey, fingerprint, "refund")
	return result, nil
}

func (p *SandboxProvider) QueryRefund(ctx context.Context, req QueryRefundRequest) (RefundResult, error) {
	if err := validateContext(ctx); err != nil {
		return RefundResult{}, err
	}
	if err := validateQueryRefundRequest(req); err != nil {
		return RefundResult{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, refund := range p.refunds {
		if (req.OutRefundNo != "" && refund.result.OutRefundNo == req.OutRefundNo) ||
			(req.TransactionID != "" && refund.request.TransactionID == req.TransactionID) {
			return refund.result, nil
		}
	}
	return RefundResult{}, ErrNotFound
}

func (p *SandboxProvider) ParseCallback(ctx context.Context, request CallbackRequest) (CallbackEvent, error) {
	parser := p.CallbackEventParser
	if parser == nil {
		parser = JSONCallbackEventParser{AllowUnsigned: true}
	}
	return parser.ParseCallback(ctx, request)
}

// MarkPaymentPaying simulates the user opening the payment sheet.
func (p *SandboxProvider) MarkPaymentPaying(outTradeNo string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	payment, ok := p.payments[outTradeNo]
	if !ok {
		return ErrNotFound
	}
	if payment.status != ProviderPaymentNotPay {
		return ErrAlreadyFinalized
	}
	payment.status = ProviderPaymentPaying
	return nil
}

// MarkPaymentPaid simulates the provider's successful payment callback.
func (p *SandboxProvider) MarkPaymentPaid(outTradeNo string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	payment, ok := p.payments[outTradeNo]
	if !ok {
		return ErrNotFound
	}
	if payment.status == ProviderPaymentSuccess || payment.status == ProviderPaymentRefunded {
		return nil
	}
	if payment.status == ProviderPaymentClosed {
		return ErrAlreadyFinalized
	}
	p.sequence++
	now := p.now()
	payment.status = ProviderPaymentSuccess
	payment.transactionID = fmt.Sprintf("sandbox_transaction_%d", p.sequence)
	payment.paidAt = &now
	return nil
}

// CompleteRefund simulates an asynchronous refund notification.
func (p *SandboxProvider) CompleteRefund(outRefundNo string, success bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	refund, ok := p.refunds[outRefundNo]
	if !ok {
		return ErrNotFound
	}
	if refund.result.Status == ProviderRefundSuccess || refund.result.Status == ProviderRefundClosed {
		return nil
	}
	if !success {
		refund.result.Status = ProviderRefundAbnormal
		return nil
	}
	now := p.now()
	refund.result.Status = ProviderRefundSuccess
	refund.result.SuccessAt = &now
	payment := p.payments[refund.result.OutTradeNo]
	if payment != nil {
		payment.status = ProviderPaymentRefunded
	}
	return nil
}

func (p *SandboxProvider) findPayment(outTradeNo, transactionID string) (*sandboxPayment, bool) {
	if outTradeNo != "" {
		payment, ok := p.payments[outTradeNo]
		return payment, ok
	}
	for _, payment := range p.payments {
		if payment.transactionID == transactionID {
			return payment, true
		}
	}
	return nil, false
}

func (p *SandboxProvider) checkIdempotency(key IdempotencyKey, fingerprint, kind string) error {
	if existing, ok := p.idempotency[key.String()]; ok {
		if existing.fingerprint != fingerprint || existing.kind != kind {
			return ErrIdempotencyConflict
		}
		return nil
	}
	return nil
}

func (p *SandboxProvider) rememberIdempotency(key IdempotencyKey, fingerprint, kind string) {
	p.idempotency[key.String()] = sandboxIdempotencyRecord{
		fingerprint: fingerprint,
		kind:        kind,
		key:         key.String(),
	}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
