package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type CallbackRequest struct {
	Headers map[string]string
	Body    []byte
}

func (r CallbackRequest) Header(name string) string {
	for key, value := range r.Headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

type CallbackEventType string

const (
	CallbackPaymentSucceeded CallbackEventType = "PAYMENT_SUCCEEDED"
	CallbackPaymentFailed    CallbackEventType = "PAYMENT_FAILED"
	CallbackPaymentClosed    CallbackEventType = "PAYMENT_CLOSED"
	CallbackRefundProcessing CallbackEventType = "REFUND_PROCESSING"
	CallbackRefundSucceeded  CallbackEventType = "REFUND_SUCCEEDED"
	CallbackRefundFailed     CallbackEventType = "REFUND_FAILED"
	CallbackRefundClosed     CallbackEventType = "REFUND_CLOSED"
)

type CallbackEvent struct {
	EventID       string                `json:"event_id"`
	Type          CallbackEventType     `json:"type"`
	OutTradeNo    string                `json:"out_trade_no"`
	TransactionID string                `json:"transaction_id,omitempty"`
	OutRefundNo   string                `json:"out_refund_no,omitempty"`
	RefundID      string                `json:"refund_id,omitempty"`
	PaymentStatus ProviderPaymentStatus `json:"payment_status,omitempty"`
	RefundStatus  ProviderRefundStatus  `json:"refund_status,omitempty"`
	Amount        Money                 `json:"amount,omitempty"`
	RefundAmount  Money                 `json:"refund_amount,omitempty"`
	OccurredAt    time.Time             `json:"occurred_at"`
}

func (e CallbackEvent) DedupeKey() (IdempotencyKey, error) {
	return NewCallbackIdempotencyKey(e.EventID)
}

type CallbackEventParser interface {
	ParseCallback(ctx context.Context, request CallbackRequest) (CallbackEvent, error)
}

// JSONCallbackEventParser parses a decrypted provider-neutral callback
// envelope. A real provider adapter should verify and decrypt the payload
// before handing it to the domain parser.
type JSONCallbackEventParser struct {
	MaxBodyBytes      int
	SignatureVerifier CallbackSignatureVerifier
	AllowUnsigned     bool
}

func (p JSONCallbackEventParser) ParseCallback(ctx context.Context, request CallbackRequest) (CallbackEvent, error) {
	if err := ctx.Err(); err != nil {
		return CallbackEvent{}, err
	}
	if p.SignatureVerifier == nil && !p.AllowUnsigned {
		return CallbackEvent{}, ErrSignatureVerifier
	}
	if p.SignatureVerifier != nil {
		if err := p.SignatureVerifier.Verify(ctx, request); err != nil {
			return CallbackEvent{}, err
		}
	}
	maxBytes := p.MaxBodyBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBody
	}
	if len(request.Body) == 0 || len(request.Body) > maxBytes {
		return CallbackEvent{}, fmt.Errorf("%w: body size", ErrInvalidCallback)
	}
	var payload callbackPayload
	decoder := json.NewDecoder(io.LimitReader(bytes.NewReader(request.Body), int64(maxBytes)+1))
	if err := decoder.Decode(&payload); err != nil {
		return CallbackEvent{}, fmt.Errorf("%w: invalid json: %v", ErrInvalidCallback, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return CallbackEvent{}, fmt.Errorf("%w: multiple json values", ErrInvalidCallback)
	}
	event, err := payload.event()
	if err != nil {
		return CallbackEvent{}, err
	}
	return event, nil
}

type callbackPayload struct {
	EventID           string                `json:"event_id"`
	Type              CallbackEventType     `json:"type"`
	EventType         CallbackEventType     `json:"event_type"`
	OutTradeNo        string                `json:"out_trade_no"`
	TransactionID     string                `json:"transaction_id"`
	OutRefundNo       string                `json:"out_refund_no"`
	RefundID          string                `json:"refund_id"`
	PaymentStatus     ProviderPaymentStatus `json:"payment_status"`
	RefundStatus      ProviderRefundStatus  `json:"refund_status"`
	AmountCents       int64                 `json:"amount_cents"`
	RefundAmountCents int64                 `json:"refund_amount_cents"`
	Currency          string                `json:"currency"`
	OccurredAt        string                `json:"occurred_at"`
}

func (p callbackPayload) event() (CallbackEvent, error) {
	eventType := p.EventType
	if eventType == "" {
		eventType = p.Type
	}
	if strings.TrimSpace(p.EventID) == "" || strings.TrimSpace(p.OutTradeNo) == "" {
		return CallbackEvent{}, fmt.Errorf("%w: event_id and out_trade_no are required", ErrInvalidCallback)
	}
	if !isSupportedCallbackEvent(eventType) {
		return CallbackEvent{}, fmt.Errorf("%w: %s", ErrUnsupportedCallback, eventType)
	}
	occurredAt, err := parseCallbackTime(p.OccurredAt)
	if err != nil {
		return CallbackEvent{}, err
	}
	event := CallbackEvent{
		EventID:       p.EventID,
		Type:          eventType,
		OutTradeNo:    p.OutTradeNo,
		TransactionID: p.TransactionID,
		OutRefundNo:   p.OutRefundNo,
		RefundID:      p.RefundID,
		PaymentStatus: p.PaymentStatus,
		RefundStatus:  p.RefundStatus,
		Amount:        Money{AmountCents: p.AmountCents, Currency: normalizedCurrency(p.Currency)},
		RefundAmount:  Money{AmountCents: p.RefundAmountCents, Currency: normalizedCurrency(p.Currency)},
		OccurredAt:    occurredAt,
	}
	switch eventType {
	case CallbackPaymentSucceeded:
		event.PaymentStatus = ProviderPaymentSuccess
		if event.Amount.AmountCents <= 0 {
			return CallbackEvent{}, fmt.Errorf("%w: amount_cents is required for payment success", ErrInvalidCallback)
		}
	case CallbackPaymentFailed:
		event.PaymentStatus = ProviderPaymentError
	case CallbackPaymentClosed:
		event.PaymentStatus = ProviderPaymentClosed
	case CallbackRefundProcessing:
		event.RefundStatus = ProviderRefundProcessing
		if err := validateRefundIdentifiers(event); err != nil {
			return CallbackEvent{}, err
		}
	case CallbackRefundSucceeded:
		event.RefundStatus = ProviderRefundSuccess
		if err := validateRefundIdentifiers(event); err != nil {
			return CallbackEvent{}, err
		}
		if event.RefundAmount.AmountCents <= 0 {
			return CallbackEvent{}, fmt.Errorf("%w: refund_amount_cents is required for refund success", ErrInvalidCallback)
		}
	case CallbackRefundFailed:
		event.RefundStatus = ProviderRefundAbnormal
		if err := validateRefundIdentifiers(event); err != nil {
			return CallbackEvent{}, err
		}
	case CallbackRefundClosed:
		event.RefundStatus = ProviderRefundClosed
		if err := validateRefundIdentifiers(event); err != nil {
			return CallbackEvent{}, err
		}
	}
	return event, nil
}

func validateRefundIdentifiers(event CallbackEvent) error {
	if strings.TrimSpace(event.OutRefundNo) == "" {
		return fmt.Errorf("%w: out_refund_no is required for refund events", ErrInvalidCallback)
	}
	return nil
}

func isSupportedCallbackEvent(eventType CallbackEventType) bool {
	switch eventType {
	case CallbackPaymentSucceeded, CallbackPaymentFailed, CallbackPaymentClosed,
		CallbackRefundProcessing, CallbackRefundSucceeded, CallbackRefundFailed,
		CallbackRefundClosed:
		return true
	default:
		return false
	}
}

func parseCallbackTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("%w: occurred_at is required", ErrInvalidCallback)
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: occurred_at must be RFC3339", ErrInvalidCallback)
	}
	return parsed.UTC(), nil
}

func normalizedCurrency(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return DefaultCurrency
	}
	return value
}
