package payment

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPaymentStateMachineAllowsProviderRaceButNeverRewinds(t *testing.T) {
	state, err := ApplyPaymentEvent(PaymentPending, PaymentEventPaymentSucceeded)
	if err != nil {
		t.Fatalf("payment success from pending: %v", err)
	}
	if state != PaymentPaid {
		t.Fatalf("state = %s, want %s", state, PaymentPaid)
	}
	if _, err := ApplyPaymentEvent(PaymentPaid, PaymentEventPaymentPending); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("payment state rewind error = %v", err)
	}
	if err := TransitionPayment(PaymentPaid, PaymentPaid); err != nil {
		t.Fatalf("same payment state should be idempotent: %v", err)
	}
}

func TestRefundStateMachineSupportsRetryAfterFailure(t *testing.T) {
	state, err := ApplyRefundEvent(RefundPending, RefundEventFailed)
	if err != nil {
		t.Fatalf("refund failure: %v", err)
	}
	state, err = ApplyRefundEvent(state, RefundEventProcessing)
	if err != nil {
		t.Fatalf("refund retry: %v", err)
	}
	if state != RefundProcessing {
		t.Fatalf("state = %s, want %s", state, RefundProcessing)
	}
}

func TestIdempotencyIdentifiersAreStableAndScoped(t *testing.T) {
	paymentKey, err := NewPaymentIdempotencyKey("order_123")
	if err != nil {
		t.Fatalf("payment key: %v", err)
	}
	repeated, err := NewPaymentIdempotencyKey("order_123")
	if err != nil {
		t.Fatalf("repeated payment key: %v", err)
	}
	if paymentKey != repeated || paymentKey.String() != "pay_order_123" {
		t.Fatalf("payment key = %#v, repeated = %#v", paymentKey, repeated)
	}
	refundKey, err := NewRefundIdempotencyKey("payment_123", "refund_001")
	if err != nil {
		t.Fatalf("refund key: %v", err)
	}
	if refundKey == paymentKey || refundKey.String() != "refund_payment_123_refund_001" {
		t.Fatalf("refund key = %#v", refundKey)
	}
	fingerprintA, err := RequestFingerprint(struct {
		Amount int64 `json:"amount"`
	}{100})
	if err != nil {
		t.Fatalf("fingerprint A: %v", err)
	}
	fingerprintB, err := RequestFingerprint(struct {
		Amount int64 `json:"amount"`
	}{200})
	if err != nil {
		t.Fatalf("fingerprint B: %v", err)
	}
	if fingerprintA == fingerprintB {
		t.Fatal("different requests must have different fingerprints")
	}
	if _, err := NormalizeIdempotencyKey(" key_123 "); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("whitespace idempotency key error = %v", err)
	}
}

func TestJSONCallbackParserRequiresVerificationUnlessExplicitlySandboxed(t *testing.T) {
	request := CallbackRequest{Body: []byte(`{
		"event_id":"evt_001",
		"event_type":"PAYMENT_SUCCEEDED",
		"out_trade_no":"order_123",
		"transaction_id":"txn_001",
		"amount_cents":2200,
		"currency":"CNY",
		"occurred_at":"2026-09-02T12:00:00+08:00"
	}`)}
	if _, err := (JSONCallbackEventParser{}).ParseCallback(context.Background(), request); !errors.Is(err, ErrSignatureVerifier) {
		t.Fatalf("unsigned parser error = %v", err)
	}
	event, err := (JSONCallbackEventParser{AllowUnsigned: true}).ParseCallback(context.Background(), request)
	if err != nil {
		t.Fatalf("sandbox callback parse: %v", err)
	}
	if event.Type != CallbackPaymentSucceeded ||
		event.PaymentStatus != ProviderPaymentSuccess ||
		event.Amount.AmountCents != 2200 ||
		event.OccurredAt.Location() != time.UTC {
		t.Fatalf("unexpected callback event: %+v", event)
	}
	key, err := event.DedupeKey()
	if err != nil || key.String() != "callback_evt_001" {
		t.Fatalf("callback dedupe key = %q, err = %v", key.String(), err)
	}
}

func TestJSONCallbackParserValidatesRefundEvents(t *testing.T) {
	body := []byte(`{
		"event_id":"evt_refund_001",
		"type":"REFUND_SUCCEEDED",
		"out_trade_no":"order_123",
		"out_refund_no":"refund_001",
		"refund_id":"refund_provider_001",
		"refund_amount_cents":2200,
		"currency":"CNY",
		"occurred_at":"2026-09-02T12:00:00Z"
	}`)
	event, err := (JSONCallbackEventParser{AllowUnsigned: true}).ParseCallback(context.Background(), CallbackRequest{Body: body})
	if err != nil {
		t.Fatalf("refund callback parse: %v", err)
	}
	if event.RefundStatus != ProviderRefundSuccess || event.RefundAmount.AmountCents != 2200 {
		t.Fatalf("unexpected refund event: %+v", event)
	}
}

func TestNoopProviderImplementsPaymentAndCallbackContracts(t *testing.T) {
	var provider PaymentProvider = NewNoopProvider()
	var parser CallbackEventParser = NewNoopProvider()
	ctx := context.Background()
	key, err := NewPaymentIdempotencyKey("order_123")
	if err != nil {
		t.Fatal(err)
	}
	prepay, err := provider.CreateJSAPIPrepay(ctx, CreateJSAPIPrepayRequest{
		IdempotencyKey: key,
		OutTradeNo:     "order_123",
		OrderID:        "order_123",
		OpenID:         "openid_123",
		Amount:         Money{AmountCents: 2200, Currency: "CNY"},
		Description:    "sandbox lunch",
		NotifyURL:      "https://example.test/payments/callback",
		ExpireAt:       time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("noop prepay: %v", err)
	}
	if prepay.PrepayID == "" || prepay.PaymentStatus != PaymentPrepayCreated {
		t.Fatalf("unexpected noop prepay: %+v", prepay)
	}
	_, err = parser.ParseCallback(ctx, CallbackRequest{Body: []byte(`{}`)})
	if !errors.Is(err, ErrInvalidCallback) {
		t.Fatalf("noop callback parser error = %v", err)
	}
}

func TestSandboxProviderIsIdempotentAndModelsRefundLifecycle(t *testing.T) {
	provider := NewSandboxProvider(WithSandboxClock(func() time.Time {
		return time.Date(2026, 9, 2, 4, 0, 0, 0, time.UTC)
	}))
	key, err := NewPaymentIdempotencyKey("order_123")
	if err != nil {
		t.Fatal(err)
	}
	req := CreateJSAPIPrepayRequest{
		IdempotencyKey: key,
		OutTradeNo:     "order_123",
		OrderID:        "order_123",
		OpenID:         "openid_123",
		Amount:         Money{AmountCents: 2200, Currency: "CNY"},
		Description:    "sandbox lunch",
		NotifyURL:      "https://example.test/payments/callback",
		ExpireAt:       time.Date(2026, 9, 2, 5, 0, 0, 0, time.UTC),
	}
	first, err := provider.CreateJSAPIPrepay(context.Background(), req)
	if err != nil {
		t.Fatalf("sandbox prepay: %v", err)
	}
	replayed, err := provider.CreateJSAPIPrepay(context.Background(), req)
	if err != nil {
		t.Fatalf("sandbox prepay replay: %v", err)
	}
	if first != replayed {
		t.Fatalf("idempotent prepay changed: first=%+v replay=%+v", first, replayed)
	}
	conflicting := req
	conflicting.Amount.AmountCents = 2300
	if _, err := provider.CreateJSAPIPrepay(context.Background(), conflicting); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("conflicting prepay error = %v", err)
	}
	if err := provider.MarkPaymentPaid("order_123"); err != nil {
		t.Fatalf("mark paid: %v", err)
	}
	refundKey, err := NewRefundIdempotencyKey("payment_123", "refund_001")
	if err != nil {
		t.Fatal(err)
	}
	refundReq := ApplyRefundRequest{
		IdempotencyKey: refundKey,
		OutRefundNo:    "refund_001",
		OutTradeNo:     "order_123",
		TotalAmount:    Money{AmountCents: 2200, Currency: "CNY"},
		RefundAmount:   Money{AmountCents: 2200, Currency: "CNY"},
		Reason:         "consumer cancellation",
	}
	refund, err := provider.ApplyRefund(context.Background(), refundReq)
	if err != nil {
		t.Fatalf("apply refund: %v", err)
	}
	if refund.Status != ProviderRefundProcessing {
		t.Fatalf("refund status = %s, want processing", refund.Status)
	}
	if err := provider.CompleteRefund("refund_001", true); err != nil {
		t.Fatalf("complete refund: %v", err)
	}
	queried, err := provider.QueryRefund(context.Background(), QueryRefundRequest{OutRefundNo: "refund_001"})
	if err != nil {
		t.Fatalf("query refund: %v", err)
	}
	if queried.Status != ProviderRefundSuccess || queried.SuccessAt == nil {
		t.Fatalf("unexpected completed refund: %+v", queried)
	}
	payment, err := provider.QueryPayment(context.Background(), QueryPaymentRequest{OutTradeNo: "order_123"})
	if err != nil {
		t.Fatalf("query payment: %v", err)
	}
	if payment.Status != ProviderPaymentRefunded {
		t.Fatalf("payment status = %s, want REFUND", payment.Status)
	}
}
