package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/auth"
	"github.com/X-Colder/chihuo/backend-go/internal/config"
	"github.com/X-Colder/chihuo/backend-go/internal/logging"
	"github.com/X-Colder/chihuo/backend-go/internal/ratelimit"
	"github.com/X-Colder/chihuo/backend-go/internal/store"
)

type Server struct {
	store            store.Store
	signer           *auth.Signer
	config           config.Config
	logger           *slog.Logger
	provider         WeChatLoginProvider
	limiter          ratelimit.Limiter
	mux              *http.ServeMux
	idempotencyLocks sync.Map
}

func New(cfg config.Config, dataStore store.Store, logger *slog.Logger) (*Server, error) {
	provider := WeChatLoginProvider(DevWeChatLoginProvider{})
	if cfg.WeChatAppID != "" || cfg.WeChatAppSecret != "" {
		realProvider, err := NewWeChatCode2SessionProvider(cfg)
		if err != nil {
			return nil, err
		}
		provider = realProvider
	}
	return NewWithWeChatProvider(cfg, dataStore, logger, provider)
}

func NewWithWeChatProvider(cfg config.Config, dataStore store.Store, logger *slog.Logger, provider WeChatLoginProvider) (*Server, error) {
	if logger == nil {
		logger = logging.New()
	}
	if provider == nil {
		return nil, errors.New("wechat login provider is required")
	}
	rps := cfg.RateLimitRPS
	if rps <= 0 {
		rps = 200
	}
	burst := cfg.RateLimitBurst
	if burst <= 0 {
		burst = 400
	}
	var limiter ratelimit.Limiter
	if cfg.RedisEnabled && cfg.RedisURL != "" {
		redisLimiter, err := ratelimit.NewRedis(cfg.RedisURL, cfg.RedisPassword, rps, burst)
		if err != nil {
			return nil, err
		}
		limiter = redisLimiter
	} else {
		limiter = ratelimit.NewLocal(rps, burst)
	}
	signer, err := auth.NewSigner(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTL)
	if err != nil {
		return nil, err
	}
	server := &Server{
		store:    dataStore,
		signer:   signer,
		config:   cfg,
		logger:   logger,
		provider: provider,
		limiter:  limiter,
		mux:      http.NewServeMux(),
	}
	server.registerRoutes()
	return server, nil
}

func (s *Server) Handler() http.Handler {
	return s.withRecovery(s.withRequestID(s.withCORS(s.withRateLimit(s.withLogging(s.mux)))))
}

func (s *Server) Store() store.Store {
	return s.store
}

func (s *Server) Close() error {
	if s.limiter == nil {
		return nil
	}
	return s.limiter.Close()
}

type WeChatIdentity struct {
	Subject string
}

type WeChatLoginProvider interface {
	Login(ctx context.Context, code string) (WeChatIdentity, error)
}

type DevWeChatLoginProvider struct{}

func (DevWeChatLoginProvider) Login(_ context.Context, code string) (WeChatIdentity, error) {
	if code == "" {
		return WeChatIdentity{}, newRequestError(http.StatusBadRequest, "INVALID_CODE", "code is required", nil)
	}
	return WeChatIdentity{Subject: "dev:" + code}, nil
}

func (s *Server) withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("panic recovered", "request_id", requestID(r.Context()), "panic", recovered)
				writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		s.logger.Info("http request",
			"request_id", requestID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func (s *Server) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health/live" || r.URL.Path == "/health/ready" {
			next.ServeHTTP(w, r)
			return
		}
		allowed, err := s.limiter.Allow(r.Context(), rateLimitKey(r))
		if err != nil {
			s.logger.Warn("rate limiter unavailable; allowing request", "error", err)
			next.ServeHTTP(w, r)
			return
		}
		if !allowed {
			w.Header().Set("Retry-After", "1")
			writeError(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
