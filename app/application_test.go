package app

import (
	"bytes"
	"context"
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
