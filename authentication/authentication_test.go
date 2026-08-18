package authentication

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestKeycloakJWTAuthenticator(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	kid := "test-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{{
				"kid": kid,
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
			}},
		})
	}))
	defer server.Close()

	now := time.Unix(1_800_000_000, 0)
	issuer := "https://keycloak.example/realms/main"
	authenticator := NewKeycloakJWTAuthenticator(KeycloakJWTConfig{
		Issuer:          issuer,
		JWKSURL:         server.URL,
		Audience:        "page-api",
		AuthorizedParty: "frontend",
		Now:             func() time.Time { return now },
	})
	token := signedJWT(t, privateKey, kid, map[string]any{
		"iss":                issuer,
		"sub":                "user-123",
		"aud":                []string{"account", "page-api"},
		"azp":                "frontend",
		"exp":                now.Add(time.Minute).Unix(),
		"iat":                now.Add(-time.Minute).Unix(),
		"preferred_username": "behzod",
	})
	principal, err := authenticator.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if principal.ID != issuer+"|user-123" {
		t.Fatalf("unexpected principal ID %q", principal.ID)
	}
	if principal.User["preferred_username"] != "behzod" {
		t.Fatalf("claims were not preserved: %#v", principal.User)
	}
}

func TestKeycloakJWTAuthenticatorRejectsExpiredToken(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	server := jwksServer(t, &privateKey.PublicKey, "key")
	defer server.Close()
	authenticator := NewKeycloakJWTAuthenticator(KeycloakJWTConfig{
		Issuer:  "issuer",
		JWKSURL: server.URL,
		Now:     func() time.Time { return now },
	})
	token := signedJWT(t, privateKey, "key", map[string]any{
		"iss": "issuer",
		"sub": "user",
		"exp": now.Add(-time.Second).Unix(),
	})
	if _, err := authenticator.Authenticate(context.Background(), token); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func jwksServer(t *testing.T, key *rsa.PublicKey, kid string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{{
				"kid": kid,
				"kty": "RSA",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
			}},
		})
	}))
}

func signedJWT(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	headerBytes, _ := json.Marshal(map[string]any{"alg": "RS256", "typ": "JWT", "kid": kid})
	claimsBytes, _ := json.Marshal(claims)
	header := base64.RawURLEncoding.EncodeToString(headerBytes)
	payload := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signingInput := header + "." + payload
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}
