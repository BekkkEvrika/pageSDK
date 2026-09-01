// Package sidebar provides transport-neutral sidebar menu registration,
// validation, tree building, and publishing contracts.
package sidebar

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/BekkkEvrika/pageSDK/access"
)

// Action identifies the full-snapshot operation requested from a publisher.
type Action string

const (
	ActionRegistration Action = "registration"
	ActionRefresh      Action = "refresh"
	ActionUnregister   Action = "unregister"
)

// Config identifies the owner and destination section of a sidebar tree.
// Publisher is supplied by the application using pageSDK; the SDK itself does
// not depend on RabbitMQ or any other transport.
type Config struct {
	ServiceID  string
	SectionKey string
	Publisher  Publisher
}

// Node is a declarative sidebar item registered in the central registry.
// A node attached to a page receives its target from the page route. A node
// without a page attachment is a group and must have at least one child.
type Node struct {
	Key         string
	Title       string
	ParentKey   string
	AccessGroup access.AccessGroup
	Order       int
}

// Binding attaches a registered sidebar node to one registered page.
type Binding struct {
	NodeKey string
	PageKey string
	Target  string
}

// PublishedNode is the transport payload produced from registered nodes and
// page bindings.
type PublishedNode struct {
	Key       string          `json:"key"`
	Title     string          `json:"title"`
	Target    string          `json:"target"`
	AccessKey string          `json:"accessKey"`
	Order     int             `json:"order"`
	Children  []PublishedNode `json:"children"`
}

// Event is a complete sidebar snapshot. Unregister events contain ServiceID
// only; registration and refresh events contain SectionKey and Nodes.
type Event struct {
	ServiceID  string          `json:"serviceId"`
	SectionKey string          `json:"sectionKey,omitempty"`
	Nodes      []PublishedNode `json:"nodes,omitempty"`
}

// Publisher is implemented by an application-specific transport adapter.
// RabbitMQ exchange names, headers, serialization, retries, and connection
// management belong to that adapter rather than pageSDK.
type Publisher interface {
	PublishSidebar(ctx context.Context, action Action, event Event) error
}

// PublisherFunc adapts a function to Publisher.
type PublisherFunc func(ctx context.Context, action Action, event Event) error

func (f PublisherFunc) PublishSidebar(ctx context.Context, action Action, event Event) error {
	return f(ctx, action, event)
}

// AccessGroupRegistry is the minimum access registry contract needed to
// validate node access groups.
type AccessGroupRegistry interface {
	Get(code string) (access.AccessGroup, bool)
}

// Registry stores sidebar node declarations before application bootstrap.
type Registry struct {
	mu    sync.RWMutex
	nodes map[string]Node
}

func NewRegistry() *Registry {
	return &Registry{nodes: map[string]Node{}}
}

// Register adds one node to the central registry.
func (r *Registry) Register(node Node) error {
	return r.RegisterMany(node)
}

// RegisterMany adds a batch atomically.
func (r *Registry) RegisterMany(nodes ...Node) error {
	if r == nil {
		return errors.New("sidebar registry is nil")
	}
	normalized := make([]Node, len(nodes))
	seen := make(map[string]struct{}, len(nodes))
	for i, node := range nodes {
		node = normalizeNode(node)
		if node.Key == "" {
			return errors.New("sidebar node key must not be empty")
		}
		if _, exists := seen[node.Key]; exists {
			return fmt.Errorf("duplicate sidebar node: %s", node.Key)
		}
		seen[node.Key] = struct{}{}
		normalized[i] = node
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, node := range normalized {
		if _, exists := r.nodes[node.Key]; exists {
			return fmt.Errorf("duplicate sidebar node: %s", node.Key)
		}
	}
	for _, node := range normalized {
		r.nodes[node.Key] = node
	}
	return nil
}

func normalizeNode(node Node) Node {
	node.Key = strings.TrimSpace(node.Key)
	node.Title = strings.TrimSpace(node.Title)
	node.ParentKey = strings.TrimSpace(node.ParentKey)
	node.AccessGroup.Code = strings.TrimSpace(node.AccessGroup.Code)
	return node
}

// Get returns a registered node by stable key.
func (r *Registry) Get(key string) (Node, bool) {
	if r == nil {
		return Node{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	node, ok := r.nodes[strings.TrimSpace(key)]
	return node, ok
}

// All returns registered nodes sorted by stable key.
func (r *Registry) All() []Node {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Node, 0, len(r.nodes))
	for _, node := range r.nodes {
		result = append(result, node)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}

func (r *Registry) Len() int {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.nodes)
}

// BuildEvent validates declarations and produces the full nested snapshot.
func (r *Registry) BuildEvent(config Config, bindings []Binding, accessGroups AccessGroupRegistry) (Event, error) {
	serviceID := strings.TrimSpace(config.ServiceID)
	sectionKey := strings.TrimSpace(config.SectionKey)
	if serviceID == "" {
		return Event{}, errors.New("sidebar service ID must not be empty")
	}
	if sectionKey == "" {
		return Event{}, errors.New("sidebar section key must not be empty")
	}

	nodes := r.All()
	if len(nodes) == 0 {
		return Event{}, errors.New("sidebar nodes must not be empty")
	}
	byKey := make(map[string]Node, len(nodes))
	children := make(map[string][]string, len(nodes))
	for _, node := range nodes {
		if node.Title == "" {
			return Event{}, fmt.Errorf("sidebar node %q title must not be empty", node.Key)
		}
		if node.AccessGroup.Code == "" {
			return Event{}, fmt.Errorf("sidebar node %q access group must not be empty", node.Key)
		}
		if accessGroups == nil {
			return Event{}, errors.New("sidebar access group registry is nil")
		}
		if _, ok := accessGroups.Get(node.AccessGroup.Code); !ok {
			return Event{}, fmt.Errorf("sidebar node %q references unknown access group %q", node.Key, node.AccessGroup.Code)
		}
		byKey[node.Key] = node
	}
	for _, node := range nodes {
		if node.ParentKey == "" {
			continue
		}
		if _, ok := byKey[node.ParentKey]; !ok {
			return Event{}, fmt.Errorf("sidebar node %q references unknown parent %q", node.Key, node.ParentKey)
		}
		children[node.ParentKey] = append(children[node.ParentKey], node.Key)
	}
	if err := validateAcyclic(nodes, byKey); err != nil {
		return Event{}, err
	}

	targets := make(map[string]string, len(bindings))
	for _, binding := range bindings {
		nodeKey := strings.TrimSpace(binding.NodeKey)
		if _, ok := byKey[nodeKey]; !ok {
			return Event{}, fmt.Errorf("page %q references unknown sidebar node %q", binding.PageKey, nodeKey)
		}
		target := strings.TrimSpace(binding.Target)
		if target == "" {
			return Event{}, fmt.Errorf("sidebar node %q has an empty target from page %q", nodeKey, binding.PageKey)
		}
		if existing, ok := targets[nodeKey]; ok && existing != target {
			return Event{}, fmt.Errorf("sidebar node %q is attached to multiple pages", nodeKey)
		}
		targets[nodeKey] = target
	}

	for _, node := range nodes {
		if targets[node.Key] == "" && len(children[node.Key]) == 0 {
			return Event{}, fmt.Errorf("sidebar node %q must be attached to a page or have children", node.Key)
		}
	}

	rootKeys := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if node.ParentKey == "" {
			rootKeys = append(rootKeys, node.Key)
		}
	}
	sortNodeKeys(rootKeys, byKey)
	for parent := range children {
		sortNodeKeys(children[parent], byKey)
	}

	result := make([]PublishedNode, 0, len(rootKeys))
	for _, key := range rootKeys {
		result = append(result, buildPublishedNode(key, byKey, children, targets))
	}
	return Event{ServiceID: serviceID, SectionKey: sectionKey, Nodes: result}, nil
}

func validateAcyclic(nodes []Node, byKey map[string]Node) error {
	const (
		unvisited = iota
		visiting
		visited
	)
	state := make(map[string]int, len(nodes))
	var visit func(string) error
	visit = func(key string) error {
		switch state[key] {
		case visiting:
			return fmt.Errorf("sidebar hierarchy contains a cycle at node %q", key)
		case visited:
			return nil
		}
		state[key] = visiting
		if parent := byKey[key].ParentKey; parent != "" {
			if err := visit(parent); err != nil {
				return err
			}
		}
		state[key] = visited
		return nil
	}
	for _, node := range nodes {
		if err := visit(node.Key); err != nil {
			return err
		}
	}
	return nil
}

func sortNodeKeys(keys []string, nodes map[string]Node) {
	sort.Slice(keys, func(i, j int) bool {
		left, right := nodes[keys[i]], nodes[keys[j]]
		if left.Order == right.Order {
			return left.Key < right.Key
		}
		return left.Order < right.Order
	})
}

func buildPublishedNode(key string, nodes map[string]Node, children map[string][]string, targets map[string]string) PublishedNode {
	node := nodes[key]
	result := PublishedNode{
		Key:       node.Key,
		Title:     node.Title,
		Target:    targets[key],
		AccessKey: node.AccessGroup.Code,
		Order:     node.Order,
		Children:  make([]PublishedNode, 0, len(children[key])),
	}
	for _, childKey := range children[key] {
		result.Children = append(result.Children, buildPublishedNode(childKey, nodes, children, targets))
	}
	return result
}

// UnregisterEvent creates the transport payload used to remove a service tree.
func UnregisterEvent(serviceID string) (Event, error) {
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		return Event{}, errors.New("sidebar service ID must not be empty")
	}
	return Event{ServiceID: serviceID}, nil
}
