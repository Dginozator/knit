// Package yds provides IAM token management for Yandex Cloud.
package yds

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// ServiceAccountKey holds the service account authorized key data.
// This is the JSON file downloaded from Yandex Cloud console:
// IAM → Service Accounts → your account → Authorized keys → Create
type ServiceAccountKey struct {
	ID               string `json:"id"`
	ServiceAccountID string `json:"service_account_id"`
	CreatedAt        string `json:"created_at"`
	KeyAlgorithm     string `json:"key_algorithm"`
	PublicKey        string `json:"public_key"`
	PrivateKey       string `json:"private_key"`
}

// IAMTokenProvider automatically obtains and refreshes IAM tokens
// using a service account authorized key (JWT-based auth).
type IAMTokenProvider struct {
	saKey     *ServiceAccountKey
	token     string
	expiresAt time.Time
	mu        sync.Mutex
}

// NewIAMTokenProvider creates a new IAM token provider from a service account key.
func NewIAMTokenProvider(saKey *ServiceAccountKey) *IAMTokenProvider {
	return &IAMTokenProvider{saKey: saKey}
}

// NewIAMTokenProviderFromJSON creates a provider from JSON key data.
func NewIAMTokenProviderFromJSON(keyJSON []byte) (*IAMTokenProvider, error) {
	var saKey ServiceAccountKey
	if err := json.Unmarshal(keyJSON, &saKey); err != nil {
		return nil, fmt.Errorf("failed to parse service account key: %w", err)
	}
	if saKey.ID == "" || saKey.ServiceAccountID == "" || saKey.PrivateKey == "" {
		return nil, fmt.Errorf("invalid service account key: missing required fields")
	}
	return NewIAMTokenProvider(&saKey), nil
}

// Token returns a valid IAM token, refreshing if necessary.
func (p *IAMTokenProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Return cached token if still valid (with 5 min buffer)
	if p.token != "" && time.Now().Add(5*time.Minute).Before(p.expiresAt) {
		return p.token, nil
	}

	// Get new token
	token, expiresAt, err := p.fetchIAMToken(ctx)
	if err != nil {
		return "", err
	}

	p.token = token
	p.expiresAt = expiresAt
	return token, nil
}

// fetchIAMToken obtains a new IAM token via JWT assertion.
func (p *IAMTokenProvider) fetchIAMToken(ctx context.Context) (string, time.Time, error) {
	// Parse private key
	privateKey, err := parseRSAPrivateKey(p.saKey.PrivateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse private key: %w", err)
	}

	now := time.Now()
	// Create JWT assertion
	claims := jwt.RegisteredClaims{
		Issuer:    p.saKey.ServiceAccountID,
		Subject:   p.saKey.ServiceAccountID,
		Audience:  jwt.ClaimStrings{"https://iam.api.cloud.yandex.net/iam/v1/tokens"},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
	}

	// Yandex Cloud requires PS256 (RSA-PSS with SHA-256)
	token := jwt.NewWithClaims(jwt.SigningMethodPS256, claims)
	token.Header["kid"] = p.saKey.ID

	signedJWT, err := token.SignedString(privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign JWT: %w", err)
	}

	// Exchange JWT for IAM token
	return exchangeJWTForIAMToken(ctx, signedJWT)
}

// exchangeJWTForIAMToken exchanges a signed JWT for a Yandex Cloud IAM token.
func exchangeJWTForIAMToken(ctx context.Context, signedJWT string) (string, time.Time, error) {
	body, _ := json.Marshal(map[string]string{"jwt": signedJWT})

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://iam.api.cloud.yandex.net/iam/v1/tokens",
		bytes.NewReader(body))
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("IAM token request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", time.Time{}, fmt.Errorf("IAM token request failed (status %d): %s",
			resp.StatusCode, string(respBody))
	}

	var result struct {
		IAMToken  string `json:"iamToken"`
		ExpiresAt string `json:"expiresAt"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse IAM token response: %w", err)
	}

	expiresAt, _ := time.Parse(time.RFC3339, result.ExpiresAt)
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(11 * time.Hour) // default 11h
	}

	return result.IAMToken, expiresAt, nil
}

// parseRSAPrivateKey parses a PEM-encoded RSA private key.
func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Try PKCS8 first
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA")
		}
		return rsaKey, nil
	}

	// Try PKCS1
	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
	}
	return rsaKey, nil
}

// iamTokenCredentials implements ydb credentials using IAMTokenProvider.
type iamTokenCredentials struct {
	provider *IAMTokenProvider
}

// Token implements the ydb credentials interface.
func (c *iamTokenCredentials) Token(ctx context.Context) (string, error) {
	return c.provider.Token(ctx)
}
