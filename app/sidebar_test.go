package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/BekkkEvrika/pageSDK/access"
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/formengine"
	"github.com/BekkkEvrika/pageSDK/engine/tableengine"
	"github.com/BekkkEvrika/pageSDK/sidebar"
)

type sidebarFormPage struct {
	*formengine.FormEngine
	node sidebar.Node
}

func (p *sidebarFormPage) Init(_ *engine.BuildContext) error {
	p.Sidebar(p.node)
	p.Text("name")
	return nil
}

type sidebarTablePage struct {
	*tableengine.TableEngine
	node sidebar.Node
}

func (p *sidebarTablePage) Init(_ *engine.BuildContext) error {
	p.Sidebar(p.node)
	p.Table("orders")
	return nil
}

type recordedSidebarPublish struct {
	action sidebar.Action
	event  sidebar.Event
}

type recordingSidebarPublisher struct {
	values []recordedSidebarPublish
}

func (p *recordingSidebarPublisher) PublishSidebar(_ context.Context, action sidebar.Action, event sidebar.Event) error {
	p.values = append(p.values, recordedSidebarPublish{action: action, event: event})
	return nil
}

func TestApplicationPublishesSidebarFromFormAndTableBindings(t *testing.T) {
	rootGroup := access.AccessGroup{Code: "orders.viewing"}
	listGroup := access.AccessGroup{Code: "orders.list.viewing"}
	createGroup := access.AccessGroup{Code: "orders.create"}
	rootNode := sidebar.Node{Key: "orders", Title: "Orders", AccessGroup: rootGroup, Order: 10}
	listNode := sidebar.Node{Key: "orders.list", Title: "List", ParentKey: rootNode.Key, AccessGroup: listGroup, Order: 10}
	createNode := sidebar.Node{Key: "orders.create", Title: "Create", ParentKey: rootNode.Key, AccessGroup: createGroup, Order: 20}
	publisher := &recordingSidebarPublisher{}
	a := New(Config{
		Module: "orders",
		Sidebar: sidebar.Config{
			ServiceID:  "orders-service",
			SectionKey: "sales",
			Publisher:  publisher,
		},
	})
	for _, group := range []access.AccessGroup{rootGroup, listGroup, createGroup} {
		if err := a.RegisterAccessGroup(group); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.RegisterSidebarNodes(rootNode, listNode, createNode); err != nil {
		t.Fatal(err)
	}
	a.Manifest().Register("orders.list", func() engine.Page {
		return &sidebarTablePage{TableEngine: &tableengine.TableEngine{}, node: listNode}
	})
	a.Manifest().Register("orders.create", func() engine.Page {
		return &sidebarFormPage{FormEngine: &formengine.FormEngine{}, node: createNode}
	})

	if err := a.PublishSidebar(context.Background(), sidebar.ActionRegistration); err != nil {
		t.Fatal(err)
	}
	if len(publisher.values) != 1 || publisher.values[0].action != sidebar.ActionRegistration {
		t.Fatalf("unexpected publishes: %#v", publisher.values)
	}
	root := publisher.values[0].event.Nodes[0]
	if len(root.Children) != 2 {
		t.Fatalf("unexpected sidebar tree: %#v", root)
	}
	if root.Children[0].Target != "/orders/page/orders.list" {
		t.Fatalf("unexpected table target: %q", root.Children[0].Target)
	}
	if root.Children[1].Target != "/orders/page/orders.create" {
		t.Fatalf("unexpected form target: %q", root.Children[1].Target)
	}

	if err := a.PublishSidebar(context.Background(), sidebar.ActionRefresh); err != nil {
		t.Fatal(err)
	}
	if err := a.PublishSidebar(context.Background(), sidebar.ActionUnregister); err != nil {
		t.Fatal(err)
	}
	if len(publisher.values) != 3 || publisher.values[2].event.ServiceID != "orders-service" {
		t.Fatalf("unexpected publish lifecycle: %#v", publisher.values)
	}
	if publisher.values[2].event.SectionKey != "" || publisher.values[2].event.Nodes != nil {
		t.Fatalf("unregister must contain service ID only: %#v", publisher.values[2].event)
	}
}

func TestApplicationSidebarRequiresPublisher(t *testing.T) {
	a := New(Config{})
	err := a.PublishSidebar(context.Background(), sidebar.ActionUnregister)
	if err == nil {
		t.Fatal("expected missing publisher error")
	}
}

func TestBootstrapPublishesRegistrationBeforeStartingServer(t *testing.T) {
	group := access.AccessGroup{Code: "orders.viewing"}
	node := sidebar.Node{Key: "orders", Title: "Orders", AccessGroup: group}
	publishErr := errors.New("broker unavailable")
	calls := 0
	a := New(Config{Sidebar: sidebar.Config{
		ServiceID:  "orders-service",
		SectionKey: "sales",
		Publisher: sidebar.PublisherFunc(func(_ context.Context, action sidebar.Action, _ sidebar.Event) error {
			calls++
			if action != sidebar.ActionRegistration {
				t.Fatalf("unexpected bootstrap action: %s", action)
			}
			return publishErr
		}),
	}})

	err := a.Bootstrap(func(app *Application) {
		if registerErr := app.RegisterAccessGroup(group); registerErr != nil {
			t.Fatal(registerErr)
		}
		if registerErr := app.RegisterSidebarNode(node); registerErr != nil {
			t.Fatal(registerErr)
		}
		app.Manifest().Register("orders.list", func() engine.Page {
			return &sidebarFormPage{FormEngine: &formengine.FormEngine{}, node: node}
		})
	}, "invalid-address-that-must-not-be-used")
	if err == nil || !strings.Contains(err.Error(), publishErr.Error()) {
		t.Fatalf("expected publisher error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected one bootstrap publication, got %d", calls)
	}
}
