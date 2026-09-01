package sidebar

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/BekkkEvrika/pageSDK/access"
)

func TestRegistryBuildEventCreatesSortedTree(t *testing.T) {
	viewing := access.AccessGroup{Code: "orders.viewing"}
	listViewing := access.AccessGroup{Code: "orders.list.viewing"}
	createViewing := access.AccessGroup{Code: "orders.create"}
	groups := access.NewRegistry()
	for _, group := range []access.AccessGroup{viewing, listViewing, createViewing} {
		if err := groups.Register(group); err != nil {
			t.Fatal(err)
		}
	}

	registry := NewRegistry()
	if err := registry.RegisterMany(
		Node{Key: "orders.create", Title: "Create", ParentKey: "orders", AccessGroup: createViewing, Order: 20},
		Node{Key: "orders", Title: "Orders", AccessGroup: viewing, Order: 10},
		Node{Key: "orders.list", Title: "List", ParentKey: "orders", AccessGroup: listViewing, Order: 10},
	); err != nil {
		t.Fatal(err)
	}

	event, err := registry.BuildEvent(Config{ServiceID: "orders-service", SectionKey: "sales"}, []Binding{
		{NodeKey: "orders.create", PageKey: "orders.create", Target: "/orders/page/orders.create"},
		{NodeKey: "orders.list", PageKey: "orders.list", Target: "/orders/page/orders.list"},
	}, groups)
	if err != nil {
		t.Fatal(err)
	}
	if event.ServiceID != "orders-service" || event.SectionKey != "sales" {
		t.Fatalf("unexpected event identity: %#v", event)
	}
	if len(event.Nodes) != 1 || event.Nodes[0].Key != "orders" || event.Nodes[0].Target != "" {
		t.Fatalf("unexpected root nodes: %#v", event.Nodes)
	}
	children := event.Nodes[0].Children
	if len(children) != 2 || children[0].Key != "orders.list" || children[1].Key != "orders.create" {
		t.Fatalf("children are not sorted by order: %#v", children)
	}
	if children[0].AccessKey != listViewing.Code || children[0].Target != "/orders/page/orders.list" {
		t.Fatalf("unexpected page node: %#v", children[0])
	}
}

func TestRegistryRegisterManyIsAtomic(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Node{Key: "existing"}); err != nil {
		t.Fatal(err)
	}
	err := registry.RegisterMany(Node{Key: "new"}, Node{Key: "existing"})
	if err == nil {
		t.Fatal("expected duplicate error")
	}
	if _, exists := registry.Get("new"); exists {
		t.Fatal("failed batch left a partially registered node")
	}
}

func TestRegistryBuildEventRejectsInvalidHierarchy(t *testing.T) {
	group := access.AccessGroup{Code: "orders.viewing"}
	groups := access.NewRegistry()
	if err := groups.Register(group); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		nodes []Node
		want  string
	}{
		{
			name:  "unknown parent",
			nodes: []Node{{Key: "orders", Title: "Orders", ParentKey: "missing", AccessGroup: group}},
			want:  "unknown parent",
		},
		{
			name: "cycle",
			nodes: []Node{
				{Key: "orders", Title: "Orders", ParentKey: "reports", AccessGroup: group},
				{Key: "reports", Title: "Reports", ParentKey: "orders", AccessGroup: group},
			},
			want: "cycle",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			if err := registry.RegisterMany(tt.nodes...); err != nil {
				t.Fatal(err)
			}
			_, err := registry.BuildEvent(Config{ServiceID: "orders", SectionKey: "sales"}, nil, groups)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestRegistryBuildEventRejectsUnknownAccessGroupAndUnboundLeaf(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Node{
		Key:         "orders",
		Title:       "Orders",
		AccessGroup: access.AccessGroup{Code: "orders.viewing"},
	}); err != nil {
		t.Fatal(err)
	}
	groups := access.NewRegistry()
	_, err := registry.BuildEvent(Config{ServiceID: "orders", SectionKey: "sales"}, nil, groups)
	if err == nil || !strings.Contains(err.Error(), "unknown access group") {
		t.Fatalf("expected unknown access group error, got %v", err)
	}
	if err := groups.Register(access.AccessGroup{Code: "orders.viewing"}); err != nil {
		t.Fatal(err)
	}
	_, err = registry.BuildEvent(Config{ServiceID: "orders", SectionKey: "sales"}, nil, groups)
	if err == nil || !strings.Contains(err.Error(), "attached to a page or have children") {
		t.Fatalf("expected unbound leaf error, got %v", err)
	}
}

func TestUnregisterEventContainsServiceIDOnly(t *testing.T) {
	event, err := UnregisterEvent("orders-service")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"serviceId":"orders-service"}` {
		t.Fatalf("unexpected unregister JSON: %s", data)
	}
}
