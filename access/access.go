// Package access implements UI access manifest generation and validation.
package access

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BekkkEvrika/pageSDK/manifest"
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
	SyncEnabled  bool
	CacheTTL     time.Duration
	HTTPClient   *http.Client
}

type Manifest struct {
	Module           string            `yaml:"module,omitempty"`
	Version          int               `yaml:"version"`
	AccessGroups     []AccessGroup     `yaml:"accessGroups,omitempty"`
	Resources        []Resource        `yaml:"resources,omitempty"`
	PermissionGroups []PermissionGroup `yaml:"permissionGroups,omitempty"`
	Stale            []StaleResource   `yaml:"stale,omitempty"`
}

type AccessGroupType string

const (
	AccessGroupPage   AccessGroupType = "page"
	AccessGroupUI     AccessGroupType = "ui_group"
	AccessGroupAction AccessGroupType = "action_group"
)

type ElementType string

const (
	ElementButton  ElementType = "button"
	ElementInput   ElementType = "input"
	ElementTable   ElementType = "table"
	ElementColumn  ElementType = "column"
	ElementBlock   ElementType = "block"
	ElementSection ElementType = "section"
	ElementMenu    ElementType = "menu"
	ElementTab     ElementType = "tab"
	ElementAction  ElementType = "action"
	ElementCustom  ElementType = "custom"
)

type NoAccessBehavior string

const (
	NoAccessHidden   NoAccessBehavior = "hidden"
	NoAccessDisabled NoAccessBehavior = "disabled"
	NoAccessReadonly NoAccessBehavior = "readonly"
	NoAccessRemove   NoAccessBehavior = "remove"
)

type AccessGroup struct {
	Code        string          `yaml:"code" json:"code"`
	Name        string          `yaml:"name,omitempty" json:"name,omitempty"`
	Description string          `yaml:"description,omitempty" json:"description,omitempty"`
	Type        AccessGroupType `yaml:"type" json:"type"`
	ParentCode  string          `yaml:"parentCode,omitempty" json:"parentCode,omitempty"`
	Elements    []AccessElement `yaml:"elements,omitempty" json:"elements,omitempty"`
	Enabled     bool            `yaml:"enabled" json:"enabled"`
}

type AccessElement struct {
	Code             string           `yaml:"code" json:"code"`
	Name             string           `yaml:"name,omitempty" json:"name,omitempty"`
	Description      string           `yaml:"description,omitempty" json:"description,omitempty"`
	ElementType      ElementType      `yaml:"elementType" json:"elementType"`
	NoAccessBehavior NoAccessBehavior `yaml:"noAccessBehavior" json:"noAccessBehavior"`
}

type ElementBinding struct {
	GroupCode string
	Page      string
	Element   AccessElement
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
	Code         string   `yaml:"code,omitempty"`
	Key          string   `yaml:"key,omitempty"`
	Name         string   `yaml:"name,omitempty"`
	Description  string   `yaml:"description,omitempty"`
	AccessGroups []string `yaml:"accessGroups,omitempty"`
	Permissions  []string `yaml:"permissions,omitempty"`
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

type AccessAuthorizer interface {
	UserAccessGroups(ctx context.Context, userID string, user map[string]any) ([]string, error)
	HasAccess(ctx context.Context, userID string, user map[string]any, accessGroupCode string) (bool, error)
	Invalidate()
}

type AccessSyncProvider interface {
	Sync(ctx context.Context, manifest Manifest, opts SyncOptions) error
	Diff(ctx context.Context, manifest Manifest) (*Diff, error)
}

type Registry struct {
	mu     sync.RWMutex
	groups map[string]AccessGroup
}

func NewRegistry() *Registry {
	return &Registry{groups: map[string]AccessGroup{}}
}

func (r *Registry) Register(group AccessGroup) error {
	if strings.TrimSpace(group.Code) == "" {
		return errors.New("access group code must not be empty")
	}
	if group.Type == "" {
		return errors.New("access group type must not be empty")
	}
	if !group.Enabled {
		group.Enabled = true
	}
	for i := range group.Elements {
		if strings.TrimSpace(group.Elements[i].Code) == "" {
			return fmt.Errorf("access group %q has element with empty code", group.Code)
		}
		if group.Elements[i].NoAccessBehavior == "" {
			group.Elements[i].NoAccessBehavior = NoAccessHidden
		}
		if group.Elements[i].ElementType == "" {
			group.Elements[i].ElementType = ElementCustom
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.groups[group.Code]; exists {
		return fmt.Errorf("duplicate access group: %s", group.Code)
	}
	r.groups[group.Code] = group
	return nil
}

func (r *Registry) All() []AccessGroup {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	groups := make([]AccessGroup, 0, len(r.groups))
	for _, group := range r.groups {
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Code < groups[j].Code })
	return groups
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

		if schema, ok := tableSchemaMetadata(engineInstance); ok {
			tableKey := schema.ID
			if tableKey == "" {
				tableKey = entry.Key
			}
			for _, columnID := range schema.ColumnIDs {
				if columnID == "" {
					continue
				}
				if err := add(Resource{
					Key:           Key(module, "ui", entry.Key, "table", tableKey, "column", columnID, "view"),
					Type:          "ui",
					Page:          entry.Key,
					ComponentType: "table",
					ComponentKey:  tableKey,
					TableKey:      tableKey,
					ColumnKey:     columnID,
					Description:   fmt.Sprintf("Column %s in table %s", columnID, tableKey),
				}); err != nil {
					return nil, err
				}
			}
		}
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Key < resources[j].Key })
	return resources, nil
}

type tableSchemaInfo struct {
	ID        string
	ColumnIDs []string
}

func tableSchemaMetadata(engineInstance any) (tableSchemaInfo, bool) {
	value := reflect.ValueOf(engineInstance)
	method := value.MethodByName("Schema")
	if !method.IsValid() || method.Type().NumIn() != 0 || method.Type().NumOut() != 1 {
		return tableSchemaInfo{}, false
	}
	out := method.Call(nil)
	if len(out) != 1 || out[0].IsNil() {
		return tableSchemaInfo{}, false
	}
	schema := out[0]
	if schema.Kind() == reflect.Pointer {
		schema = schema.Elem()
	}
	if schema.Kind() != reflect.Struct {
		return tableSchemaInfo{}, false
	}
	idField := schema.FieldByName("ID")
	columnsField := schema.FieldByName("Columns")
	if !idField.IsValid() || idField.Kind() != reflect.String || !columnsField.IsValid() || columnsField.Kind() != reflect.Slice {
		return tableSchemaInfo{}, false
	}
	info := tableSchemaInfo{ID: idField.String()}
	for i := 0; i < columnsField.Len(); i++ {
		column := columnsField.Index(i)
		if column.Kind() == reflect.Pointer {
			column = column.Elem()
		}
		if column.Kind() != reflect.Struct {
			continue
		}
		columnID := column.FieldByName("ID")
		if columnID.IsValid() && columnID.Kind() == reflect.String {
			info.ColumnIDs = append(info.ColumnIDs, columnID.String())
		}
	}
	return info, true
}

func CollectPageGroups(registry *manifest.Manifest) ([]AccessGroup, error) {
	groups := make([]AccessGroup, 0, len(registry.All()))
	seen := map[string]struct{}{}
	for _, entry := range registry.All() {
		code := PageAccessGroupCode(entry.Key)
		if _, exists := seen[code]; exists {
			return nil, fmt.Errorf("access collector: duplicate access group %q", code)
		}
		seen[code] = struct{}{}
		groups = append(groups, AccessGroup{
			Code:        code,
			Name:        "Page " + entry.Key,
			Description: "Page " + entry.Key,
			Type:        AccessGroupPage,
			Enabled:     true,
		})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Code < groups[j].Code })
	return groups, nil
}

func CollectElementBindings(registry *manifest.Manifest) ([]ElementBinding, error) {
	var bindings []ElementBinding
	for _, entry := range registry.All() {
		page := entry.Factory()
		engineInstance := page.GetEngine()
		_ = engineInstance.Routes(entry.Key, page)
		provider, ok := engineInstance.(interface{ AccessElements() []ElementBinding })
		if !ok {
			continue
		}
		for _, binding := range provider.AccessElements() {
			if binding.Page == "" {
				binding.Page = entry.Key
			}
			bindings = append(bindings, binding)
		}
	}
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].GroupCode == bindings[j].GroupCode {
			return bindings[i].Element.Code < bindings[j].Element.Code
		}
		return bindings[i].GroupCode < bindings[j].GroupCode
	})
	return bindings, nil
}

func MergeAccessGroupElements(groups []AccessGroup, bindings []ElementBinding) ([]AccessGroup, error) {
	result := make([]AccessGroup, len(groups))
	copy(result, groups)
	index := make(map[string]int, len(result))
	for i, group := range result {
		index[group.Code] = i
	}
	for _, binding := range bindings {
		groupIndex, ok := index[binding.GroupCode]
		if !ok {
			return nil, fmt.Errorf("unknown access group %q referenced by element %q", binding.GroupCode, binding.Element.Code)
		}
		element := binding.Element
		if strings.TrimSpace(element.Code) == "" {
			return nil, fmt.Errorf("access group %q has element with empty code", binding.GroupCode)
		}
		if element.NoAccessBehavior == "" {
			element.NoAccessBehavior = NoAccessHidden
		}
		if element.ElementType == "" {
			element.ElementType = ElementCustom
		}
		result[groupIndex].Elements = appendOrReplaceElement(result[groupIndex].Elements, element)
	}
	for i := range result {
		sort.Slice(result[i].Elements, func(j, k int) bool {
			return result[i].Elements[j].Code < result[i].Elements[k].Code
		})
	}
	return result, nil
}

func appendOrReplaceElement(elements []AccessElement, element AccessElement) []AccessElement {
	for i, existing := range elements {
		if existing.Code == element.Code {
			elements[i] = element
			return elements
		}
	}
	return append(elements, element)
}

func PageAccessGroupCode(pageKey string) string {
	return Key("", "page", pageKey)
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

func GenerateAccess(path, module string, resources []Resource, groups []AccessGroup) (Manifest, error) {
	current := Manifest{}
	if existing, err := Read(path); err == nil {
		current = existing
	} else if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, err
	}
	merged := Merge(current, module, resources)
	merged = MergeAccessGroups(merged, groups)
	merged.Module = module
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

func MergeAccessGroups(current Manifest, discovered []AccessGroup) Manifest {
	result := current
	if result.Version == 0 {
		result.Version = CurrentVersion
	}

	oldGroups := make(map[string]AccessGroup, len(current.AccessGroups))
	for _, item := range current.AccessGroups {
		oldGroups[item.Code] = item
	}
	discoveredCodes := make(map[string]struct{}, len(discovered))
	result.AccessGroups = make([]AccessGroup, 0, len(discovered))
	for _, item := range discovered {
		discoveredCodes[item.Code] = struct{}{}
		if old, ok := oldGroups[item.Code]; ok {
			item = preserveAccessGroupMetadata(item, old)
		}
		result.AccessGroups = append(result.AccessGroups, item)
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

func preserveAccessGroupMetadata(discovered, old AccessGroup) AccessGroup {
	if old.Name != "" {
		discovered.Name = old.Name
	}
	if old.Description != "" {
		discovered.Description = old.Description
	}
	if old.ParentCode != "" {
		discovered.ParentCode = old.ParentCode
	}
	if len(old.Elements) > 0 {
		discovered.Elements = old.Elements
	}
	discovered.Enabled = old.Enabled
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
	accessGroupCodes := map[string]struct{}{}
	for _, group := range value.AccessGroups {
		if strings.TrimSpace(group.Code) == "" {
			problems = append(problems, "access group code must not be empty")
			continue
		}
		if _, exists := accessGroupCodes[group.Code]; exists {
			problems = append(problems, "duplicate access group: "+group.Code)
		}
		accessGroupCodes[group.Code] = struct{}{}
		if group.Type == "" {
			problems = append(problems, fmt.Sprintf("access group %q type must not be empty", group.Code))
		}
		if group.ParentCode != "" {
			if _, exists := accessGroupCodes[group.ParentCode]; !exists && !containsAccessGroup(value.AccessGroups, group.ParentCode) {
				problems = append(problems, fmt.Sprintf("access group %q references unknown parent %q", group.Code, group.ParentCode))
			}
		}
		elementCodes := map[string]struct{}{}
		for _, element := range group.Elements {
			if strings.TrimSpace(element.Code) == "" {
				problems = append(problems, fmt.Sprintf("access group %q has element with empty code", group.Code))
				continue
			}
			if _, exists := elementCodes[element.Code]; exists {
				problems = append(problems, fmt.Sprintf("access group %q has duplicate element %q", group.Code, element.Code))
			}
			elementCodes[element.Code] = struct{}{}
			if element.ElementType == "" {
				problems = append(problems, fmt.Sprintf("access element %q type must not be empty", element.Code))
			}
			if element.NoAccessBehavior == "" {
				problems = append(problems, fmt.Sprintf("access element %q noAccessBehavior must not be empty", element.Code))
			}
		}
	}
	groupKeys := map[string]struct{}{}
	for _, group := range value.PermissionGroups {
		groupCode := permissionGroupCode(group)
		if strings.TrimSpace(groupCode) == "" {
			problems = append(problems, "permission group key must not be empty")
		} else if _, exists := groupKeys[groupCode]; exists {
			problems = append(problems, "duplicate permission group key: "+groupCode)
		}
		groupKeys[groupCode] = struct{}{}
		for _, accessGroup := range permissionGroupAccessGroups(group) {
			if !matchesAny(accessGroup, accessGroupCodes) {
				problems = append(problems, fmt.Sprintf("permission group %q references unknown access group %q", groupCode, accessGroup))
			}
		}
		for _, permission := range group.Permissions {
			if matchesAny(permission, staleKeys) {
				problems = append(problems, fmt.Sprintf("group %q uses stale permission %q", groupCode, permission))
				continue
			}
			if !matchesAny(permission, resourceKeys) {
				problems = append(problems, fmt.Sprintf("group %q references unknown permission %q", groupCode, permission))
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return errors.New(strings.Join(problems, "\n"))
	}
	return nil
}

func containsAccessGroup(groups []AccessGroup, code string) bool {
	for _, group := range groups {
		if group.Code == code {
			return true
		}
	}
	return false
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
		groupCode := permissionGroupCode(group)
		diff.ExistingGroups = append(diff.ExistingGroups, groupCode)
		for _, permission := range group.Permissions {
			if matchesAny(permission, stale) || !matchesAny(permission, active) || !matchesAny(permission, dsl) {
				diff.BrokenGroupPermissions = append(diff.BrokenGroupPermissions, groupCode+": "+permission)
			}
		}
	}
	sortStringsInDiff(&diff)
	return diff
}

func permissionGroupCode(group PermissionGroup) string {
	if group.Code != "" {
		return group.Code
	}
	return group.Key
}

func permissionGroupAccessGroups(group PermissionGroup) []string {
	if len(group.AccessGroups) > 0 {
		return group.AccessGroups
	}
	return nil
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
	sort.Slice(value.AccessGroups, func(i, j int) bool { return value.AccessGroups[i].Code < value.AccessGroups[j].Code })
	sort.Slice(value.Resources, func(i, j int) bool { return value.Resources[i].Key < value.Resources[j].Key })
	sort.Slice(value.PermissionGroups, func(i, j int) bool {
		return permissionGroupCode(value.PermissionGroups[i]) < permissionGroupCode(value.PermissionGroups[j])
	})
	sort.Slice(value.Stale, func(i, j int) bool { return value.Stale[i].Key < value.Stale[j].Key })
	for i := range value.AccessGroups {
		sort.Slice(value.AccessGroups[i].Elements, func(j, k int) bool {
			return value.AccessGroups[i].Elements[j].Code < value.AccessGroups[i].Elements[k].Code
		})
	}
	for i := range value.PermissionGroups {
		sort.Strings(value.PermissionGroups[i].AccessGroups)
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
