package httpapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/X-Colder/chihuo/backend-go/internal/config"
)

func TestWeChatCode2SessionProvider(t *testing.T) {
	provider, err := NewWeChatCode2SessionProvider(config.Config{
		WeChatAppID:     "wx-test",
		WeChatAppSecret: "secret-test",
		WeChatLoginURL:  "https://wechat.test/jscode2session",
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	provider.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		query := request.URL.Query()
		if query.Get("appid") != "wx-test" {
			t.Errorf("appid = %q", query.Get("appid"))
		}
		if query.Get("secret") != "secret-test" {
			t.Errorf("secret = %q", query.Get("secret"))
		}
		if query.Get("js_code") != "code-test" {
			t.Errorf("js_code = %q", query.Get("js_code"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"openid":"openid-1","unionid":"unionid-1","session_key":"not-used"}`)),
			Header:     make(http.Header),
		}, nil
	})}
	identity, err := provider.Login(context.Background(), "code-test")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if identity.Subject != "unionid-1" {
		t.Fatalf("subject = %q", identity.Subject)
	}
}

func TestWeChatCode2SessionProviderUsesOpenIDFallback(t *testing.T) {
	provider, err := NewWeChatCode2SessionProvider(config.Config{
		WeChatAppID:     "wx-test",
		WeChatAppSecret: "secret-test",
		WeChatLoginURL:  "https://wechat.test/jscode2session",
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	provider.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"openid":"openid-only"}`)),
			Header:     make(http.Header),
		}, nil
	})}
	identity, err := provider.Login(context.Background(), "code-test")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if identity.Subject != "openid-only" {
		t.Fatalf("subject = %q", identity.Subject)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
