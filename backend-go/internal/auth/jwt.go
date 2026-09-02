package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/domain"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type Claims struct {
	Subject    string      `json:"sub"`
	Name       string      `json:"name"`
	Role       domain.Role `json:"role"`
	MerchantID string      `json:"merchant_id,omitempty"`
	Issuer     string      `json:"iss"`
	IssuedAt   int64       `json:"iat"`
	ExpiresAt  int64       `json:"exp"`
	JTI        string      `json:"jti"`
}

type Signer struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

func NewSigner(secret, issuer string, ttl time.Duration) (*Signer, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT secret must be at least 32 bytes")
	}
	if issuer == "" || ttl <= 0 {
		return nil, errors.New("JWT issuer and TTL are required")
	}
	return &Signer{secret: []byte(secret), issuer: issuer, ttl: ttl}, nil
}

func (s *Signer) Sign(user domain.SessionUser, now time.Time) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	claims := Claims{
		Subject:    user.ID,
		Name:       user.Name,
		Role:       user.Role,
		MerchantID: user.MerchantID,
		Issuer:     s.issuer,
		IssuedAt:   now.Unix(),
		ExpiresAt:  now.Add(s.ttl).Unix(),
		JTI:        newTokenID(now),
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsigned := encodedHeader + "." + encodedClaims
	return unsigned + "." + s.sign(unsigned), nil
}

func (s *Signer) Verify(token string, now time.Time) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return Claims{}, ErrInvalidToken
	}
	expected := s.sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return Claims{}, ErrInvalidToken
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := decodeSegment(parts[0], &header); err != nil || header.Alg != "HS256" || header.Typ != "JWT" {
		return Claims{}, ErrInvalidToken
	}
	var claims Claims
	if err := decodeSegment(parts[1], &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.Issuer != s.issuer || claims.Subject == "" || !claims.Role.Valid() {
		return Claims{}, ErrInvalidToken
	}
	if claims.ExpiresAt <= now.Unix() {
		return Claims{}, ErrExpiredToken
	}
	return claims, nil
}

func (s *Signer) sign(input string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func decodeSegment(segment string, value any) error {
	decoded, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return err
	}
	return json.Unmarshal(decoded, value)
}

func newTokenID(now time.Time) string {
	return fmt.Sprintf("%d", now.UnixNano())
}
