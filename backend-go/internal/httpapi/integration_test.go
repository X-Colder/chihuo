package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/config"
	"github.com/X-Colder/chihuo/backend-go/internal/store"
)

type testClient struct {
	handler http.Handler
	t       *testing.T
}

type loginData struct {
	Token string `json:"token"`
	User  struct {
		ID         string `json:"id"`
		Role       string `json:"role"`
		MerchantID string `json:"merchant_id"`
	} `json:"user"`
}

func TestVerticalDemandOfferCampaignOrderFlow(t *testing.T) {
	cfg := config.Config{
		HTTPAddr:           ":0",
		JWTSecret:          "test-secret-that-is-longer-than-32-bytes",
		JWTIssuer:          "test",
		JWTTTL:             time.Hour,
		CORSAllowedOrigins: []string{"https://consumer.example"},
		DevLoginEnabled:    true,
	}
	api, err := New(cfg, store.NewMemoryStore(), nil)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	client := &testClient{handler: api.Handler(), t: t}

	admin := client.login("admin-code", "platform admin", "ADMIN", "")
	merchant := client.login("merchant-code", "local kitchen", "MERCHANT", "local kitchen")
	consumerA := client.login("consumer-a", "consumer a", "CONSUMER", "")
	consumerB := client.login("consumer-b", "consumer b", "CONSUMER", "")

	client.request("PATCH", "/v1/admin/merchants/"+merchant.User.MerchantID+"/review", admin.Token, "merchant-review", map[string]any{
		"status": "APPROVED",
	})

	demandPayload := map[string]any{
		"category":         "午餐",
		"title":            "低油定量鸡肉饭",
		"service_area":     "科技园",
		"serving_date":     "2026-09-05",
		"serving_time":     "12:00",
		"budget_min_cents": 1800,
		"budget_max_cents": 2600,
		"quantity":         1,
		"weight_min_grams": 330,
		"weight_max_grams": 380,
		"hard_constraints": []string{"不含花生"},
		"preferences":      []string{"少油"},
		"minimum_members":  2,
		"maximum_members":  20,
	}
	firstResponse := client.request("POST", "/v1/demands", consumerA.Token, "demand-create", demandPayload)
	if firstResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create demand status = %d, body = %s", firstResponse.StatusCode, firstResponse.Body)
	}
	var firstData struct {
		Demand struct {
			ID string `json:"id"`
		} `json:"demand"`
	}
	decodeData(t, firstResponse.Body, &firstData)
	if firstData.Demand.ID == "" {
		t.Fatal("created demand id is empty")
	}
	replayed := client.request("POST", "/v1/demands", consumerA.Token, "demand-create", demandPayload)
	if replayed.StatusCode != http.StatusCreated || replayed.Body != firstResponse.Body {
		t.Fatalf("idempotency replay did not return original response: status=%d body=%s", replayed.StatusCode, replayed.Body)
	}

	matched := client.request("POST", "/v1/demands", consumerB.Token, "demand-create-b", demandPayload)
	if matched.StatusCode != http.StatusOK {
		t.Fatalf("matching demand status = %d, body = %s", matched.StatusCode, matched.Body)
	}
	var matchedData struct {
		Matched bool `json:"matched"`
	}
	decodeData(t, matched.Body, &matchedData)
	if !matchedData.Matched {
		t.Fatal("second consumer was not matched into the existing demand")
	}

	demandID := firstData.Demand.ID
	client.request("PATCH", "/v1/admin/demands/"+demandID+"/review", admin.Token, "demand-review", map[string]any{
		"status": "OPEN",
	})

	offerResponse := client.request("POST", "/v1/merchant/offers", merchant.Token, "offer-create", map[string]any{
		"demand_id":            demandID,
		"unit_price_cents":     2200,
		"production_capacity":  20,
		"weight_grams":         350,
		"ingredients":          []string{"鸡肉", "米饭", "西兰花"},
		"allergens":            []string{"无花生"},
		"oil_level":            "LOW",
		"salt_level":           "LOW",
		"production_time":      "11:30",
		"shelf_life_minutes":   180,
		"storage_instructions": "常温不超过两小时",
	})
	if offerResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create offer status = %d, body = %s", offerResponse.StatusCode, offerResponse.Body)
	}
	var offerData struct {
		ID string `json:"id"`
	}
	decodeData(t, offerResponse.Body, &offerData)

	campaignResponse := client.request("POST", "/v1/merchant/campaigns", merchant.Token, "campaign-create", map[string]any{
		"demand_id":          demandID,
		"offer_id":           offerData.ID,
		"title":              "低油鸡肉饭预售",
		"unit_price_cents":   2200,
		"delivery_fee_cents": 500,
		"platform_fee_bps":   500,
		"minimum_orders":     1,
		"maximum_orders":     20,
		"starts_at":          "2026-09-05T10:00:00+08:00",
		"ends_at":            "2026-09-05T11:00:00+08:00",
		"pickup_point":       "科技园南门",
		"food_spec": map[string]any{
			"weight_grams":         350,
			"ingredients":          []string{"鸡肉", "米饭", "西兰花"},
			"allergens":            []string{"无花生"},
			"oil_level":            "LOW",
			"salt_level":           "LOW",
			"production_time":      "11:30",
			"shelf_life_minutes":   180,
			"storage_instructions": "常温不超过两小时",
		},
	})
	if campaignResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create campaign status = %d, body = %s", campaignResponse.StatusCode, campaignResponse.Body)
	}
	var campaignData struct {
		ID string `json:"id"`
	}
	decodeData(t, campaignResponse.Body, &campaignData)

	client.request("PATCH", "/v1/admin/campaigns/"+campaignData.ID+"/review", admin.Token, "campaign-review", map[string]any{
		"status": "OPEN",
	})
	orderPayload := map[string]any{
		"quantity":         2,
		"delivery_address": "科技园A座101",
		"contact_name":     "消费者A",
		"contact_phone":    "13800000000",
	}
	orderResponse := client.request("POST", "/v1/campaigns/"+campaignData.ID+"/orders", consumerA.Token, "order-create", orderPayload)
	if orderResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create order status = %d, body = %s", orderResponse.StatusCode, orderResponse.Body)
	}
	var orderData struct {
		TotalCents       int64 `json:"total_cents"`
		PlatformFeeCents int64 `json:"platform_fee_cents"`
	}
	decodeData(t, orderResponse.Body, &orderData)
	if orderData.TotalCents != 5120 || orderData.PlatformFeeCents != 220 {
		t.Fatalf("unexpected order total: %+v", orderData)
	}
	replayedOrder := client.request("POST", "/v1/campaigns/"+campaignData.ID+"/orders", consumerA.Token, "order-create", orderPayload)
	if replayedOrder.StatusCode != http.StatusCreated || replayedOrder.Body != orderResponse.Body {
		t.Fatalf("order idempotency replay failed: status=%d body=%s", replayedOrder.StatusCode, replayedOrder.Body)
	}
	ordersResponse := client.request("GET", "/v1/orders", consumerA.Token, "", nil)
	if ordersResponse.StatusCode != http.StatusOK {
		t.Fatalf("list orders status = %d, body = %s", ordersResponse.StatusCode, ordersResponse.Body)
	}
}

func TestAuthenticationCORSAndRequestID(t *testing.T) {
	cfg := config.Config{
		JWTSecret:          "test-secret-that-is-longer-than-32-bytes",
		JWTIssuer:          "test",
		JWTTTL:             time.Hour,
		CORSAllowedOrigins: []string{"https://consumer.example"},
		DevLoginEnabled:    true,
	}
	api, err := New(cfg, store.NewMemoryStore(), nil)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	client := &testClient{handler: api.Handler(), t: t}

	request := client.newRequest("GET", "/health/live", "", nil)
	request.Header.Set("Origin", "https://consumer.example")
	request.Header.Set("X-Request-ID", "test-request-id")
	response := client.do(request)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, body = %s", response.StatusCode, response.Body)
	}
	if response.Headers.Get("X-Request-ID") != "test-request-id" {
		t.Fatalf("request id header = %q", response.Headers.Get("X-Request-ID"))
	}
	if response.Headers.Get("Access-Control-Allow-Origin") != "https://consumer.example" {
		t.Fatalf("cors header = %q", response.Headers.Get("Access-Control-Allow-Origin"))
	}

	unauthorized := client.request("GET", "/v1/orders", "", "", nil)
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, body = %s", unauthorized.StatusCode, unauthorized.Body)
	}
}

func TestProductionRequiresExplicitPaymentProvider(t *testing.T) {
	cfg := config.Config{
		JWTSecret: "production-payment-test-secret-that-is-longer-than-32",
		JWTIssuer: "production-test",
		JWTTTL:    time.Hour,
	}
	if _, err := New(cfg, store.NewMemoryStore(), nil); err == nil {
		t.Fatal("production configuration without a payment provider should fail startup")
	}
}

type testResponse struct {
	StatusCode int
	Body       string
	Headers    http.Header
}

func (c *testClient) login(code, name, role, merchantName string) loginData {
	payload := map[string]any{"code": code, "name": name, "role": role}
	if merchantName != "" {
		payload["merchant_name"] = merchantName
	}
	response := c.request("POST", "/v1/auth/dev/wechat-login", "", "", payload)
	if response.StatusCode != http.StatusOK {
		c.t.Fatalf("login status = %d, body = %s", response.StatusCode, response.Body)
	}
	var data loginData
	decodeData(c.t, response.Body, &data)
	if data.Token == "" || data.User.ID == "" {
		c.t.Fatalf("login response missing session: %+v", data)
	}
	return data
}

func (c *testClient) request(method, path, token, idempotencyKey string, payload any) testResponse {
	request := c.newRequest(method, path, token, payload)
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}
	return c.do(request)
}

func (c *testClient) newRequest(method, path, token string, payload any) *http.Request {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			c.t.Fatalf("marshal request: %v", err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, "http://example.test"+path, body)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	return request
}

func (c *testClient) do(request *http.Request) testResponse {
	recorder := httptest.NewRecorder()
	c.handler.ServeHTTP(recorder, request)
	body := recorder.Body.Bytes()
	return testResponse{
		StatusCode: recorder.Code,
		Body:       string(body),
		Headers:    recorder.Header().Clone(),
	}
}

func decodeData(t *testing.T, body string, target any) {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("decode envelope: %v; body=%s", err, body)
	}
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		t.Fatalf("decode data: %v; body=%s", err, body)
	}
}
