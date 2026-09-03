package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/payment"
)

// PaymentHTTPHandler adapts the provider-neutral payment domain to HTTP.
// Authentication, order ownership, and persistence of payment state remain
// responsibilities of the main HTTP server and its application services.
type PaymentHTTPHandler struct {
	Provider       payment.PaymentProvider
	CallbackParser payment.CallbackEventParser
}

// NewPaymentHTTPHandler injects a payment provider and callback parser. When
// the parser is omitted, providers such as SandboxProvider and NoopProvider
// that implement CallbackEventParser are used automatically.
func NewPaymentHTTPHandler(provider payment.PaymentProvider, parser payment.CallbackEventParser) (*PaymentHTTPHandler, error) {
	if provider == nil {
		return nil, errors.New("payment provider is required")
	}
	if parser == nil {
		if candidate, ok := provider.(payment.CallbackEventParser); ok {
			parser = candidate
		}
	}
	if parser == nil {
		return nil, errors.New("payment callback parser is required")
	}
	return &PaymentHTTPHandler{
		Provider:       provider,
		CallbackParser: parser,
	}, nil
}

type paymentMoneyRequest struct {
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
}

type createPaymentIntentRequest struct {
	OutTradeNo  string              `json:"out_trade_no"`
	OrderID     string              `json:"order_id"`
	OpenID      string              `json:"openid"`
	Amount      paymentMoneyRequest `json:"amount"`
	Description string              `json:"description"`
	Attach      string              `json:"attach,omitempty"`
	NotifyURL   string              `json:"notify_url"`
	ExpireAt    time.Time           `json:"expire_at"`
}

type createRefundRequest struct {
	OutRefundNo   string              `json:"out_refund_no"`
	OutTradeNo    string              `json:"out_trade_no"`
	TransactionID string              `json:"transaction_id,omitempty"`
	TotalAmount   paymentMoneyRequest `json:"total_amount"`
	RefundAmount  paymentMoneyRequest `json:"refund_amount"`
	Reason        string              `json:"reason,omitempty"`
	NotifyURL     string              `json:"notify_url,omitempty"`
}

func (h *PaymentHTTPHandler) HandleCreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	var input createPaymentIntentRequest
	if err := decodeJSON(body, &input); err != nil {
		h.writeError(w, r, err)
		return
	}
	key, err := payment.NormalizeIdempotencyKey(r.Header.Get("Idempotency-Key"))
	if err != nil {
		h.writeError(w, r, newRequestError(http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "Idempotency-Key is required and must be 8-128 characters without surrounding whitespace", nil))
		return
	}
	if err := input.validate(); err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.Provider.CreateJSAPIPrepay(r.Context(), payment.CreateJSAPIPrepayRequest{
		IdempotencyKey: key,
		OutTradeNo:     strings.TrimSpace(input.OutTradeNo),
		OrderID:        strings.TrimSpace(input.OrderID),
		OpenID:         strings.TrimSpace(input.OpenID),
		Amount:         input.Amount.toDomain(),
		Description:    strings.TrimSpace(input.Description),
		Attach:         optionalText(input.Attach, 1024),
		NotifyURL:      strings.TrimSpace(input.NotifyURL),
		ExpireAt:       input.ExpireAt.UTC(),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeData(w, http.StatusCreated, result)
}

func (h *PaymentHTTPHandler) HandleQueryPayment(w http.ResponseWriter, r *http.Request) {
	request, err := queryPaymentRequestFromURL(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.Provider.QueryPayment(r.Context(), request)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, result)
}

func (h *PaymentHTTPHandler) HandleCreateRefund(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	var input createRefundRequest
	if err := decodeJSON(body, &input); err != nil {
		h.writeError(w, r, err)
		return
	}
	key, err := payment.NormalizeIdempotencyKey(r.Header.Get("Idempotency-Key"))
	if err != nil {
		h.writeError(w, r, newRequestError(http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "Idempotency-Key is required and must be 8-128 characters without surrounding whitespace", nil))
		return
	}
	if err := input.validate(); err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.Provider.ApplyRefund(r.Context(), payment.ApplyRefundRequest{
		IdempotencyKey: key,
		OutRefundNo:    strings.TrimSpace(input.OutRefundNo),
		OutTradeNo:     strings.TrimSpace(input.OutTradeNo),
		TransactionID:  strings.TrimSpace(input.TransactionID),
		TotalAmount:    input.TotalAmount.toDomain(),
		RefundAmount:   input.RefundAmount.toDomain(),
		Reason:         optionalText(input.Reason, 256),
		NotifyURL:      strings.TrimSpace(input.NotifyURL),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeData(w, http.StatusCreated, result)
}

func (h *PaymentHTTPHandler) HandleQueryRefund(w http.ResponseWriter, r *http.Request) {
	request, err := queryRefundRequestFromURL(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.Provider.QueryRefund(r.Context(), request)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, result)
}

// HandlePaymentCallback parses and validates a provider callback. It does not
// mutate local payment state; the application layer must deduplicate the
// returned event using CallbackEvent.DedupeKey and persist the transition.
func (h *PaymentHTTPHandler) HandlePaymentCallback(w http.ResponseWriter, r *http.Request) {
	h.handleCallback(w, r, false)
}

// HandleRefundCallback parses and validates a provider refund callback. It
// does not mutate local refund state; the application layer owns persistence.
func (h *PaymentHTTPHandler) HandleRefundCallback(w http.ResponseWriter, r *http.Request) {
	h.handleCallback(w, r, true)
}

// ParsePaymentCallback parses a provider callback for an application service
// that needs to persist the event in the same transaction as its state change.
func (h *PaymentHTTPHandler) ParsePaymentCallback(ctx context.Context, request payment.CallbackRequest) (payment.CallbackEvent, payment.IdempotencyKey, error) {
	return h.parseCallback(ctx, request, false)
}

// ParseRefundCallback parses a provider refund callback for an application
// service that owns refund state persistence and event deduplication.
func (h *PaymentHTTPHandler) ParseRefundCallback(ctx context.Context, request payment.CallbackRequest) (payment.CallbackEvent, payment.IdempotencyKey, error) {
	return h.parseCallback(ctx, request, true)
}

func (h *PaymentHTTPHandler) handleCallback(w http.ResponseWriter, r *http.Request, refund bool) {
	body, err := readJSONBody(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	headers := make(map[string]string, len(r.Header))
	for name, values := range r.Header {
		if len(values) > 0 {
			headers[name] = values[0]
		}
	}
	event, dedupeKey, err := h.parseCallback(r.Context(), payment.CallbackRequest{
		Headers: headers,
		Body:    body,
	}, refund)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"accepted":   true,
		"dedupe_key": dedupeKey.String(),
		"event":      event,
	})
}

func (h *PaymentHTTPHandler) parseCallback(ctx context.Context, request payment.CallbackRequest, refund bool) (payment.CallbackEvent, payment.IdempotencyKey, error) {
	event, err := h.CallbackParser.ParseCallback(ctx, request)
	if err != nil {
		return payment.CallbackEvent{}, payment.IdempotencyKey{}, err
	}
	if err := validateCallbackEventKind(event.Type, refund); err != nil {
		return payment.CallbackEvent{}, payment.IdempotencyKey{}, err
	}
	dedupeKey, err := event.DedupeKey()
	if err != nil {
		return payment.CallbackEvent{}, payment.IdempotencyKey{}, err
	}
	return event, dedupeKey, nil
}

func (v createPaymentIntentRequest) validate() error {
	if err := requiredText(v.OutTradeNo, "out_trade_no", 128); err != nil {
		return err
	}
	if err := requiredText(v.OrderID, "order_id", 128); err != nil {
		return err
	}
	if err := requiredText(v.OpenID, "openid", 256); err != nil {
		return err
	}
	if err := requiredText(v.Description, "description", 127); err != nil {
		return err
	}
	if err := validateNotifyURL(v.NotifyURL, true); err != nil {
		return err
	}
	if v.ExpireAt.IsZero() || !v.ExpireAt.After(time.Now().UTC()) {
		return newRequestError(http.StatusBadRequest, "INVALID_EXPIRE_AT", "expire_at must be a future RFC3339 timestamp", nil)
	}
	return v.Amount.toDomain().Validate()
}

func (v createRefundRequest) validate() error {
	if err := requiredText(v.OutRefundNo, "out_refund_no", 128); err != nil {
		return err
	}
	if err := requiredText(v.OutTradeNo, "out_trade_no", 128); err != nil {
		return err
	}
	if err := validateNotifyURL(v.NotifyURL, false); err != nil {
		return err
	}
	total := v.TotalAmount.toDomain()
	refund := v.RefundAmount.toDomain()
	if err := total.Validate(); err != nil {
		return newRequestError(http.StatusBadRequest, "INVALID_TOTAL_AMOUNT", "total_amount is invalid", nil)
	}
	if err := refund.Validate(); err != nil {
		return newRequestError(http.StatusBadRequest, "INVALID_REFUND_AMOUNT", "refund_amount is invalid", nil)
	}
	if total.Currency != refund.Currency || refund.AmountCents > total.AmountCents {
		return newRequestError(http.StatusBadRequest, "INVALID_REFUND_AMOUNT", "refund_amount must not exceed total_amount and currencies must match", nil)
	}
	return nil
}

func (v paymentMoneyRequest) toDomain() payment.Money {
	return payment.Money{
		AmountCents: v.AmountCents,
		Currency:    v.Currency,
	}
}

func queryPaymentRequestFromURL(r *http.Request) (payment.QueryPaymentRequest, error) {
	outTradeNo := strings.TrimSpace(r.URL.Query().Get("out_trade_no"))
	transactionID := strings.TrimSpace(r.URL.Query().Get("transaction_id"))
	if outTradeNo == "" && transactionID == "" {
		return payment.QueryPaymentRequest{}, newRequestError(http.StatusBadRequest, "PAYMENT_IDENTIFIER_REQUIRED", "out_trade_no or transaction_id is required", nil)
	}
	if len(outTradeNo) > 128 || len(transactionID) > 128 {
		return payment.QueryPaymentRequest{}, newRequestError(http.StatusBadRequest, "INVALID_PAYMENT_IDENTIFIER", "payment identifier is too long", nil)
	}
	return payment.QueryPaymentRequest{
		OutTradeNo:    outTradeNo,
		TransactionID: transactionID,
	}, nil
}

func queryRefundRequestFromURL(r *http.Request) (payment.QueryRefundRequest, error) {
	outRefundNo := strings.TrimSpace(r.URL.Query().Get("out_refund_no"))
	transactionID := strings.TrimSpace(r.URL.Query().Get("transaction_id"))
	if outRefundNo == "" && transactionID == "" {
		return payment.QueryRefundRequest{}, newRequestError(http.StatusBadRequest, "REFUND_IDENTIFIER_REQUIRED", "out_refund_no or transaction_id is required", nil)
	}
	if len(outRefundNo) > 128 || len(transactionID) > 128 {
		return payment.QueryRefundRequest{}, newRequestError(http.StatusBadRequest, "INVALID_REFUND_IDENTIFIER", "refund identifier is too long", nil)
	}
	return payment.QueryRefundRequest{
		OutRefundNo:   outRefundNo,
		TransactionID: transactionID,
	}, nil
}

func validateNotifyURL(value string, required bool) error {
	value = strings.TrimSpace(value)
	if value == "" {
		if required {
			return newRequestError(http.StatusBadRequest, "INVALID_NOTIFY_URL", "notify_url is required", nil)
		}
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return newRequestError(http.StatusBadRequest, "INVALID_NOTIFY_URL", "notify_url must be an absolute HTTPS URL", nil)
	}
	return nil
}

func validateCallbackEventKind(eventType payment.CallbackEventType, refund bool) error {
	isRefundEvent := eventType == payment.CallbackRefundProcessing ||
		eventType == payment.CallbackRefundSucceeded ||
		eventType == payment.CallbackRefundFailed ||
		eventType == payment.CallbackRefundClosed
	if refund != isRefundEvent {
		return newRequestError(http.StatusBadRequest, "CALLBACK_EVENT_MISMATCH", "callback event does not match this endpoint", nil)
	}
	return nil
}

func (h *PaymentHTTPHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var typed *requestError
	if errors.As(err, &typed) {
		writeError(w, r, typed.Status, typed.Code, typed.Message, typed.Details)
		return
	}
	switch {
	case errors.Is(err, payment.ErrInvalidRequest):
		writeError(w, r, http.StatusBadRequest, "PAYMENT_INVALID_REQUEST", "payment request is invalid", nil)
	case errors.Is(err, payment.ErrInvalidCallback), errors.Is(err, payment.ErrUnsupportedCallback):
		writeError(w, r, http.StatusBadRequest, "PAYMENT_INVALID_CALLBACK", "payment callback is invalid", nil)
	case errors.Is(err, payment.ErrSignatureVerifier), errors.Is(err, payment.ErrCredentialUnavailable):
		writeError(w, r, http.StatusServiceUnavailable, "PAYMENT_PROVIDER_NOT_CONFIGURED", "payment provider is not configured", nil)
	case errors.Is(err, payment.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "PAYMENT_NOT_FOUND", "payment resource not found", nil)
	case errors.Is(err, payment.ErrIdempotencyConflict):
		writeError(w, r, http.StatusConflict, "PAYMENT_IDEMPOTENCY_CONFLICT", "payment idempotency key conflicts with an existing request", nil)
	case errors.Is(err, payment.ErrAlreadyFinalized), errors.Is(err, payment.ErrInvalidTransition):
		writeError(w, r, http.StatusConflict, "PAYMENT_INVALID_STATE", "payment state does not allow this operation", nil)
	case errors.Is(err, payment.ErrProviderUnavailable):
		writeError(w, r, http.StatusBadGateway, "PAYMENT_PROVIDER_UNAVAILABLE", "payment provider is temporarily unavailable", nil)
	default:
		writeError(w, r, http.StatusBadGateway, "PAYMENT_PROVIDER_ERROR", "payment provider request failed", nil)
	}
}
