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
	if err := manager.Add("instance", "users.edit", page); err != nil {
		t.Fatal(err)
	}

	now = now.Add(time.Minute + time.Second)
	if _, err := manager.Acquire("instance", "users.edit"); !errors.Is(err, ErrPageInstanceExpired) {
		t.Fatalf("expected expired instance, got %v", err)
	}
}
