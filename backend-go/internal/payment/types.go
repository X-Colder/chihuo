package payment

import (
	"context"
	"strings"
	"time"
)

const (
	DefaultCurrency = "CNY"
	DefaultMaxBody  = 1 << 20
)

type PaymentStatus string

const (
	PaymentPending           PaymentStatus = "PENDING"
	PaymentPrepayCreated     PaymentStatus = "PREPAY_CREATED"
	PaymentPaying            PaymentStatus = "PAYING"
	PaymentPaid              PaymentStatus = "PAID"
	PaymentClosed            PaymentStatus = "CLOSED"
	PaymentFailed            PaymentStatus = "FAILED"
	PaymentPartiallyRefunded PaymentStatus = "PARTIALLY_REFUNDED"
	PaymentRefunded          PaymentStatus = "REFUNDED"
)

func (s PaymentStatus) Valid() bool {
	switch s {
	case PaymentPending, PaymentPrepayCreated, PaymentPaying, PaymentPaid,
		PaymentClosed, PaymentFailed, PaymentPartiallyRefunded, PaymentRefunded:
		return true
	default:
		return false
	}
}

func (s PaymentStatus) Terminal() bool {
	switch s {
	case PaymentClosed, PaymentFailed, PaymentRefunded:
		return true
	default:
		return false
	}
}

type RefundStatus string

const (
	RefundPending    RefundStatus = "PENDING"
	RefundProcessing RefundStatus = "PROCESSING"
	RefundSucceeded  RefundStatus = "SUCCEEDED"
	RefundFailed     RefundStatus = "FAILED"
	RefundClosed     RefundStatus = "CLOSED"
)

func (s RefundStatus) Valid() bool {
	switch s {
	case RefundPending, RefundProcessing, RefundSucceeded, RefundFailed, RefundClosed:
		return true
	default:
		return false
	}
}

func (s RefundStatus) Terminal() bool {
	switch s {
	case RefundSucceeded, RefundClosed:
		return true
	default:
		return false
	}
}

type PaymentEvent string

const (
	PaymentEventPrepayCreated    PaymentEvent = "PREPAY_CREATED"
	PaymentEventPaymentPending   PaymentEvent = "PAYMENT_PENDING"
	PaymentEventPaymentPaying    PaymentEvent = "PAYMENT_PAYING"
	PaymentEventPaymentSucceeded PaymentEvent = "PAYMENT_SUCCEEDED"
	PaymentEventPaymentClosed    PaymentEvent = "PAYMENT_CLOSED"
	PaymentEventPaymentFailed    PaymentEvent = "PAYMENT_FAILED"
	PaymentEventRefundPartially  PaymentEvent = "REFUND_PARTIALLY_SUCCEEDED"
	PaymentEventRefundSucceeded  PaymentEvent = "REFUND_SUCCEEDED"
)

type RefundEvent string

const (
	RefundEventProcessing RefundEvent = "REFUND_PROCESSING"
	RefundEventSucceeded  RefundEvent = "REFUND_SUCCEEDED"
	RefundEventFailed     RefundEvent = "REFUND_FAILED"
	RefundEventClosed     RefundEvent = "REFUND_CLOSED"
)

type ProviderPaymentStatus string

const (
	ProviderPaymentNotPay   ProviderPaymentStatus = "NOTPAY"
	ProviderPaymentPaying   ProviderPaymentStatus = "USERPAYING"
	ProviderPaymentSuccess  ProviderPaymentStatus = "SUCCESS"
	ProviderPaymentClosed   ProviderPaymentStatus = "CLOSED"
	ProviderPaymentRevoked  ProviderPaymentStatus = "REVOKED"
	ProviderPaymentError    ProviderPaymentStatus = "PAYERROR"
	ProviderPaymentRefunded ProviderPaymentStatus = "REFUND"
	ProviderPaymentUnknown  ProviderPaymentStatus = "UNKNOWN"
)

type ProviderRefundStatus string

const (
	ProviderRefundProcessing ProviderRefundStatus = "PROCESSING"
	ProviderRefundSuccess    ProviderRefundStatus = "SUCCESS"
	ProviderRefundClosed     ProviderRefundStatus = "CLOSED"
	ProviderRefundAbnormal   ProviderRefundStatus = "ABNORMAL"
	ProviderRefundUnknown    ProviderRefundStatus = "UNKNOWN"
)

type Money struct {
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
}

func (m Money) Validate() error {
	if m.AmountCents <= 0 {
		return ErrInvalidRequest
	}
	if strings.TrimSpace(m.Currency) == "" {
		return ErrInvalidRequest
	}
	return nil
}

func (m Money) normalized() Money {
	m.Currency = strings.ToUpper(strings.TrimSpace(m.Currency))
	if m.Currency == "" {
		m.Currency = DefaultCurrency
	}
	return m
}

type IdempotencyKey struct {
	Value string `json:"value"`
}

func (k IdempotencyKey) String() string {
	return k.Value
}

func (k IdempotencyKey) Valid() bool {
	return strings.TrimSpace(k.Value) == k.Value && len(k.Value) >= 8 && len(k.Value) <= 128
}

type CreateJSAPIPrepayRequest struct {
	IdempotencyKey IdempotencyKey `json:"-"`
	OutTradeNo     string         `json:"out_trade_no"`
	OrderID        string         `json:"order_id"`
	OpenID         string         `json:"openid"`
	Amount         Money          `json:"amount"`
	Description    string         `json:"description"`
	Attach         string         `json:"attach,omitempty"`
	NotifyURL      string         `json:"notify_url"`
	ExpireAt       time.Time      `json:"expire_at"`
}

type JSAPIPaymentParams struct {
	AppID     string `json:"app_id,omitempty"`
	TimeStamp string `json:"time_stamp,omitempty"`
	NonceStr  string `json:"nonce_str,omitempty"`
	Package   string `json:"package,omitempty"`
	SignType  string `json:"sign_type,omitempty"`
	PaySign   string `json:"pay_sign,omitempty"`
}

type PrepayResult struct {
	Provider      string             `json:"provider"`
	OutTradeNo    string             `json:"out_trade_no"`
	PrepayID      string             `json:"prepay_id"`
	PaymentStatus PaymentStatus      `json:"payment_status"`
	PaymentParams JSAPIPaymentParams `json:"payment_params"`
	ExpiresAt     time.Time          `json:"expires_at,omitempty"`
}

type QueryPaymentRequest struct {
	OutTradeNo    string
	TransactionID string
}

type PaymentResult struct {
	Provider      string                `json:"provider"`
	OutTradeNo    string                `json:"out_trade_no"`
	TransactionID string                `json:"transaction_id,omitempty"`
	Status        ProviderPaymentStatus `json:"status"`
	PaidAmount    Money                 `json:"paid_amount"`
	PaidAt        *time.Time            `json:"paid_at,omitempty"`
}

type ClosePaymentRequest struct {
	IdempotencyKey IdempotencyKey
	OutTradeNo     string
}

type ClosePaymentResult struct {
	Provider   string                `json:"provider"`
	OutTradeNo string                `json:"out_trade_no"`
	Status     ProviderPaymentStatus `json:"status"`
	ClosedAt   *time.Time            `json:"closed_at,omitempty"`
}

type ApplyRefundRequest struct {
	IdempotencyKey IdempotencyKey
	OutRefundNo    string
	OutTradeNo     string
	TransactionID  string
	TotalAmount    Money
	RefundAmount   Money
	Reason         string
	NotifyURL      string
}

type RefundResult struct {
	Provider     string               `json:"provider"`
	OutRefundNo  string               `json:"out_refund_no"`
	OutTradeNo   string               `json:"out_trade_no"`
	RefundID     string               `json:"refund_id,omitempty"`
	Status       ProviderRefundStatus `json:"status"`
	TotalAmount  Money                `json:"total_amount"`
	RefundAmount Money                `json:"refund_amount"`
	SuccessAt    *time.Time           `json:"success_at,omitempty"`
}

type QueryRefundRequest struct {
	OutRefundNo   string
	TransactionID string
}

type PaymentProvider interface {
	CreateJSAPIPrepay(ctx context.Context, req CreateJSAPIPrepayRequest) (PrepayResult, error)
	QueryPayment(ctx context.Context, req QueryPaymentRequest) (PaymentResult, error)
	ClosePayment(ctx context.Context, req ClosePaymentRequest) (ClosePaymentResult, error)
	ApplyRefund(ctx context.Context, req ApplyRefundRequest) (RefundResult, error)
	QueryRefund(ctx context.Context, req QueryRefundRequest) (RefundResult, error)
}
