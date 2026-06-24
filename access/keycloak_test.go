package access

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKeycloakUMAProviderSyncsOnlyAccessGroups(t *testing.T) {
	var createdPayload map[string]any
	var createdRole map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/realms/sfp/protocol/openid-connect/token":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("client_id") != "gateway" {
				t.Fatalf("unexpected token form: %v", r.Form)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "service-token"})
		case r.URL.Path == "/realms/sfp/authz/protection/resource_set" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode([]keycloakResource{
				{
					ID:   "existing-id",
					Name: "client.card.viewing",
					Attributes: map[string][]string{
						"sfp.accessGroup": []string{"true"},
					},
				},
			})
		case r.URL.Path == "/realms/sfp/authz/protection/resource_set" && r.Method == http.MethodPost:
			if r.Header.Get("Authorization") != "Bearer service-token" {
				t.Fatalf("missing bearer token: %s", r.Header.Get("Authorization"))
			}
			if err := json.NewDecoder(r.Body).Decode(&createdPayload); err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"_id": "created-id"})
		case r.URL.Path == "/realms/sfp/authz/protection/resource_set/existing-id" && r.Method == http.MethodPut:
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/admin/realms/sfp/roles/client_operator" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
		case r.URL.Path == "/admin/realms/sfp/roles" && r.Method == http.MethodPost:
			if err := json.NewDecoder(r.Body).Decode(&createdRole); err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(http.StatusCreated)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	provider := NewKeycloakUMAProvider(Config{
		KeycloakURL:  server.URL,
		Realm:        "sfp",
		ClientID:     "gateway",
		ClientSecret: "secret",
	})
	manifest := Manifest{
		AccessGroups: []AccessGroup{
			{
				Code:    "client.card.viewing",
				Name:    "View",
				Type:    AccessGroupUI,
				Enabled: true,
				Elements: []AccessElement{{
					Code:             "client.name.input",
					ElementType:      ElementInput,
					NoAccessBehavior: NoAccessReadonly,
				}},
			},
			{Code: "client.card.editing", Name: "Edit", Type: AccessGroupUI, Enabled: true},
		},
		PermissionGroups: []PermissionGroup{
			{Code: "client_operator", AccessGroups: []string{"client.card.editing"}},
		},
	}
	if err := provider.Sync(context.Background(), manifest, SyncOptions{}); err != nil {
		t.Fatal(err)
	}
	if createdPayload["name"] != "client.card.editing" {
		t.Fatalf("unexpected created resource: %#v", createdPayload)
	}
	body, _ := json.Marshal(createdPayload)
	if strings.Contains(string(body), "client.name.input") {
		t.Fatalf("UI element leaked into Keycloak payload: %s", body)
	}
	if createdRole["name"] != "client_operator" {
		t.Fatalf("permission group role was not created: %#v", createdRole)
	}
}
