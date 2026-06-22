// Package access implements UI access manifest generation and validation.
package access

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BekkkEvrika/pageSDK/manifest"
	"github.com/BekkkEvrika/pageSDK/table"
	"gopkg.in/yaml.v3"
)

const CurrentVersion = 1

type Config struct {
	Module       string
	ManifestPath string
	KeycloakURL  string
	Realm        string
	ClientID     string
	ClientSecret string
}

type Manifest struct {
	Module           string            `yaml:"module,omitempty"`
	Version          int               `yaml:"version"`
	Resources        []Resource        `yaml:"resources,omitempty"`
	PermissionGroups []PermissionGroup `yaml:"permissionGroups,omitempty"`
	Stale            []StaleResource   `yaml:"stale,omitempty"`
}

type Resource struct {
	Key           string `yaml:"key"`
	Type          string `yaml:"type"`
	Page          string `yaml:"page,omitempty"`
	ComponentType string `yaml:"componentType,omitempty"`
	ComponentKey  string `yaml:"componentKey,omitempty"`
	TableKey      string `yaml:"tableKey,omitempty"`
	ColumnKey     string `yaml:"columnKey,omitempty"`
	ActionKey     string `yaml:"actionKey,omitempty"`
	Description   string `yaml:"description,omitempty"`
}

type PermissionGroup struct {
	Key         string   `yaml:"key"`
	Name        string   `yaml:"name,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Permissions []string `yaml:"permissions,omitempty"`
}

type StaleResource struct {
	Key    string `yaml:"key"`
	Reason string `yaml:"reason"`
}

type Diff struct {
	NewInDSL               []string
	MissingInDSL           []string
	MissingInManifest      []string
	ExistingGroups         []string
	BrokenGroupPermissions []string
}

type SyncOptions struct {
	DryRun bool
}

type AccessSyncProvider interface {
	Sync(ctx context.Context, manifest Manifest, opts SyncOptions) error
	Diff(ctx context.Context, manifest Manifest) (*Diff, error)
}

type UnsupportedKeycloakProvider struct {
	Config Config
}

func (p UnsupportedKeycloakProvider) Sync(_ context.Context, _ Manifest, opts SyncOptions) error {
	if opts.DryRun {
		return nil
	}
	return errors.New("access sync: Keycloak UMA provider is not configured; use SetAccessSyncProvider or --dry-run")
}

func (p UnsupportedKeycloakProvider) Diff(_ context.Context, _ Manifest) (*Diff, error) {
	return nil, errors.New("access diff: Keycloak UMA provider is not configured")
}

func Key(module string, parts ...string) string {
	items := make([]string, 0, len(parts)+1)
	if module = strings.Trim(module, ". "); module != "" {
		items = append(items, module)
	}
	for _, part := range parts {
		if part = strings.Trim(part, ". "); part != "" {
			items = append(items, part)
		}
	}
	return strings.Join(items, ".")
}

func Collect(registry *manifest.Manifest, module string) ([]Resource, error) {
	var resources []Resource
	seen := map[string]struct{}{}
	add := func(resource Resource) error {
		if resource.Key == "" {
			return errors.New("access collector: empty resource key")
		}
		if _, ok := seen[resource.Key]; ok {
			return fmt.Errorf("access collector: duplicate resource key %q", resource.Key)
		}
		seen[resource.Key] = struct{}{}
		resources = append(resources, resource)
		return nil
	}

	for _, entry := range registry.All() {
		page := entry.Factory()
		engineInstance := page.GetEngine()
		routes := engineInstance.Routes(entry.Key, page)
		if err := add(Resource{
			Key:         Key(module, "page", entry.Key),
			Type:        "page",
			Page:        entry.Key,
			Description: "Page " + entry.Key,
		}); err != nil {
			return nil, err
		}

		for _, route := range routes {
			resource, ok := resourceFromRoute(module, entry.Key, route.Path)
			if !ok {
				continue
			}
			if err := add(resource); err != nil {
				return nil, err
			}
		}

		if schemaProvider, ok := engineInstance.(interface{ Schema() *table.TableSchema }); ok {
			schema := schemaProvider.Schema()
			tableKey := schema.ID
			if tableKey == "" {
				tableKey = entry.Key
			}
			for _, column := range schema.Columns {
				if column.ID == "" {
					continue
				}
				if err := add(Resource{
					Key:           Key(module, "ui", entry.Key, "table", tableKey, "column", column.ID, "view"),
					Type:          "ui",
					Page:          entry.Key,
					ComponentType: "table",
					ComponentKey:  tableKey,
					TableKey:      tableKey,
					ColumnKey:     column.ID,
					Description:   fmt.Sprintf("Column %s in table %s", column.ID, tableKey),
				}); err != nil {
					return nil, err
				}
			}
		}
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Key < resources[j].Key })
	return resources, nil
}

func resourceFromRoute(module, pageKey, path string) (Resource, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 || parts[0] != "event" || parts[1] != pageKey {
		return Resource{}, false
	}
	eventParts := append([]string{"event", pageKey}, parts[2:]...)
	resource := Resource{
		Key:         Key(module, eventParts...),
		Type:        "event",
		Page:        pageKey,
		Description: "Event " + strings.Join(parts[2:], " "),
	}
	switch parts[2] {
	case "table":
		if len(parts) < 5 {
			return Resource{}, false
		}
		resource.ComponentType = "table"
		resource.ComponentKey = parts[3]
		resource.TableKey = parts[3]
		switch parts[4] {
		case "row", "toolbar", "selected":
			if len(parts) != 6 {
				return Resource{}, false
			}
			resource.ActionKey = parts[5]
		case "column":
			if len(parts) != 7 {
				return Resource{}, false
			}
			resource.ColumnKey = parts[5]
			resource.ActionKey = parts[6]
		default:
			resource.ActionKey = parts[4]
		}
	case "dialog", "callback":
		if len(parts) != 4 {
			return Resource{}, false
		}
		resource.ComponentType = parts[2]
		resource.ComponentKey = "*"
		resource.Key = Key(module, "event", pageKey, parts[2], "*")
	default:
		if len(parts) != 4 {
			return Resource{}, false
		}
		resource.ComponentType = parts[2]
		resource.ComponentKey = parts[3]
		resource.ActionKey = parts[3]
	}
	return resource, true
}

func Read(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var result Manifest
	if err := yaml.Unmarshal(data, &result); err != nil {
		return Manifest{}, fmt.Errorf("read access manifest: %w", err)
	}
	return result, nil
}

func Write(path string, value Manifest) error {
	if path == "" {
		path = "sfp.access.yaml"
	}
	if value.Version == 0 {
		value.Version = CurrentVersion
	}
	sortManifest(&value)
	data, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal access manifest: %w", err)
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

func Generate(path, module string, resources []Resource) (Manifest, error) {
	current := Manifest{}
	if existing, err := Read(path); err == nil {
		current = existing
	} else if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, err
	}
	merged := Merge(current, module, resources)
	if err := Write(path, merged); err != nil {
		return Manifest{}, err
	}
	return merged, nil
}

func Merge(current Manifest, module string, discovered []Resource) Manifest {
	result := current
	result.Module = module
	result.Version = CurrentVersion

	oldResources := make(map[string]Resource, len(current.Resources))
	for _, item := range current.Resources {
		oldResources[item.Key] = item
	}
	discoveredKeys := make(map[string]struct{}, len(discovered))
	result.Resources = make([]Resource, 0, len(discovered))
	for _, item := range discovered {
		discoveredKeys[item.Key] = struct{}{}
		if old, ok := oldResources[item.Key]; ok {
			item = preserveResourceMetadata(item, old)
		}
		result.Resources = append(result.Resources, item)
	}

	stale := map[string]StaleResource{}
	for _, item := range current.Stale {
		if _, active := discoveredKeys[item.Key]; !active {
			stale[item.Key] = item
		}
	}
	for _, item := range current.Resources {
		if _, active := discoveredKeys[item.Key]; !active {
			stale[item.Key] = StaleResource{Key: item.Key, Reason: "Not found in current DSL"}
		}
	}
	result.Stale = result.Stale[:0]
	for _, item := range stale {
		result.Stale = append(result.Stale, item)
	}
	sortManifest(&result)
	return result
}

func preserveResourceMetadata(discovered, old Resource) Resource {
	if old.Description != "" {
		discovered.Description = old.Description
	}
	return discovered
}

func Validate(value Manifest, module string) error {
	var problems []string
	if value.Version <= 0 {
		problems = append(problems, "version must be greater than zero")
	}
	if value.Module != module {
		problems = append(problems, fmt.Sprintf("manifest module %q does not match config module %q", value.Module, module))
	}
	resourceKeys := map[string]struct{}{}
	for _, item := range value.Resources {
		if item.Key == "" {
			problems = append(problems, "resource key must not be empty")
			continue
		}
		if _, exists := resourceKeys[item.Key]; exists {
			problems = append(problems, "duplicate resource key: "+item.Key)
		}
		resourceKeys[item.Key] = struct{}{}
	}
	staleKeys := map[string]struct{}{}
	for _, item := range value.Stale {
		if item.Key == "" {
			problems = append(problems, "stale key must not be empty")
			continue
		}
		if _, exists := staleKeys[item.Key]; exists {
			problems = append(problems, "duplicate stale key: "+item.Key)
		}
		if _, active := resourceKeys[item.Key]; active {
			problems = append(problems, "key is both active and stale: "+item.Key)
		}
		staleKeys[item.Key] = struct{}{}
	}
	groupKeys := map[string]struct{}{}
	for _, group := range value.PermissionGroups {
		if strings.TrimSpace(group.Key) == "" {
			problems = append(problems, "permission group key must not be empty")
		} else if _, exists := groupKeys[group.Key]; exists {
			problems = append(problems, "duplicate permission group key: "+group.Key)
		}
		groupKeys[group.Key] = struct{}{}
		for _, permission := range group.Permissions {
			if matchesAny(permission, staleKeys) {
				problems = append(problems, fmt.Sprintf("group %q uses stale permission %q", group.Key, permission))
				continue
			}
			if !matchesAny(permission, resourceKeys) {
				problems = append(problems, fmt.Sprintf("group %q references unknown permission %q", group.Key, permission))
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return errors.New(strings.Join(problems, "\n"))
	}
	return nil
}

func Compare(discovered []Resource, value Manifest) Diff {
	dsl := map[string]struct{}{}
	for _, item := range discovered {
		dsl[item.Key] = struct{}{}
	}
	active := map[string]struct{}{}
	for _, item := range value.Resources {
		active[item.Key] = struct{}{}
		if _, ok := dsl[item.Key]; !ok {
			// Active manifest resource disappeared from the DSL.
		}
	}
	stale := map[string]struct{}{}
	for _, item := range value.Stale {
		stale[item.Key] = struct{}{}
	}
	diff := Diff{}
	for key := range dsl {
		if _, ok := active[key]; !ok {
			diff.NewInDSL = append(diff.NewInDSL, key)
			diff.MissingInManifest = append(diff.MissingInManifest, key)
		}
	}
	for key := range active {
		if _, ok := dsl[key]; !ok {
			diff.MissingInDSL = append(diff.MissingInDSL, key)
		}
	}
	for _, group := range value.PermissionGroups {
		diff.ExistingGroups = append(diff.ExistingGroups, group.Key)
		for _, permission := range group.Permissions {
			if matchesAny(permission, stale) || !matchesAny(permission, active) || !matchesAny(permission, dsl) {
				diff.BrokenGroupPermissions = append(diff.BrokenGroupPermissions, group.Key+": "+permission)
			}
		}
	}
	sortStringsInDiff(&diff)
	return diff
}

func matchesAny(permission string, keys map[string]struct{}) bool {
	if _, ok := keys[permission]; ok {
		return true
	}
	if !strings.HasSuffix(permission, "*") {
		return false
	}
	prefix := strings.TrimSuffix(permission, "*")
	for key := range keys {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func sortManifest(value *Manifest) {
	sort.Slice(value.Resources, func(i, j int) bool { return value.Resources[i].Key < value.Resources[j].Key })
	sort.Slice(value.PermissionGroups, func(i, j int) bool { return value.PermissionGroups[i].Key < value.PermissionGroups[j].Key })
	sort.Slice(value.Stale, func(i, j int) bool { return value.Stale[i].Key < value.Stale[j].Key })
	for i := range value.PermissionGroups {
		sort.Strings(value.PermissionGroups[i].Permissions)
	}
}

func sortStringsInDiff(diff *Diff) {
	sort.Strings(diff.NewInDSL)
	sort.Strings(diff.MissingInDSL)
	sort.Strings(diff.MissingInManifest)
	sort.Strings(diff.ExistingGroups)
	sort.Strings(diff.BrokenGroupPermissions)
}
