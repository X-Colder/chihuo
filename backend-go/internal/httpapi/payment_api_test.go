package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/payment"
)

func TestPaymentHTTPHandlerSandboxFlow(t *testing.T) {
	provider := payment.NewSandboxProvider()
	handler, err := NewPaymentHTTPHandler(provider, nil)
	if err != nil {
		t.Fatalf("new payment handler: %v", err)
	}

	expireAt := time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339)
	create := paymentHTTPJSONRequest(http.MethodPost, "/v1/payments/intents", `{
		"out_trade_no":"order_001",
		"order_id":"order_001",
		"openid":"openid_001",
		"amount":{"amount_cents":2200,"currency":"CNY"},
		"description":"低油鸡肉饭",
		"notify_url":"https://api.example.test/v1/payments/callback",
		"expire_at":"`+expireAt+`"
	}`)
	create.Header.Set("Idempotency-Key", "pay-order-001")
	response := httptest.NewRecorder()
	handler.HandleCreatePaymentIntent(response, create)
	if response.Code != http.StatusCreated {
		t.Fatalf("create payment status = %d, body = %s", response.Code, response.Body.String())
	}
	var prepay payment.PrepayResult
	decodePaymentData(t, response.Body.Bytes(), &prepay)
	if prepay.PrepayID == "" || prepay.PaymentStatus != payment.PaymentPrepayCreated {
		t.Fatalf("unexpected prepay result: %+v", prepay)
	}

	replay := paymentHTTPJSONRequest(http.MethodPost, "/v1/payments/intents", createBody(expireAt))
	replay.Header.Set("Idempotency-Key", "pay-order-001")
	replayResponse := httptest.NewRecorder()
	handler.HandleCreatePaymentIntent(replayResponse, replay)
	if replayResponse.Code != http.StatusCreated || replayResponse.Body.String() != response.Body.String() {
		t.Fatalf("payment replay mismatch: status=%d body=%s", replayResponse.Code, replayResponse.Body.String())
	}

	query := httptest.NewRecorder()
	handler.HandleQueryPayment(query, httptest.NewRequest(http.MethodGet, "/v1/payments?out_trade_no=order_001", nil))
	if query.Code != http.StatusOK {
		t.Fatalf("query payment status = %d, body = %s", query.Code, query.Body.String())
	}
	var paymentResult payment.PaymentResult
	decodePaymentData(t, query.Body.Bytes(), &paymentResult)
	if paymentResult.Status != payment.ProviderPaymentNotPay {
		t.Fatalf("payment status before sandbox completion = %s", paymentResult.Status)
	}

	if err := provider.MarkPaymentPaid("order_001"); err != nil {
		t.Fatalf("mark payment paid: %v", err)
	}
	refund := paymentHTTPJSONRequest(http.MethodPost, "/v1/payments/refunds", `{
		"out_refund_no":"refund_001",
		"out_trade_no":"order_001",
		"total_amount":{"amount_cents":2200,"currency":"CNY"},
		"refund_amount":{"amount_cents":2200,"currency":"CNY"},
		"reason":"消费者取消订单",
		"notify_url":"https://api.example.test/v1/refunds/callback"
	}`)
	refund.Header.Set("Idempotency-Key", "refund-order-001")
	refundResponse := httptest.NewRecorder()
	handler.HandleCreateRefund(refundResponse, refund)
	if refundResponse.Code != http.StatusCreated {
		t.Fatalf("create refund status = %d, body = %s", refundResponse.Code, refundResponse.Body.String())
	}
	var refundResult payment.RefundResult
	decodePaymentData(t, refundResponse.Body.Bytes(), &refundResult)
	if refundResult.Status != payment.ProviderRefundProcessing {
		t.Fatalf("refund status = %s", refundResult.Status)
	}

	refundQuery := httptest.NewRecorder()
	handler.HandleQueryRefund(refundQuery, httptest.NewRequest(http.MethodGet, "/v1/payments/refunds?out_refund_no=refund_001", nil))
	if refundQuery.Code != http.StatusOK {
		t.Fatalf("query refund status = %d, body = %s", refundQuery.Code, refundQuery.Body.String())
	}
}

func TestPaymentHTTPHandlerCallbacks(t *testing.T) {
	handler, err := NewPaymentHTTPHandler(payment.NewNoopProvider(), nil)
	if err != nil {
		t.Fatalf("new payment handler: %v", err)
	}

	paymentCallback := paymentHTTPJSONRequest(http.MethodPost, "/v1/payments/callback", `{
		"event_id":"evt_payment_001",
		"event_type":"PAYMENT_SUCCEEDED",
		"out_trade_no":"order_001",
		"transaction_id":"wx_001",
		"amount_cents":2200,
		"currency":"CNY",
		"occurred_at":"2026-09-02T12:00:00Z"
	}`)
	paymentResponse := httptest.NewRecorder()
	handler.HandlePaymentCallback(paymentResponse, paymentCallback)
	if paymentResponse.Code != http.StatusOK {
		t.Fatalf("payment callback status = %d, body = %s", paymentResponse.Code, paymentResponse.Body.String())
	}
	var paymentCallbackData struct {
		Accepted  bool   `json:"accepted"`
		DedupeKey string `json:"dedupe_key"`
		Event     struct {
			Type payment.CallbackEventType `json:"type"`
		} `json:"event"`
	}
	decodePaymentData(t, paymentResponse.Body.Bytes(), &paymentCallbackData)
	if !paymentCallbackData.Accepted ||
		paymentCallbackData.DedupeKey != "callback_evt_payment_001" ||
		paymentCallbackData.Event.Type != payment.CallbackPaymentSucceeded {
		t.Fatalf("unexpected payment callback data: %+v", paymentCallbackData)
	}

	refundCallback := paymentHTTPJSONRequest(http.MethodPost, "/v1/refunds/callback", `{
		"event_id":"evt_refund_001",
		"event_type":"REFUND_SUCCEEDED",
		"out_trade_no":"order_001",
		"out_refund_no":"refund_001",
		"refund_id":"wx_refund_001",
		"refund_amount_cents":2200,
		"currency":"CNY",
		"occurred_at":"2026-09-02T12:01:00Z"
	}`)
	refundResponse := httptest.NewRecorder()
	handler.HandleRefundCallback(refundResponse, refundCallback)
	if refundResponse.Code != http.StatusOK {
		t.Fatalf("refund callback status = %d, body = %s", refundResponse.Code, refundResponse.Body.String())
	}

	mismatch := paymentHTTPJSONRequest(http.MethodPost, "/v1/refunds/callback", `{
		"event_id":"evt_payment_002",
		"event_type":"PAYMENT_FAILED",
		"out_trade_no":"order_001",
		"occurred_at":"2026-09-02T12:02:00Z"
	}`)
	mismatchResponse := httptest.NewRecorder()
	handler.HandleRefundCallback(mismatchResponse, mismatch)
	if mismatchResponse.Code != http.StatusBadRequest {
		t.Fatalf("callback mismatch status = %d, body = %s", mismatchResponse.Code, mismatchResponse.Body.String())
	}
}

func TestPaymentHTTPHandlerValidationAndProviderErrors(t *testing.T) {
	handler, err := NewPaymentHTTPHandler(payment.NewSandboxProvider(), nil)
	if err != nil {
		t.Fatalf("new payment handler: %v", err)
	}

	missingKey := paymentHTTPJSONRequest(http.MethodPost, "/v1/payments/intents", createBody(time.Now().UTC().Add(time.Hour).Format(time.RFC3339)))
	response := httptest.NewRecorder()
	handler.HandleCreatePaymentIntent(response, missingKey)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "INVALID_IDEMPOTENCY_KEY") {
		t.Fatalf("missing idempotency key response = %d %s", response.Code, response.Body.String())
	}

	invalidNotify := paymentHTTPJSONRequest(http.MethodPost, "/v1/payments/intents", `{
		"out_trade_no":"order_002",
		"order_id":"order_002",
		"openid":"openid_002",
		"amount":{"amount_cents":2200,"currency":"CNY"},
		"description":"测试",
		"notify_url":"http://api.example.test/callback",
		"expire_at":"2026-09-02T13:00:00Z"
	}`)
	invalidNotify.Header.Set("Idempotency-Key", "pay-order-002")
	invalidResponse := httptest.NewRecorder()
	handler.HandleCreatePaymentIntent(invalidResponse, invalidNotify)
	if invalidResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid notify URL status = %d, body = %s", invalidResponse.Code, invalidResponse.Body.String())
	}

	notFound := httptest.NewRecorder()
	handler.HandleQueryPayment(notFound, httptest.NewRequest(http.MethodGet, "/v1/payments?out_trade_no=missing", nil))
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("not found query status = %d, body = %s", notFound.Code, notFound.Body.String())
	}
}

func createBody(expireAt string) string {
	return `{
		"out_trade_no":"order_001",
		"order_id":"order_001",
		"openid":"openid_001",
		"amount":{"amount_cents":2200,"currency":"CNY"},
		"description":"低油鸡肉饭",
		"notify_url":"https://api.example.test/v1/payments/callback",
		"expire_at":"` + expireAt + `"
	}`
}

func paymentHTTPJSONRequest(method, path, body string) *http.Request {
	return httptest.NewRequest(method, "http://example.test"+path, strings.NewReader(body))
}

func decodePaymentData(t *testing.T, body []byte, target any) {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode payment envelope: %v; body=%s", err, body)
	}
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		t.Fatalf("decode payment data: %v; body=%s", err, body)
	}
}
