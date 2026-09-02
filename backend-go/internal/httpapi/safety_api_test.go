package httpapi

import (
	"net/http"
	"testing"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/config"
	"github.com/X-Colder/chihuo/backend-go/internal/store"
)

func TestMerchantSafetyIncidentAPI(t *testing.T) {
	cfg := config.Config{
		JWTSecret:       "safety-api-test-secret-that-is-longer-than-32",
		JWTIssuer:       "safety-test",
		JWTTTL:          time.Hour,
		DevLoginEnabled: true,
		RateLimitRPS:    100,
		RateLimitBurst:  100,
	}
	api, err := New(cfg, store.NewMemoryStore(), nil)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	client := &testClient{handler: api.Handler(), t: t}
	merchant := client.login("safety-merchant", "safety merchant", "MERCHANT", "safety store")

	created := client.request("POST", "/v1/merchant/safety/incidents", merchant.Token, "safety-create", map[string]any{
		"category":    "PACKAGING",
		"severity":    "MEDIUM",
		"title":       "外包装破损",
		"description": "配送过程中发现汤盒外包装破损，已暂停涉事订单。",
		"batch_ids":   []string{},
		"order_ids":   []string{"order-safety-1"},
	})
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create safety incident status = %d, body = %s", created.StatusCode, created.Body)
	}
	var createdData struct {
		ID string `json:"id"`
	}
	decodeData(t, created.Body, &createdData)
	if createdData.ID == "" {
		t.Fatal("created safety incident id is empty")
	}

	listed := client.request("GET", "/v1/merchant/safety/incidents", merchant.Token, "", nil)
	if listed.StatusCode != http.StatusOK {
		t.Fatalf("list safety incidents status = %d, body = %s", listed.StatusCode, listed.Body)
	}

	transitioned := client.request("PATCH", "/v1/merchant/safety/incidents/"+createdData.ID, merchant.Token, "safety-transition", map[string]any{
		"status":             "CONTAINED",
		"containment_action": "暂停批次并保留证据",
		"evidence_ids":       []string{},
	})
	if transitioned.StatusCode != http.StatusOK {
		t.Fatalf("transition safety incident status = %d, body = %s", transitioned.StatusCode, transitioned.Body)
	}
}
