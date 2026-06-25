package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BekkkEvrika/pageSDK/access"
	"github.com/BekkkEvrika/pageSDK/authentication"
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/formengine"
	"github.com/BekkkEvrika/pageSDK/engine/tableengine"
)

type cliPage struct {
	*formengine.FormEngine
}

type accessAnnotatedPage struct {
	*formengine.FormEngine
	group access.AccessGroup
}

type accessAnnotatedTablePage struct {
	*tableengine.TableEngine
	group access.AccessGroup
}

func TestPageAccessDeniedBeforeInit(t *testing.T) {
	initCalls := 0
	authenticator := authentication.AuthenticatorFunc(func(_ context.Context, token string) (authentication.Principal, error) {
		return authentication.Principal{
			ID:   token,
			User: engine.User{"sub": token},
		}, nil
	})
	a := New(Config{
		Authenticator: authenticator,
		AccessAuthorizer: access.StaticAuthorizer{
			Groups: map[string][]string{"allowed-user": {"page.secure.page"}},
		},
	})
	a.Manifest().Register("secure.page", func() engine.Page {
		return &instanceLifecyclePage{
			FormEngine: &formengine.FormEngine{},
			initCalls:  &initCalls,
		}
	})
	a.registerRoutes()
	beforeDenied := initCalls

	deniedRequest := httptest.NewRequest(http.MethodGet, "/page/secure.page", nil)
	deniedRequest.Header.Set("Authorization", "Bearer denied-user")
	deniedResponse := httptest.NewRecorder()
	a.router.ServeHTTP(deniedResponse, deniedRequest)
	if deniedResponse.Code != http.StatusForbidden {
		t.Fatalf("denied render returned %d: %s", deniedResponse.Code, deniedResponse.Body.String())
	}
	if initCalls != beforeDenied {
		t.Fatalf("Init was called for denied user")
	}

	allowedRequest := httptest.NewRequest(http.MethodGet, "/page/secure.page", nil)
	allowedRequest.Header.Set("Authorization", "Bearer allowed-user")
	allowedResponse := httptest.NewRecorder()
	a.router.ServeHTTP(allowedResponse, allowedRequest)
	if allowedResponse.Code != http.StatusOK {
		t.Fatalf("allowed render returned %d: %s", allowedResponse.Code, allowedResponse.Body.String())
	}
}

func TestRPTPermissionsAuthorizePage(t *testing.T) {
	authenticator := authentication.AuthenticatorFunc(func(_ context.Context, token string) (authentication.Principal, error) {
		user := engine.User{
			"sub": token,
			"authorization": map[string]any{
				"permissions": []any{
					map[string]any{"resource_set_name": "page.secure.page"},
				},
			},
		}
		if token == "denied-user" {
			user["authorization"] = map[string]any{"permissions": []any{}}
		}
		return authentication.Principal{ID: token, User: user}, nil
	})
	a := New(Config{Authenticator: authenticator})
	a.UseRPTAccessAuthorizer()
	a.Manifest().Register("secure.page", func() engine.Page {
		return &authenticatedPage{FormEngine: &formengine.FormEngine{}}
	})
	a.registerRoutes()

	deniedRequest := httptest.NewRequest(http.MethodGet, "/page/secure.page", nil)
	deniedRequest.Header.Set("Authorization", "Bearer denied-user")
	deniedResponse := httptest.NewRecorder()
	a.router.ServeHTTP(deniedResponse, deniedRequest)
	if deniedResponse.Code != http.StatusForbidden {
		t.Fatalf("denied RPT render returned %d: %s", deniedResponse.Code, deniedResponse.Body.String())
	}

	allowedRequest := httptest.NewRequest(http.MethodGet, "/page/secure.page", nil)
	allowedRequest.Header.Set("Authorization", "Bearer allowed-user")
	allowedResponse := httptest.NewRecorder()
	a.router.ServeHTTP(allowedResponse, allowedRequest)
	if allowedResponse.Code != http.StatusOK {
		t.Fatalf("allowed RPT render returned %d: %s", allowedResponse.Code, allowedResponse.Body.String())
	}
}

type authenticatedPage struct {
	*formengine.FormEngine
}

func (p *authenticatedPage) Init(ctx *engine.BuildContext) error {
	p.Text("subject").DefaultValue(ctx.User["sub"])
	p.Button("save").OnClick(func(*formengine.RuntimeContext) {})
	return nil
}

func TestAuthenticationClaimsAndPageInstanceOwnership(t *testing.T) {
	authenticator := authentication.AuthenticatorFunc(func(_ context.Context, token string) (authentication.Principal, error) {
		return authentication.Principal{
			ID:   "issuer|" + token,
			User: engine.User{"iss": "issuer", "sub": token},
		}, nil
	})
	a := New(Config{Authenticator: authenticator})
	a.Manifest().Register("secure.page", func() engine.Page {
		return &authenticatedPage{FormEngine: &formengine.FormEngine{}}
	})
	a.registerRoutes()

	unauthenticated := httptest.NewRecorder()
	a.router.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/page/secure.page", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("request without JWT returned %d", unauthenticated.Code)
	}

	renderRequest := httptest.NewRequest(http.MethodGet, "/page/secure.page", nil)
	renderRequest.Header.Set("Authorization", "Bearer user-1")
	renderResponse := httptest.NewRecorder()
	a.router.ServeHTTP(renderResponse, renderRequest)
	if renderResponse.Code != http.StatusOK {
		t.Fatalf("authenticated render returned %d: %s", renderResponse.Code, renderResponse.Body.String())
	}
	if !strings.Contains(renderResponse.Body.String(), `"defaultValue":"user-1"`) {
		t.Fatalf("JWT claims did not reach Init: %s", renderResponse.Body.String())
	}
	rendered := decodeRenderedInstance(t, renderResponse)

	foreignEvent := httptest.NewRequest(http.MethodPost, rendered.EventURL, strings.NewReader(`{}`))
	foreignEvent.Header.Set("Authorization", "Bearer user-2")
	foreignResponse := httptest.NewRecorder()
	a.router.ServeHTTP(foreignResponse, foreignEvent)
	if foreignResponse.Code != http.StatusNotFound {
		t.Fatalf("foreign owner event returned %d: %s", foreignResponse.Code, foreignResponse.Body.String())
	}

	ownerEvent := httptest.NewRequest(http.MethodPost, rendered.EventURL, strings.NewReader(`{}`))
	ownerEvent.Header.Set("Authorization", "Bearer user-1")
	ownerResponse := httptest.NewRecorder()
	a.router.ServeHTTP(ownerResponse, ownerEvent)
	if ownerResponse.Code != http.StatusOK {
		t.Fatalf("owner event returned %d: %s", ownerResponse.Code, ownerResponse.Body.String())
	}
}

func (p *cliPage) Init(_ *engine.BuildContext) error {
	p.Button("save").OnClick(func(*formengine.RuntimeContext) {})
	return nil
}

func (p *accessAnnotatedPage) Init(_ *engine.BuildContext) error {
	p.Text("client.name").Access(p.group, access.NoAccessReadonly)
	p.Button("save").Access(p.group, access.NoAccessHidden)
	return nil
}

func (p *accessAnnotatedTablePage) Init(_ *engine.BuildContext) error {
	p.Table("clients").
		Access(p.group, access.NoAccessHidden).
		Columns(
			p.Column("name").Header("Name").Access(p.group, access.NoAccessHidden),
			p.Column("status").Header("Status").AddActionBuilder(
				p.Action("approve", func(*tableengine.TableRuntimeContext) {}).Access(p.group, access.NoAccessHidden),
			),
		).
		ToolbarActions(
			p.Action("export", func(*tableengine.TableRuntimeContext) {}).Access(p.group, access.NoAccessHidden),
		).
		RowActions(
			p.Action("delete", func(*tableengine.TableRuntimeContext) {}).Access(p.group, access.NoAccessRemove),
		).
		SelectedActions(
			p.Action("archive", func(*tableengine.TableRuntimeContext) {}).Access(p.group, access.NoAccessHidden),
		)
	return nil
}

func registerCLIPage(a *Application) {
	a.Manifest().Register("users.edit", func() engine.Page {
		return &cliPage{FormEngine: &formengine.FormEngine{}}
	})
}

func TestExecuteAccessGenerateValidateAndDryRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sfp.access.yaml")
	a := New(Config{Module: "clients", AccessManifestPath: path})
	var output bytes.Buffer

	if err := a.Execute(context.Background(), registerCLIPage, ":0", []string{"access", "generate"}, &output); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "clients.event.users.edit.button.save") {
		t.Fatalf("generated manifest does not contain button access key:\n%s", data)
	}
	if err := a.Execute(context.Background(), registerCLIPage, ":0", []string{"access", "validate"}, &output); err != nil {
		t.Fatal(err)
	}
	if err := a.Execute(context.Background(), registerCLIPage, ":0", []string{"access", "sync", "--dry-run"}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "dry-run: would sync") {
		t.Fatalf("unexpected command output %q", output.String())
	}
}

func TestAccessGenerateCollectsRegisteredGroupElements(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sfp.access.yaml")
	group := access.AccessGroup{
		Code:       "client.card.editing",
		Name:       "Редактирование клиента",
		Type:       access.AccessGroupUI,
		ParentCode: "page.clients.card",
		Enabled:    true,
	}
	a := New(Config{Module: "clients", AccessManifestPath: path})
	if err := a.RegisterAccessGroup(access.AccessGroup{
		Code:    "page.clients.card",
		Name:    "Карточка клиента",
		Type:    access.AccessGroupPage,
		Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.RegisterAccessGroup(group); err != nil {
		t.Fatal(err)
	}
	register := func(a *Application) {
		a.Manifest().Register("clients.card", func() engine.Page {
			return &accessAnnotatedPage{FormEngine: &formengine.FormEngine{}, group: group}
		})
	}
	var output bytes.Buffer
	if err := a.Execute(context.Background(), register, ":0", []string{"access", "generate"}, &output); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"client.card.editing",
		"client.name",
		"noAccessBehavior: readonly",
		"save",
		"noAccessBehavior: hidden",
	} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("generated manifest missing %q:\n%s", expected, data)
		}
	}
}

func TestAccessGenerateCollectsTableElements(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sfp.access.yaml")
	group := access.AccessGroup{
		Code:       "client.table.actions",
		Name:       "Действия таблицы клиентов",
		Type:       access.AccessGroupAction,
		ParentCode: "page.clients.list",
		Enabled:    true,
	}
	a := New(Config{Module: "clients", AccessManifestPath: path})
	if err := a.RegisterAccessGroup(access.AccessGroup{
		Code:    "page.clients.list",
		Name:    "Список клиентов",
		Type:    access.AccessGroupPage,
		Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.RegisterAccessGroup(group); err != nil {
		t.Fatal(err)
	}
	register := func(a *Application) {
		a.Manifest().Register("clients.list", func() engine.Page {
			return &accessAnnotatedTablePage{TableEngine: &tableengine.TableEngine{}, group: group}
		})
	}
	var output bytes.Buffer
	if err := a.Execute(context.Background(), register, ":0", []string{"access", "generate"}, &output); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"table.clients.column.name",
		"table.clients",
		"table.clients.toolbar.export",
		"table.clients.row.delete",
		"table.clients.selected.archive",
		"table.clients.column.status.action.approve",
		"elementType: column",
		"elementType: table",
		"elementType: action",
	} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("generated manifest missing %q:\n%s", expected, data)
		}
	}
}

func TestAccessGenerateRejectsUnknownAnnotatedGroup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sfp.access.yaml")
	unknown := access.AccessGroup{
		Code:    "client.card.edting",
		Type:    access.AccessGroupUI,
		Enabled: true,
	}
	a := New(Config{Module: "clients", AccessManifestPath: path})
	register := func(a *Application) {
		a.Manifest().Register("clients.card", func() engine.Page {
			return &accessAnnotatedPage{FormEngine: &formengine.FormEngine{}, group: unknown}
		})
	}
	var output bytes.Buffer
	err := a.Execute(context.Background(), register, ":0", []string{"access", "generate"}, &output)
	if err == nil || !strings.Contains(err.Error(), "unknown access group") {
		t.Fatalf("expected unknown access group error, got %v", err)
	}
}

func TestModuleRoutePath(t *testing.T) {
	if got := engine.RoutePath("clients", "/page/users"); got != "/clients/page/users" {
		t.Fatalf("unexpected module route %q", got)
	}
	if got := engine.RoutePath("", "/page/users"); got != "/page/users" {
		t.Fatalf("legacy route changed: %q", got)
	}
}

type instanceLifecyclePage struct {
	*formengine.FormEngine
	initCalls *int
}

func (p *instanceLifecyclePage) Init(ctx *engine.BuildContext) error {
	*p.initCalls++
	p.Text("name").DefaultValue(ctx.Params["value"])
	p.Button("save").OnClick(func(runtime *formengine.RuntimeContext) {
		control, err := runtime.GetTextById("name")
		if err != nil {
			runtime.SetError(err)
			return
		}
		control.SetLabel("saved")
	})
	return nil
}

func TestRenderedPageInstanceHandlesEventsWithoutReinitializing(t *testing.T) {
	initCalls := 0
	a := New()
	a.Manifest().Register("users.edit", func() engine.Page {
		return &instanceLifecyclePage{
			FormEngine: &formengine.FormEngine{},
			initCalls:  &initCalls,
		}
	})
	a.registerRoutes()

	first := renderInstance(t, a, "/page/users.edit?value=first")
	second := renderInstance(t, a, "/page/users.edit?value=second")
	if first.InstanceID == second.InstanceID {
		t.Fatal("separate renders returned the same page instance id")
	}
	if !strings.Contains(first.Body, `"defaultValue":"first"`) {
		t.Fatalf("first render did not retain request-specific DSL: %s", first.Body)
	}
	if !strings.Contains(second.Body, `"defaultValue":"second"`) {
		t.Fatalf("second render did not retain request-specific DSL: %s", second.Body)
	}
	if !strings.Contains(first.EventURL, engine.PageInstanceParam+"="+first.InstanceID) {
		t.Fatalf("event URL is not bound to the first instance: %q", first.EventURL)
	}
	if !strings.Contains(second.EventURL, engine.PageInstanceParam+"="+second.InstanceID) {
		t.Fatalf("event URL is not bound to the second instance: %q", second.EventURL)
	}

	beforeEvent := initCalls
	eventRequest := httptest.NewRequest(http.MethodPost, first.EventURL, strings.NewReader(`{}`))
	eventResponse := httptest.NewRecorder()
	a.router.ServeHTTP(eventResponse, eventRequest)
	if eventResponse.Code != http.StatusOK {
		t.Fatalf("event returned %d: %s", eventResponse.Code, eventResponse.Body.String())
	}
	if initCalls != beforeEvent {
		t.Fatalf("event called Init: before=%d after=%d", beforeEvent, initCalls)
	}

	deleteRequest := httptest.NewRequest(
		http.MethodDelete,
		first.InstanceURL,
		nil,
	)
	deleteResponse := httptest.NewRecorder()
	a.router.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete returned %d: %s", deleteResponse.Code, deleteResponse.Body.String())
	}
}

type renderedInstance struct {
	InstanceID  string
	InstanceURL string
	EventURL    string
	Body        string
}

func renderInstance(t *testing.T, a *Application, target string) renderedInstance {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	response := httptest.NewRecorder()
	a.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("render returned %d: %s", response.Code, response.Body.String())
	}
	return decodeRenderedInstance(t, response)
}

func decodeRenderedInstance(t *testing.T, response *httptest.ResponseRecorder) renderedInstance {
	t.Helper()
	var payload struct {
		InstanceID  string `json:"instanceId"`
		InstanceURL string `json:"instanceUrl"`
		DSL         struct {
			Actions []struct {
				Config struct {
					URL string `json:"url"`
				} `json:"config"`
			} `json:"actions"`
		} `json:"dsl"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.InstanceID == "" {
		t.Fatal("render response does not contain instanceId")
	}
	if !strings.Contains(payload.InstanceURL, engine.PageInstanceParam+"="+payload.InstanceID) {
		t.Fatalf("render response does not contain a valid instanceUrl: %q", payload.InstanceURL)
	}
	if len(payload.DSL.Actions) == 0 || payload.DSL.Actions[0].Config.URL == "" {
		t.Fatalf("render response does not contain an event URL: %s", response.Body.String())
	}
	return renderedInstance{
		InstanceID:  payload.InstanceID,
		InstanceURL: payload.InstanceURL,
		EventURL:    payload.DSL.Actions[0].Config.URL,
		Body:        response.Body.String(),
	}
}
