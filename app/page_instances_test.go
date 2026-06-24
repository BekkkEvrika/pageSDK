package app

import (
	"errors"
	"testing"
	"time"

	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/formengine"
)

type storedTestPage struct {
	*formengine.FormEngine
}

func (p *storedTestPage) Init(*engine.BuildContext) error { return nil }

func TestPageInstanceExpiresAfterIdleTTL(t *testing.T) {
	manager := newPageInstanceManager(time.Minute, 10)
	now := time.Unix(100, 0)
	manager.now = func() time.Time { return now }
	page := &storedTestPage{FormEngine: &formengine.FormEngine{}}
	if err := manager.Add("instance", "users.edit", "issuer|user-1", page); err != nil {
		t.Fatal(err)
	}

	now = now.Add(time.Minute + time.Second)
	if _, err := manager.Acquire("instance", "users.edit", "issuer|user-1"); !errors.Is(err, ErrPageInstanceExpired) {
		t.Fatalf("expected expired instance, got %v", err)
	}
}

func TestPageInstanceOwnerMustMatch(t *testing.T) {
	manager := newPageInstanceManager(time.Minute, 10)
	page := &storedTestPage{FormEngine: &formengine.FormEngine{}}
	if err := manager.Add("instance", "users.edit", "issuer|user-1", page); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Acquire("instance", "users.edit", "issuer|user-2"); !errors.Is(err, ErrPageInstanceNotFound) {
		t.Fatalf("expected owner mismatch to look like not found, got %v", err)
	}
	if manager.Delete("instance", "users.edit", "issuer|user-2") {
		t.Fatal("another owner deleted the page instance")
	}
}
