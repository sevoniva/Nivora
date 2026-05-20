package authn

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
)

var (
	ErrInvalidToken  = errors.New("invalid OIDC token")
	ErrExpiredToken  = errors.New("OIDC token has expired")
	ErrInvalidIssuer = errors.New("OIDC token issuer mismatch")
	ErrNoJWKS        = errors.New("JWKS endpoint returned no keys")
)

// OIDCProvider validates OIDC JWTs against a JWKS endpoint.
type OIDCProvider struct {
	jwksURL    string
	client     *http.Client
	cachedJWKS *jwkSet
	cachedAt   time.Time
	cacheTTL   time.Duration
}

func NewOIDCProvider(jwksURL string) *OIDCProvider {
	return &OIDCProvider{
		jwksURL:  jwksURL,
		client:   http.DefaultClient,
		cacheTTL: 5 * time.Minute,
	}
}

var _ authusecase.OIDCProvider = (*OIDCProvider)(nil)

func (p *OIDCProvider) Validate(ctx context.Context, token, issuer, audience string) (authusecase.OIDCClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return authusecase.OIDCClaims{}, ErrInvalidToken
	}

	headerJSON, _ := base64.RawURLEncoding.DecodeString(parts[0])
	payloadJSON, _ := base64.RawURLEncoding.DecodeString(parts[1])

	var header struct{ Alg, Kid string }
	if err := json.Unmarshal(headerJSON, &header); err != nil || header.Kid == "" {
		return authusecase.OIDCClaims{}, ErrInvalidToken
	}

	var payload struct {
		Iss   string      `json:"iss"`
		Sub   string      `json:"sub"`
		Aud   interface{} `json:"aud"`
		Exp   int64       `json:"exp"`
		Name  string      `json:"name"`
		Email string      `json:"email"`
		Roles []string    `json:"roles"`
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return authusecase.OIDCClaims{}, ErrInvalidToken
	}

	// Validate issuer.
	if issuer != "" && payload.Iss != issuer {
		return authusecase.OIDCClaims{}, ErrInvalidIssuer
	}

	// Validate audience (string or array).
	if audience != "" {
		matched := false
		switch aud := payload.Aud.(type) {
		case string:
			matched = aud == audience
		case []interface{}:
			for _, a := range aud {
				if s, ok := a.(string); ok && s == audience {
					matched = true
					break
				}
			}
		}
		if !matched {
			return authusecase.OIDCClaims{}, fmt.Errorf("audience mismatch")
		}
	}

	// Validate expiration.
	if payload.Exp > 0 && time.Now().After(time.Unix(payload.Exp, 0)) {
		return authusecase.OIDCClaims{}, ErrExpiredToken
	}

	// Fetch JWKS and verify signature.
	jwks, err := p.fetchJWKS(ctx)
	if err != nil {
		return authusecase.OIDCClaims{}, fmt.Errorf("fetch JWKS: %w", err)
	}

	key, err := jwks.findKey(header.Kid)
	if err != nil {
		return authusecase.OIDCClaims{}, fmt.Errorf("find key: %w", err)
	}

	if err := verifyRS256(key, parts[0]+"."+parts[1], parts[2]); err != nil {
		return authusecase.OIDCClaims{}, err
	}

	claims := authusecase.OIDCClaims{
		Subject:  payload.Sub,
		Username: payload.Email,
		Roles:    payload.Roles,
	}
	if claims.Username == "" {
		claims.Username = payload.Name
	}
	if claims.Username == "" {
		claims.Username = payload.Sub
	}
	return claims, nil
}

type jwkSet struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (s *jwkSet) findKey(kid string) (*jwkKey, error) {
	for i := range s.Keys {
		if s.Keys[i].Kid == kid {
			return &s.Keys[i], nil
		}
	}
	return nil, fmt.Errorf("key %s not found in JWKS", kid)
}

func (p *OIDCProvider) fetchJWKS(ctx context.Context) (*jwkSet, error) {
	if p.cachedJWKS != nil && time.Since(p.cachedAt) < p.cacheTTL {
		return p.cachedJWKS, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.jwksURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS returned %d", resp.StatusCode)
	}

	var jwks jwkSet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}
	if len(jwks.Keys) == 0 {
		return nil, ErrNoJWKS
	}
	p.cachedJWKS = &jwks
	p.cachedAt = time.Now()
	return &jwks, nil
}

func verifyRS256(key *jwkKey, signingInput, signatureB64 string) error {
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return fmt.Errorf("decode JWK n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return fmt.Errorf("decode JWK e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	pubKey := &rsa.PublicKey{N: n, E: int(e.Int64())}

	signature, err := base64.RawURLEncoding.DecodeString(signatureB64)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	hash := sha256.Sum256([]byte(signingInput))
	return rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], signature)
}
