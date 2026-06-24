// Package authentication provides request authentication for pageSDK.
package authentication

import (
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/BekkkEvrika/pageSDK/engine"
)

var ErrUnauthenticated = errors.New("authentication failed")

// Principal is a verified request identity.
type Principal struct {
	// ID is the stable owner identity used for page instances.
	// The built-in JWT authenticator uses "{issuer}|{subject}".
	ID   string
	User engine.User
}

// Authenticator verifies a bearer token and returns trusted user claims.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (Principal, error)
}

// AuthenticatorFunc adapts a function to Authenticator.
type AuthenticatorFunc func(ctx context.Context, token string) (Principal, error)

func (f AuthenticatorFunc) Authenticate(ctx context.Context, token string) (Principal, error) {
	return f(ctx, token)
}

// KeycloakJWTConfig configures RS256 access-token verification through JWKS.
type KeycloakJWTConfig struct {
	KeycloakURL string
	Realm       string

	// Issuer defaults to {KeycloakURL}/realms/{Realm}.
	Issuer string
	// JWKSURL defaults to {Issuer}/protocol/openid-connect/certs.
	JWKSURL string

	// Audience, when set, must be present in aud.
	Audience string
	// AuthorizedParty, when set, must match azp. This is useful for Keycloak
	// access tokens whose aud does not contain the calling client.
	AuthorizedParty string

	ClockSkew  time.Duration
	CacheTTL   time.Duration
	HTTPClient *http.Client
	Now        func() time.Time
}

type KeycloakJWTAuthenticator struct {
	config KeycloakJWTConfig
	mu     sync.RWMutex
	keys   map[string]*rsa.PublicKey
	expiry time.Time
}

func NewKeycloakJWTAuthenticator(config KeycloakJWTConfig) *KeycloakJWTAuthenticator {
	config.KeycloakURL = strings.TrimRight(config.KeycloakURL, "/")
	if config.Issuer == "" && config.KeycloakURL != "" && config.Realm != "" {
		config.Issuer = config.KeycloakURL + "/realms/" + config.Realm
	}
	config.Issuer = strings.TrimRight(config.Issuer, "/")
	if config.JWKSURL == "" && config.Issuer != "" {
		config.JWKSURL = config.Issuer + "/protocol/openid-connect/certs"
	}
	if config.CacheTTL <= 0 {
		config.CacheTTL = 15 * time.Minute
	}
	if config.ClockSkew < 0 {
		config.ClockSkew = 0
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &KeycloakJWTAuthenticator{config: config}
}

func (a *KeycloakJWTAuthenticator) Authenticate(ctx context.Context, token string) (Principal, error) {
	header, claims, signingInput, signature, err := parseJWT(token)
	if err != nil {
		return Principal{}, authError(err)
	}
	if header.Alg != "RS256" || header.Kid == "" {
		return Principal{}, authError(errors.New("unsupported JWT header"))
	}
	key, err := a.publicKey(ctx, header.Kid)
	if err != nil {
		return Principal{}, authError(err)
	}
	hash := crypto.SHA256.New()
	_, _ = hash.Write([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hash.Sum(nil), signature); err != nil {
		return Principal{}, authError(errors.New("invalid JWT signature"))
	}
	if err := a.validateClaims(claims); err != nil {
		return Principal{}, authError(err)
	}
	user := make(engine.User, len(claims))
	for key, value := range claims {
		user[key] = value
	}
	return Principal{
		ID:   claims.string("iss") + "|" + claims.string("sub"),
		User: user,
	}, nil
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type jwtClaims map[string]any

func parseJWT(token string) (jwtHeader, jwtClaims, string, []byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtHeader{}, nil, "", nil, errors.New("malformed JWT")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return jwtHeader{}, nil, "", nil, errors.New("malformed JWT header")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtHeader{}, nil, "", nil, errors.New("malformed JWT claims")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return jwtHeader{}, nil, "", nil, errors.New("malformed JWT signature")
	}
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return jwtHeader{}, nil, "", nil, errors.New("malformed JWT header")
	}
	var claims jwtClaims
	decoder := json.NewDecoder(strings.NewReader(string(payloadBytes)))
	decoder.UseNumber()
	if err := decoder.Decode(&claims); err != nil {
		return jwtHeader{}, nil, "", nil, errors.New("malformed JWT claims")
	}
	return header, claims, parts[0] + "." + parts[1], signature, nil
}

func (a *KeycloakJWTAuthenticator) validateClaims(claims jwtClaims) error {
	now := a.config.Now()
	skew := a.config.ClockSkew
	issuer := claims.string("iss")
	subject := claims.string("sub")
	if issuer == "" || subject == "" {
		return errors.New("JWT must contain iss and sub")
	}
	if a.config.Issuer == "" || issuer != a.config.Issuer {
		return errors.New("invalid JWT issuer")
	}
	expiresAt, ok := claims.numericDate("exp")
	if !ok || !now.Before(expiresAt.Add(skew)) {
		return errors.New("JWT is expired")
	}
	if notBefore, ok := claims.numericDate("nbf"); ok && now.Add(skew).Before(notBefore) {
		return errors.New("JWT is not active")
	}
	if issuedAt, ok := claims.numericDate("iat"); ok && now.Add(skew).Before(issuedAt) {
		return errors.New("JWT issued-at time is in the future")
	}
	if a.config.Audience != "" && !claims.hasAudience(a.config.Audience) {
		return errors.New("invalid JWT audience")
	}
	if a.config.AuthorizedParty != "" && claims.string("azp") != a.config.AuthorizedParty {
		return errors.New("invalid JWT authorized party")
	}
	return nil
}

func (c jwtClaims) string(key string) string {
	value, _ := c[key].(string)
	return value
}

func (c jwtClaims) numericDate(key string) (time.Time, bool) {
	switch value := c[key].(type) {
	case json.Number:
		seconds, err := value.Int64()
		if err != nil {
			return time.Time{}, false
		}
		return time.Unix(seconds, 0), true
	case float64:
		return time.Unix(int64(value), 0), true
	default:
		return time.Time{}, false
	}
}

func (c jwtClaims) hasAudience(expected string) bool {
	switch audience := c["aud"].(type) {
	case string:
		return audience == expected
	case []any:
		for _, item := range audience {
			if value, ok := item.(string); ok && value == expected {
				return true
			}
		}
	}
	return false
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (a *KeycloakJWTAuthenticator) publicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	now := a.config.Now()
	a.mu.RLock()
	key := a.keys[kid]
	fresh := now.Before(a.expiry)
	a.mu.RUnlock()
	if key != nil && fresh {
		return key, nil
	}
	if err := a.refreshKeys(ctx); err != nil {
		return nil, err
	}
	a.mu.RLock()
	key = a.keys[kid]
	a.mu.RUnlock()
	if key == nil {
		return nil, errors.New("JWT signing key not found")
	}
	return key, nil
}

func (a *KeycloakJWTAuthenticator) refreshKeys(ctx context.Context) error {
	if a.config.JWKSURL == "" {
		return errors.New("JWKS URL is not configured")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, a.config.JWKSURL, nil)
	if err != nil {
		return err
	}
	response, err := a.config.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("load JWKS: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("load JWKS: unexpected status %d", response.StatusCode)
	}
	var document jwksDocument
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, item := range document.Keys {
		if item.Kid == "" || item.Kty != "RSA" || (item.Use != "" && item.Use != "sig") ||
			(item.Alg != "" && item.Alg != "RS256") {
			continue
		}
		key, err := rsaPublicKey(item.N, item.E)
		if err == nil {
			keys[item.Kid] = key
		}
	}
	if len(keys) == 0 {
		return errors.New("JWKS contains no usable RS256 keys")
	}
	a.mu.Lock()
	a.keys = keys
	a.expiry = a.config.Now().Add(a.config.CacheTTL)
	a.mu.Unlock()
	return nil
}

func rsaPublicKey(modulus, exponent string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil || len(nBytes) == 0 {
		return nil, errors.New("invalid RSA modulus")
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(exponent)
	if err != nil || len(eBytes) == 0 || len(eBytes) > 4 {
		return nil, errors.New("invalid RSA exponent")
	}
	e := 0
	for _, value := range eBytes {
		e = e<<8 | int(value)
	}
	if e < 3 {
		return nil, errors.New("invalid RSA exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}

func authError(err error) error {
	return fmt.Errorf("%w: %v", ErrUnauthenticated, err)
}
