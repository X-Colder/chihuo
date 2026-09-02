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
	"github.com/X-Colder/chihuo/backend-go/internal/store"
)

type Server struct {
	store            store.Store
	signer           *auth.Signer
	config           config.Config
	logger           *slog.Logger
	provider         WeChatLoginProvider
	mux              *http.ServeMux
	idempotencyLocks sync.Map
}

func New(cfg config.Config, dataStore store.Store, logger *slog.Logger) (*Server, error) {
	return NewWithWeChatProvider(cfg, dataStore, logger, DevWeChatLoginProvider{})
}

func NewWithWeChatProvider(cfg config.Config, dataStore store.Store, logger *slog.Logger, provider WeChatLoginProvider) (*Server, error) {
	if logger == nil {
		logger = logging.New()
	}
	if provider == nil {
		return nil, errors.New("wechat login provider is required")
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
		mux:      http.NewServeMux(),
	}
	server.registerRoutes()
	return server, nil
}

func (s *Server) Handler() http.Handler {
	return s.withRecovery(s.withRequestID(s.withCORS(s.withLogging(s.mux))))
}

func (s *Server) Store() store.Store {
	return s.store
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
