package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/config"
	"github.com/X-Colder/chihuo/backend-go/internal/store"
)

func BenchmarkHealthParallel(b *testing.B) {
	api, err := New(config.Config{
		JWTSecret:          "benchmark-secret-that-is-longer-than-32-bytes",
		JWTIssuer:          "benchmark",
		JWTTTL:             time.Hour,
		PaymentProvider:    "sandbox",
		RateLimitRPS:       1_000_000,
		RateLimitBurst:     1_000_000,
		CORSAllowedOrigins: []string{"*"},
	}, store.NewMemoryStore(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		b.Fatalf("new server: %v", err)
	}
	handler := api.Handler()
	b.Cleanup(func() { _ = api.Close() })

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				b.Fatalf("health status = %d", recorder.Code)
			}
		}
	})
}
