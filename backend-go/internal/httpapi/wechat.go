package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/config"
)

type WeChatCode2SessionProvider struct {
	appID     string
	appSecret string
	endpoint  string
	client    *http.Client
}

func NewWeChatCode2SessionProvider(cfg config.Config) (*WeChatCode2SessionProvider, error) {
	if cfg.WeChatAppID == "" || cfg.WeChatAppSecret == "" {
		return nil, fmt.Errorf("WECHAT_APP_ID and WECHAT_APP_SECRET are required")
	}
	if _, err := url.ParseRequestURI(cfg.WeChatLoginURL); err != nil {
		return nil, fmt.Errorf("invalid WECHAT_CODE2SESSION_URL: %w", err)
	}
	return &WeChatCode2SessionProvider{
		appID:     cfg.WeChatAppID,
		appSecret: cfg.WeChatAppSecret,
		endpoint:  cfg.WeChatLoginURL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}, nil
}

type code2SessionResponse struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	SessionKey string `json:"session_key"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func (p *WeChatCode2SessionProvider) Login(ctx context.Context, code string) (WeChatIdentity, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return WeChatIdentity{}, newRequestError(http.StatusBadRequest, "INVALID_CODE", "code is required", nil)
	}
	query := url.Values{}
	query.Set("appid", p.appID)
	query.Set("secret", p.appSecret)
	query.Set("js_code", code)
	query.Set("grant_type", "authorization_code")

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return WeChatIdentity{}, fmt.Errorf("create WeChat login request: %w", err)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return WeChatIdentity{}, fmt.Errorf("WeChat login request failed: %w", err)
	}
	defer response.Body.Close()

	var payload code2SessionResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return WeChatIdentity{}, fmt.Errorf("decode WeChat login response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || payload.ErrCode != 0 {
		return WeChatIdentity{}, fmt.Errorf("WeChat login rejected: code=%d message=%s", payload.ErrCode, payload.ErrMsg)
	}
	if payload.OpenID == "" {
		return WeChatIdentity{}, fmt.Errorf("WeChat login response did not include openid")
	}
	subject := payload.UnionID
	if subject == "" {
		subject = payload.OpenID
	}
	return WeChatIdentity{Subject: subject}, nil
}
