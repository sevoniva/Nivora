package authn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestOIDCProviderExpiredToken(t *testing.T) {
	p := NewOIDCProvider("https://example.com/jwks")
	// Create an expired JWT (exp in the past).
	token := createTestJWT(t, "RS256", "kid-1", map[string]interface{}{
		"iss": "https://issuer.example",
		"sub": "user-1",
		"aud": "nivora",
		"exp": 1000000000, // 2001
		"iat": 1000000000,
	})
	_, err := p.Validate(context.Background(), token, "https://issuer.example", "nivora")
	if err == nil {
		t.Fatal("expected expired token error")
	}
}

func TestOIDCProviderInvalidIssuer(t *testing.T) {
	p := NewOIDCProvider("https://example.com/jwks")
	token := createTestJWT(t, "RS256", "kid-1", map[string]interface{}{
		"iss": "https://wrong-issuer.example",
		"sub": "user-1",
		"aud": "nivora",
		"exp": 2000000000, // 2033
	})
	_, err := p.Validate(context.Background(), token, "https://issuer.example", "nivora")
	if err != ErrInvalidIssuer {
		t.Fatalf("expected ErrInvalidIssuer, got %v", err)
	}
}

func TestOIDCProviderMissingKid(t *testing.T) {
	p := NewOIDCProvider("https://example.com/jwks")
	// Create a JWT without kid in header.
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"u1","exp":2000000000}`))
	token := header + "." + payload + ".bad-signature"

	_, err := p.Validate(context.Background(), token, "", "")
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for missing kid, got %v", err)
	}
}

func TestOIDCProviderJWTStructure(t *testing.T) {
	p := NewOIDCProvider("https://example.com/jwks")

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"no dots", "not-a-jwt"},
		{"one dot", "a.b"},
		{"four parts", "a.b.c.d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Validate(context.Background(), tt.token, "", "")
			if err != ErrInvalidToken {
				t.Errorf("expected ErrInvalidToken for %q, got %v", tt.token, err)
			}
		})
	}
}

func TestNewOIDCProviderDefaults(t *testing.T) {
	p := NewOIDCProvider("https://accounts.google.com/.well-known/openid-configuration/jwks")
	if p.jwksURL == "" {
		t.Fatal("jwksURL not set")
	}
	if p.cacheTTL == 0 {
		t.Fatal("cacheTTL not set")
	}
	if p.client == nil {
		t.Fatal("http client not set")
	}
}

// createTestJWT creates a simple JWT for testing header/payload checks.
func createTestJWT(t *testing.T, alg, kid string, payload map[string]interface{}) string {
	t.Helper()
	header := map[string]string{"alg": alg, "kid": kid}
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)
	token := strings.Join([]string{
		base64.RawURLEncoding.EncodeToString(headerJSON),
		base64.RawURLEncoding.EncodeToString(payloadJSON),
		base64.RawURLEncoding.EncodeToString([]byte("fake-signature")),
	}, ".")
	return token
}
