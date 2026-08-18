package access

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const umaTicketGrantType = "urn:ietf:params:oauth:grant-type:uma-ticket"
const defaultAccessScope = "access"

type KeycloakUMAAccessAuthorizer struct {
	Config Config
	now    func() time.Time
	mu     sync.Mutex
	cache  map[string]cachedDecision
}

type cachedDecision struct {
	expiresAt time.Time
	allowed   bool
}

func NewKeycloakUMAAccessAuthorizer(config Config) *KeycloakUMAAccessAuthorizer {
	if config.CacheTTL <= 0 {
		config.CacheTTL = 30 * time.Second
	}
	return &KeycloakUMAAccessAuthorizer{
		Config: config,
		now:    time.Now,
		cache:  map[string]cachedDecision{},
	}
}

func (a *KeycloakUMAAccessAuthorizer) UserAccessGroups(ctx context.Context, userID string, user map[string]any) ([]string, error) {
	return RPTClaimSource{}.UserAccessGroups(ctx, userID, user)
}

func (a *KeycloakUMAAccessAuthorizer) HasAccess(ctx context.Context, userID string, user map[string]any, accessGroupCode string) (bool, error) {
	if accessGroupCode == "" {
		return true, nil
	}
	if hasClaimAccess(user, accessGroupCode) {
		return true, nil
	}
	token := bearerTokenFromContext(ctx)
	if token == "" {
		return false, nil
	}
	return a.decision(ctx, userID, token, accessGroupCode)
}

func (a *KeycloakUMAAccessAuthorizer) Invalidate() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cache = map[string]cachedDecision{}
}

func (a *KeycloakUMAAccessAuthorizer) decision(ctx context.Context, userID, token, resource string) (bool, error) {
	cacheKey := userID + "\x00" + tokenFingerprint(token) + "\x00" + resource
	now := a.now()
	a.mu.Lock()
	if cached, ok := a.cache[cacheKey]; ok && now.Before(cached.expiresAt) {
		a.mu.Unlock()
		return cached.allowed, nil
	}
	a.mu.Unlock()

	allowed, err := a.requestDecision(ctx, token, resource)
	if err != nil {
		return false, err
	}

	a.mu.Lock()
	a.cache[cacheKey] = cachedDecision{
		expiresAt: now.Add(a.Config.CacheTTL),
		allowed:   allowed,
	}
	a.mu.Unlock()
	return allowed, nil
}

func (a *KeycloakUMAAccessAuthorizer) requestDecision(ctx context.Context, token, resource string) (bool, error) {
	if a.Config.KeycloakURL == "" || a.Config.Realm == "" || a.Config.ClientID == "" {
		return false, fmt.Errorf("Keycloak UMA access authorizer is not configured")
	}
	values := url.Values{}
	values.Set("grant_type", umaTicketGrantType)
	values.Set("audience", a.Config.ClientID)
	values.Set("permission", resource+"#"+defaultAccessScope)
	values.Set("response_mode", "decision")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, keycloakTokenURL(a.Config), strings.NewReader(values.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := a.Config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusForbidden {
		return false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if keycloakDecisionDenied(body) {
			return false, nil
		}
		return false, fmt.Errorf("Keycloak UMA decision for %q failed: %s: %s", resource, resp.Status, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Result bool `json:"result"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, fmt.Errorf("decode Keycloak UMA decision for %q: %w", resource, err)
	}
	return payload.Result, nil
}

func hasClaimAccess(user map[string]any, accessGroupCode string) bool {
	groups, err := RPTClaimSource{}.UserAccessGroups(context.Background(), "", user)
	if err != nil {
		return false
	}
	for _, group := range groups {
		if group == accessGroupCode {
			return true
		}
	}
	return false
}

func keycloakDecisionDenied(body []byte) bool {
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	return payload.Error == "access_denied" || payload.Error == "not_authorized"
}

func keycloakTokenURL(config Config) string {
	return strings.TrimRight(config.KeycloakURL, "/") + "/realms/" + url.PathEscape(config.Realm) + "/protocol/openid-connect/token"
}

func tokenFingerprint(token string) string {
	if len(token) <= 16 {
		return token
	}
	return token[:8] + token[len(token)-8:]
}
