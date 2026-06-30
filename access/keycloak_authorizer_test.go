package access

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestKeycloakUMAAccessAuthorizerFallsBackToDecisionEndpoint(t *testing.T) {
	var gotAuthorization string
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/realms/sfp/protocol/openid-connect/token" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		gotAuthorization = r.Header.Get("Authorization")
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		gotForm = r.Form
		_ = json.NewEncoder(w).Encode(map[string]bool{"result": true})
	}))
	defer server.Close()

	authorizer := NewKeycloakUMAAccessAuthorizer(Config{
		KeycloakURL: server.URL,
		Realm:       "sfp",
		ClientID:    "gateway",
		CacheTTL:    time.Minute,
	})
	ctx := WithBearerToken(context.Background(), "incoming-rpt")
	allowed, err := authorizer.HasAccess(ctx, "user-1", map[string]any{
		"authorization": map[string]any{
			"permissions": []any{
				map[string]any{"rsname": "page.calculator"},
			},
		},
	}, "calculator.usage")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("expected decision endpoint to allow access")
	}
	if gotAuthorization != "Bearer incoming-rpt" {
		t.Fatalf("unexpected authorization header %q", gotAuthorization)
	}
	if gotForm.Get("grant_type") != umaTicketGrantType {
		t.Fatalf("unexpected grant_type %#v", gotForm)
	}
	if gotForm.Get("audience") != "gateway" {
		t.Fatalf("unexpected audience %#v", gotForm)
	}
	if gotForm.Get("permission") != "calculator.usage#access" {
		t.Fatalf("unexpected permission %#v", gotForm)
	}
	if gotForm.Get("response_mode") != "decision" {
		t.Fatalf("unexpected response_mode %#v", gotForm)
	}
}

func TestKeycloakUMAAccessAuthorizerUsesRPTClaimsBeforeDecisionEndpoint(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	authorizer := NewKeycloakUMAAccessAuthorizer(Config{
		KeycloakURL: server.URL,
		Realm:       "sfp",
		ClientID:    "gateway",
	})
	allowed, err := authorizer.HasAccess(context.Background(), "user-1", map[string]any{
		"authorization": map[string]any{
			"permissions": []any{
				map[string]any{"rsname": "calculator.usage"},
			},
		},
	}, "calculator.usage")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("expected RPT claim to allow access")
	}
	if called {
		t.Fatal("decision endpoint should not be called when RPT already contains resource")
	}
}

func TestKeycloakUMAAccessAuthorizerDeniedDecision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"access_denied"}`))
	}))
	defer server.Close()

	authorizer := NewKeycloakUMAAccessAuthorizer(Config{
		KeycloakURL: server.URL,
		Realm:       "sfp",
		ClientID:    "gateway",
	})
	ctx := WithBearerToken(context.Background(), "incoming-rpt")
	allowed, err := authorizer.HasAccess(ctx, "user-1", nil, "calculator.usage")
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Fatal("expected denied decision")
	}
}

func TestKeycloakUMAAccessAuthorizerReportsDecisionErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	defer server.Close()

	authorizer := NewKeycloakUMAAccessAuthorizer(Config{
		KeycloakURL: server.URL,
		Realm:       "sfp",
		ClientID:    "gateway",
	})
	ctx := WithBearerToken(context.Background(), "incoming-rpt")
	_, err := authorizer.HasAccess(ctx, "user-1", nil, "calculator.usage")
	if err == nil || !strings.Contains(err.Error(), "bad gateway") {
		t.Fatalf("expected decision error, got %v", err)
	}
}
