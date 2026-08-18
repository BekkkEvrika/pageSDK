package access

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type KeycloakUMAProvider struct {
	Config Config
}

func NewKeycloakUMAProvider(config Config) *KeycloakUMAProvider {
	return &KeycloakUMAProvider{Config: config}
}

func (p *KeycloakUMAProvider) Sync(ctx context.Context, manifest Manifest, opts SyncOptions) error {
	if opts.DryRun {
		return nil
	}
	token, err := p.serviceToken(ctx)
	if err != nil {
		return err
	}
	existing, err := p.resourceIndex(ctx, token)
	if err != nil {
		return err
	}
	for _, group := range manifest.AccessGroups {
		if !group.Enabled {
			continue
		}
		resource := keycloakResourceFromAccessGroup(group)
		if id := existing[group.Code]; id != "" {
			if err := p.putJSON(ctx, token, p.resourceSetURL()+"/"+url.PathEscape(id), resource); err != nil {
				return fmt.Errorf("update Keycloak access group %q: %w", group.Code, err)
			}
			continue
		}
		if err := p.postJSON(ctx, token, p.resourceSetURL(), resource, nil); err != nil {
			return fmt.Errorf("create Keycloak access group %q: %w", group.Code, err)
		}
	}
	for _, stale := range staleAccessGroups(manifest, existing) {
		if err := p.markDeprecated(ctx, token, stale.code, stale.id); err != nil {
			return err
		}
	}
	return nil
}

func (p *KeycloakUMAProvider) Diff(ctx context.Context, manifest Manifest) (*Diff, error) {
	token, err := p.serviceToken(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := p.resourceIndex(ctx, token)
	if err != nil {
		return nil, err
	}
	local := map[string]struct{}{}
	diff := &Diff{}
	for _, group := range manifest.AccessGroups {
		local[group.Code] = struct{}{}
		if existing[group.Code] == "" {
			diff.MissingInManifest = append(diff.MissingInManifest, group.Code)
		}
	}
	for code := range existing {
		if _, ok := local[code]; !ok {
			diff.MissingInDSL = append(diff.MissingInDSL, code)
		}
	}
	sortStringsInDiff(diff)
	return diff, nil
}

func (p *KeycloakUMAProvider) serviceToken(ctx context.Context) (string, error) {
	values := url.Values{}
	values.Set("grant_type", "client_credentials")
	values.Set("client_id", p.Config.ClientID)
	values.Set("client_secret", p.Config.ClientSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL(), strings.NewReader(values.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var response struct {
		AccessToken string `json:"access_token"`
	}
	if err := p.do(req, &response); err != nil {
		return "", fmt.Errorf("get Keycloak service account token: %w", err)
	}
	if response.AccessToken == "" {
		return "", fmt.Errorf("get Keycloak service account token: empty access_token")
	}
	return response.AccessToken, nil
}

func (p *KeycloakUMAProvider) resourceIndex(ctx context.Context, token string) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.resourceSetURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	var raw json.RawMessage
	if err := p.do(req, &raw); err != nil {
		return nil, fmt.Errorf("list Keycloak UMA resources: %w", err)
	}
	result := map[string]string{}
	var objects []keycloakResource
	if err := json.Unmarshal(raw, &objects); err == nil {
		for _, resource := range objects {
			if resource.Name != "" && resource.ID != "" && resource.isSFPAccessGroup() {
				result[resource.Name] = resource.ID
			}
		}
		return result, nil
	}
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, fmt.Errorf("decode Keycloak UMA resources: %w", err)
	}
	for _, id := range ids {
		resource, err := p.getResource(ctx, token, id)
		if err != nil {
			return nil, err
		}
		if resource.Name != "" && resource.isSFPAccessGroup() {
			result[resource.Name] = resource.ID
		}
	}
	return result, nil
}

func (p *KeycloakUMAProvider) getResource(ctx context.Context, token, id string) (keycloakResource, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.resourceSetURL()+"/"+url.PathEscape(id), nil)
	if err != nil {
		return keycloakResource{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	var resource keycloakResource
	if err := p.do(req, &resource); err != nil {
		return keycloakResource{}, fmt.Errorf("get Keycloak UMA resource %q: %w", id, err)
	}
	return resource, nil
}

func (p *KeycloakUMAProvider) markDeprecated(ctx context.Context, token, code, id string) error {
	resource, err := p.getResource(ctx, token, id)
	if err != nil {
		return err
	}
	if resource.Attributes == nil {
		resource.Attributes = map[string][]string{}
	}
	resource.Attributes["sfp.deprecated"] = []string{"true"}
	resource.Attributes["sfp.deprecatedAt"] = []string{time.Now().UTC().Format(time.RFC3339)}
	if err := p.putJSON(ctx, token, p.resourceSetURL()+"/"+url.PathEscape(id), resource); err != nil {
		return fmt.Errorf("deprecate Keycloak access group %q: %w", code, err)
	}
	return nil
}

func (p *KeycloakUMAProvider) postJSON(ctx context.Context, token, endpoint string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return p.do(req, out)
}

func (p *KeycloakUMAProvider) putJSON(ctx context.Context, token, endpoint string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return p.do(req, nil)
}

func (p *KeycloakUMAProvider) do(req *http.Request, out any) error {
	client := p.Config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func (p *KeycloakUMAProvider) tokenURL() string {
	return strings.TrimRight(p.Config.KeycloakURL, "/") + "/realms/" + url.PathEscape(p.Config.Realm) + "/protocol/openid-connect/token"
}

func (p *KeycloakUMAProvider) resourceSetURL() string {
	return strings.TrimRight(p.Config.KeycloakURL, "/") + "/realms/" + url.PathEscape(p.Config.Realm) + "/authz/protection/resource_set"
}

type keycloakResource struct {
	ID          string              `json:"_id,omitempty"`
	Name        string              `json:"name"`
	DisplayName string              `json:"displayName,omitempty"`
	Type        string              `json:"type,omitempty"`
	Scopes      []keycloakScope     `json:"scopes,omitempty"`
	Attributes  map[string][]string `json:"attributes,omitempty"`
}

func (r keycloakResource) isSFPAccessGroup() bool {
	for _, value := range r.Attributes["sfp.accessGroup"] {
		if value == "true" {
			return true
		}
	}
	return false
}

type keycloakScope struct {
	Name string `json:"name"`
}

func keycloakResourceFromAccessGroup(group AccessGroup) keycloakResource {
	attributes := map[string][]string{
		"sfp.accessGroup": []string{"true"},
		"sfp.enabled":     []string{fmt.Sprintf("%t", group.Enabled)},
	}
	if group.ParentCode != "" {
		attributes["sfp.parentCode"] = []string{group.ParentCode}
	}
	return keycloakResource{
		Name:        group.Code,
		DisplayName: group.Name,
		Type:        string(group.Type),
		Scopes:      []keycloakScope{{Name: "access"}},
		Attributes:  attributes,
	}
}

type staleKeycloakResource struct {
	code string
	id   string
}

func staleAccessGroups(manifest Manifest, existing map[string]string) []staleKeycloakResource {
	local := map[string]struct{}{}
	for _, group := range manifest.AccessGroups {
		local[group.Code] = struct{}{}
	}
	stale := []staleKeycloakResource{}
	for code, id := range existing {
		if _, ok := local[code]; !ok {
			stale = append(stale, staleKeycloakResource{code: code, id: id})
		}
	}
	return stale
}
