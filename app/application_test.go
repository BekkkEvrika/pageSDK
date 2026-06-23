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

	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/formengine"
)

type cliPage struct {
	*formengine.FormEngine
}

func (p *cliPage) Init(_ *engine.BuildContext) error {
	p.Button("save").OnClick(func(*formengine.RuntimeContext) {})
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
