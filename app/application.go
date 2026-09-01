package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/BekkkEvrika/pageSDK/access"
	"github.com/BekkkEvrika/pageSDK/authentication"
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/logging"
	sdklog "github.com/BekkkEvrika/pageSDK/logging/log"
	"github.com/BekkkEvrika/pageSDK/manifest"
	"github.com/BekkkEvrika/pageSDK/sidebar"
	"github.com/gin-gonic/gin"
)

// InitFunc — функция инициализации проекта.
// Вызывается один раз при старте: регистрирует pages в манифесте.
type InitFunc func(app *Application)

// Application — центральный orchestrator framework.
// Хранит manifest, запускает bootstrap, регистрирует routes в Gin.
// НЕ знает о DSL, UI логике, бизнес-логике.
type Application struct {
	manifest     *manifest.Manifest
	router       *gin.Engine
	customRoutes []Route
	config       Config
	setupErr     error
	syncer       access.AccessSyncProvider
	access       *access.Registry
	sidebar      *sidebar.Registry
	instances    *pageInstanceManager
	initialized  bool
}

const principalContextKey = "pagesdk.principal"

// Route describes an application-owned Gin route protected by a registered
// access group. Custom routes are registered before the HTTP server starts.
type Route struct {
	Method      string
	Path        string
	AccessGroup access.AccessGroup
	Handler     gin.HandlerFunc
}

type Config struct {
	Module              string
	KeycloakURL         string
	Realm               string
	ClientID            string
	ClientSecret        string
	KeycloakSyncEnabled bool
	AccessManifestPath  string
	PageInstanceTTL     time.Duration
	MaxPageInstances    int
	AccessCacheTTL      time.Duration
	Sidebar             sidebar.Config
	// Authenticator enables Bearer authentication for all page, event and
	// instance lifecycle routes. Nil preserves the legacy unauthenticated mode.
	Authenticator    authentication.Authenticator
	AccessAuthorizer access.AccessAuthorizer
}

// SetupFunc declaratively registers access groups, custom routes, and other
// application-owned configuration while the Application is being created.
type SetupFunc func(app *Application) error

// New создаёт новый Application.
func New(config ...Config) *Application {
	a := &Application{
		manifest: manifest.New(),
		access:   access.NewRegistry(),
		sidebar:  sidebar.NewRegistry(),
	}
	if len(config) > 0 {
		a.config = config[0]
	}
	a.applyEnvConfig()
	if a.config.AccessManifestPath == "" {
		a.config.AccessManifestPath = "sfp.access.yaml"
	}
	a.instances = newPageInstanceManager(a.config.PageInstanceTTL, a.config.MaxPageInstances)
	a.syncer = access.NewKeycloakUMAProvider(a.accessConfig())
	return a
}

// Configure executes setup callbacks in declaration order. The first error is
// deferred and returned by Run, Bootstrap, or an access command.
func (a *Application) Configure(setups ...SetupFunc) {
	if a == nil || a.setupErr != nil {
		return
	}
	for _, setup := range setups {
		if setup == nil {
			continue
		}
		if err := setup(a); err != nil {
			a.setupErr = fmt.Errorf("application setup: %w", err)
			return
		}
	}
}

func (a *Application) applyEnvConfig() {
	if a.config.KeycloakURL == "" {
		a.config.KeycloakURL = os.Getenv("KEYCLOAK_BASE_URL")
	}
	if a.config.Realm == "" {
		a.config.Realm = os.Getenv("KEYCLOAK_REALM")
	}
	if a.config.ClientID == "" {
		a.config.ClientID = os.Getenv("KEYCLOAK_CLIENT_ID")
	}
	if a.config.ClientSecret == "" {
		a.config.ClientSecret = os.Getenv("KEYCLOAK_CLIENT_SECRET")
	}
	if !a.config.KeycloakSyncEnabled {
		a.config.KeycloakSyncEnabled = strings.EqualFold(os.Getenv("KEYCLOAK_SYNC_ENABLED"), "true")
	}
}

// Manifest возвращает манифест приложения.
// Используется в InitFunc для регистрации pages.
func (a *Application) Manifest() *manifest.Manifest {
	return a.manifest
}

func (a *Application) Config() Config {
	return a.config
}

func (a *Application) AccessRegistry() *access.Registry {
	return a.access
}

func (a *Application) RegisterAccessGroup(group access.AccessGroup) error {
	return a.access.Register(group)
}

func (a *Application) SidebarRegistry() *sidebar.Registry {
	return a.sidebar
}

// RegisterSidebarNode adds one declarative sidebar node to the central registry.
func (a *Application) RegisterSidebarNode(node sidebar.Node) error {
	return a.sidebar.Register(node)
}

// RegisterSidebarNodes adds a batch atomically.
func (a *Application) RegisterSidebarNodes(nodes ...sidebar.Node) error {
	return a.sidebar.RegisterMany(nodes...)
}

// SetSidebarPublisher replaces the transport adapter from Config.Sidebar.
func (a *Application) SetSidebarPublisher(publisher sidebar.Publisher) {
	a.config.Sidebar.Publisher = publisher
}

// PublishSidebar validates and publishes one full sidebar operation. Page
// targets are derived from the currently registered manifest.
func (a *Application) PublishSidebar(ctx context.Context, action sidebar.Action) error {
	return a.publishSidebar(ctx, action, nil)
}

func (a *Application) publishSidebar(ctx context.Context, action sidebar.Action, bindings []sidebar.Binding) error {
	if ctx == nil {
		ctx = context.Background()
	}
	publisher := a.config.Sidebar.Publisher
	if publisher == nil {
		return errors.New("sidebar publisher is not configured")
	}

	var (
		event sidebar.Event
		err   error
	)
	switch action {
	case sidebar.ActionRegistration, sidebar.ActionRefresh:
		if bindings == nil {
			bindings = a.collectSidebarBindings()
		}
		event, err = a.sidebar.BuildEvent(a.config.Sidebar, bindings, a.access)
	case sidebar.ActionUnregister:
		event, err = sidebar.UnregisterEvent(a.config.Sidebar.ServiceID)
	default:
		return fmt.Errorf("unknown sidebar action %q", action)
	}
	if err != nil {
		return err
	}
	if err := publisher.PublishSidebar(ctx, action, event); err != nil {
		return fmt.Errorf("publish sidebar %s: %w", action, err)
	}
	return nil
}

func (a *Application) collectSidebarBindings() []sidebar.Binding {
	var result []sidebar.Binding
	for _, entry := range a.manifest.All() {
		page := entry.Factory()
		engineInstance := page.GetEngine()
		_ = engineInstance.Routes(entry.Key, page)
		provider, ok := engineInstance.(interface{ SidebarBindings() []sidebar.Binding })
		if !ok {
			continue
		}
		for _, binding := range provider.SidebarBindings() {
			binding.PageKey = entry.Key
			binding.Target = engine.RoutePath(a.config.Module, "/page/"+entry.Key)
			result = append(result, binding)
		}
	}
	return result
}

// RegisterRoute registers an application-owned route protected by JWT
// authentication and UMA authorization through an existing access group.
func (a *Application) RegisterRoute(route Route) error {
	if a.initialized {
		return errors.New("register route: application is already initialized")
	}
	route.Method = strings.ToUpper(strings.TrimSpace(route.Method))
	route.Path = strings.TrimSpace(route.Path)
	if route.Method == "" {
		return errors.New("register route: HTTP method is required")
	}
	if strings.ContainsAny(route.Method, " \t\r\n") {
		return fmt.Errorf("register route %q: invalid HTTP method %q", route.Path, route.Method)
	}
	if route.Path == "" || !strings.HasPrefix(route.Path, "/") || strings.ContainsAny(route.Path, "?#") {
		return fmt.Errorf("register route %s: path must start with / and must not contain query or fragment", route.Method)
	}
	if reservedRoutePath(route.Path) {
		return fmt.Errorf("register route %s %s: /page and /event are reserved by pageSDK", route.Method, route.Path)
	}
	if route.Handler == nil {
		return fmt.Errorf("register route %s %s: handler is required", route.Method, route.Path)
	}
	group, ok := a.access.Get(route.AccessGroup.Code)
	if !ok {
		return fmt.Errorf("register route %s %s: access group %q is not registered", route.Method, route.Path, route.AccessGroup.Code)
	}
	route.AccessGroup = group
	for _, existing := range a.customRoutes {
		if existing.Method == route.Method && existing.Path == route.Path {
			return fmt.Errorf("register route: duplicate route %s %s", route.Method, route.Path)
		}
	}
	a.customRoutes = append(a.customRoutes, route)
	return nil
}

// RegisterRoutes registers application-owned routes as a single batch. If any
// route is invalid, none of the routes from the batch remain registered.
func (a *Application) RegisterRoutes(routes ...Route) error {
	registered := len(a.customRoutes)
	for _, route := range routes {
		if err := a.RegisterRoute(route); err != nil {
			a.customRoutes = a.customRoutes[:registered]
			return err
		}
	}
	return nil
}

func reservedRoutePath(path string) bool {
	path = "/" + strings.Trim(strings.TrimSpace(path), "/")
	return path == "/page" || strings.HasPrefix(path, "/page/") ||
		path == "/event" || strings.HasPrefix(path, "/event/")
}

// PrincipalFromContext returns the identity verified for a custom route.
func PrincipalFromContext(ctx *gin.Context) (authentication.Principal, bool) {
	if ctx == nil {
		return authentication.Principal{}, false
	}
	value, ok := ctx.Get(principalContextKey)
	if !ok {
		return authentication.Principal{}, false
	}
	principal, ok := value.(authentication.Principal)
	return principal, ok
}

func (a *Application) SetAccessSyncProvider(provider access.AccessSyncProvider) {
	if provider != nil {
		a.syncer = provider
	}
}

func (a *Application) SetAuthenticator(authenticator authentication.Authenticator) {
	a.config.Authenticator = authenticator
}

func (a *Application) SetAccessAuthorizer(authorizer access.AccessAuthorizer) {
	a.config.AccessAuthorizer = authorizer
}

func (a *Application) UseRPTAccessAuthorizer(ttl ...time.Duration) {
	cacheTTL := a.config.AccessCacheTTL
	if len(ttl) > 0 {
		cacheTTL = ttl[0]
	}
	if a.config.KeycloakURL != "" && a.config.Realm != "" && a.config.ClientID != "" {
		config := a.accessConfig()
		config.CacheTTL = cacheTTL
		a.config.AccessAuthorizer = access.NewKeycloakUMAAccessAuthorizer(config)
		return
	}
	a.config.AccessAuthorizer = access.NewCachedAuthorizer(access.RPTClaimSource{}, cacheTTL)
}

// Bootstrap запускает lifecycle:
// 1. Вызывает initFn — проект заполняет manifest.
// 2. Генерирует routes для всех pages из manifest.
// 3. Запускает Gin на указанном адресе.
func (a *Application) Bootstrap(initFn InitFunc, addr string) error {
	a.initialize(initFn)
	if a.setupErr != nil {
		return a.setupErr
	}
	if err := a.validateCustomRouteSecurity(); err != nil {
		return err
	}

	// Шаг 2: auto route generation и сбор sidebar bindings из sample pages.
	bindings := a.registerRoutes()
	if a.sidebar.Len() > 0 || len(bindings) > 0 {
		if err := a.publishSidebar(context.Background(), sidebar.ActionRegistration, bindings); err != nil {
			return err
		}
	}

	// Шаг 3: запуск HTTP сервера
	return a.router.Run(addr)
}

// Run dispatches CLI commands and keeps "no arguments" compatible with the
// historical HTTP-server entrypoint.
func (a *Application) Run(initFn InitFunc, addr string) error {
	return a.Execute(context.Background(), initFn, addr, os.Args[1:], os.Stdout)
}

func (a *Application) Execute(ctx context.Context, initFn InitFunc, addr string, args []string, output io.Writer) error {
	if len(args) == 0 || args[0] == "serve" {
		return a.Bootstrap(initFn, addr)
	}
	if args[0] != "access" {
		return fmt.Errorf("unknown command %q (expected serve or access)", args[0])
	}
	if len(args) < 2 {
		return errors.New("access command requires one of: generate, validate, diff, sync")
	}
	a.initialize(initFn)
	if a.setupErr != nil {
		return a.setupErr
	}
	path := a.config.AccessManifestPath
	switch args[1] {
	case "generate":
		resources, err := access.Collect(a.manifest, a.config.Module)
		if err != nil {
			return err
		}
		pageGroups, err := access.CollectPageGroups(a.manifest)
		if err != nil {
			return err
		}
		accessGroups := append(pageGroups, a.access.All()...)
		bindings, err := access.CollectElementBindings(a.manifest)
		if err != nil {
			return err
		}
		accessGroups, err = access.MergeAccessGroupElements(accessGroups, bindings)
		if err != nil {
			return err
		}
		generated, err := access.GenerateAccess(path, a.config.Module, resources, accessGroups)
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "generated %s (%d access groups, %d resources, %d stale)\n",
			path, len(generated.AccessGroups), len(generated.Resources), len(generated.Stale))
		return nil
	case "validate":
		value, err := access.Read(path)
		if err != nil {
			return err
		}
		if err := access.Validate(value, a.config.Module); err != nil {
			return err
		}
		fmt.Fprintf(output, "%s is valid\n", path)
		return nil
	case "diff":
		resources, err := access.Collect(a.manifest, a.config.Module)
		if err != nil {
			return err
		}
		value, err := access.Read(path)
		if err != nil {
			return err
		}
		printDiff(output, access.Compare(resources, value))
		return nil
	case "sync":
		flags := flag.NewFlagSet("access sync", flag.ContinueOnError)
		flags.SetOutput(output)
		dryRun := flags.Bool("dry-run", false, "print and validate the local sync plan without changing Keycloak")
		if err := flags.Parse(args[2:]); err != nil {
			return err
		}
		value, err := access.Read(path)
		if err != nil {
			return err
		}
		if err := access.Validate(value, a.config.Module); err != nil {
			return err
		}
		if *dryRun {
			fmt.Fprintf(output, "dry-run: would sync %d access groups\n", len(value.AccessGroups))
		}
		if !*dryRun && missingKeycloakConfig(a.config) != "" {
			return fmt.Errorf("access sync: missing Keycloak config: %s", missingKeycloakConfig(a.config))
		}
		return a.syncer.Sync(ctx, value, access.SyncOptions{DryRun: *dryRun})
	default:
		return fmt.Errorf("unknown access command %q", args[1])
	}
}

func (a *Application) initialize(initFn InitFunc) {
	if a.initialized {
		return
	}
	initFn(a)
	a.initialized = true
}

func (a *Application) accessConfig() access.Config {
	return access.Config{
		Module:       a.config.Module,
		ManifestPath: a.config.AccessManifestPath,
		KeycloakURL:  a.config.KeycloakURL,
		Realm:        a.config.Realm,
		ClientID:     a.config.ClientID,
		ClientSecret: a.config.ClientSecret,
		SyncEnabled:  a.config.KeycloakSyncEnabled,
		CacheTTL:     a.config.AccessCacheTTL,
	}
}

func missingKeycloakConfig(config Config) string {
	var missing []string
	if config.KeycloakURL == "" {
		missing = append(missing, "KeycloakURL")
	}
	if config.Realm == "" {
		missing = append(missing, "Realm")
	}
	if config.ClientID == "" {
		missing = append(missing, "ClientID")
	}
	if config.ClientSecret == "" {
		missing = append(missing, "ClientSecret")
	}
	return strings.Join(missing, ", ")
}

func printDiff(output io.Writer, diff access.Diff) {
	printDiffSection(output, "New in DSL", diff.NewInDSL)
	printDiffSection(output, "Missing in DSL / stale", diff.MissingInDSL)
	printDiffSection(output, "Missing in manifest", diff.MissingInManifest)
	printDiffSection(output, "Existing groups", diff.ExistingGroups)
	printDiffSection(output, "Broken group permissions", diff.BrokenGroupPermissions)
}

func printDiffSection(output io.Writer, title string, values []string) {
	fmt.Fprintln(output, title+":")
	if len(values) == 0 {
		fmt.Fprintln(output, "  (none)")
		return
	}
	for _, value := range values {
		fmt.Fprintln(output, "  - "+value)
	}
}

// registerRoutes итерирует manifest и получает route metadata из sample Engine.
// Render creates a Page instance; events reuse that stored Page.
func (a *Application) registerRoutes() []sidebar.Binding {
	var bindings []sidebar.Binding
	a.ensureRouter()
	a.router.Use(logging.LogMiddleware)
	for _, entry := range a.manifest.All() {
		entry := entry // capture

		// Создаём временный экземпляр page только для получения route metadata.
		// Сам page и его Engine не используются для обработки request.
		samplePage := entry.Factory()
		eng := samplePage.GetEngine()

		// Движок знает routing semantics (form, table, etc.) и возвращает routes.
		for _, route := range eng.Routes(entry.Key, samplePage) {
			a.registerRoute(entry, route)
		}
		if provider, ok := eng.(interface{ SidebarBindings() []sidebar.Binding }); ok {
			for _, binding := range provider.SidebarBindings() {
				binding.PageKey = entry.Key
				binding.Target = engine.RoutePath(a.config.Module, "/page/"+entry.Key)
				bindings = append(bindings, binding)
			}
		}
		a.router.DELETE(
			engine.RoutePath(a.config.Module, "/page/"+entry.Key+"/instance"),
			a.deletePageInstance(entry.Key),
		)
	}
	for _, route := range a.customRoutes {
		a.router.Handle(route.Method, engine.RoutePath(a.config.Module, route.Path), a.makeCustomRouteHandler(route))
	}
	return bindings
}

func (a *Application) validateCustomRouteSecurity() error {
	if len(a.customRoutes) == 0 {
		return nil
	}
	if a.config.Authenticator == nil {
		return errors.New("custom routes require an Authenticator")
	}
	if a.config.AccessAuthorizer == nil {
		return errors.New("custom routes require an AccessAuthorizer")
	}
	return nil
}

func (a *Application) makeCustomRouteHandler(route Route) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		principal, ok := a.authenticate(ctx)
		if !ok {
			return
		}
		ctx.Set(principalContextKey, principal)
		if !a.authorizeAccessGroup(ctx, principal, route.AccessGroup.Code) {
			return
		}
		route.Handler(ctx)
	}
}

func (a *Application) ensureRouter() {
	if a.router == nil {
		a.router = gin.New()
	}
}

// registerRoute регистрирует RouteDefinition в Gin.
func (a *Application) registerRoute(entry manifest.Entry, route engine.RouteDefinition) {
	a.router.Handle(route.Method, engine.RoutePath(a.config.Module, route.Path), a.makeGinHandler(entry, route))
}

// makeGinHandler creates a Page on render and reuses the stored instance for
// subsequent events identified by the pageInstanceId query parameter.
func (a *Application) makeGinHandler(entry manifest.Entry, route engine.RouteDefinition) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		principal, ok := a.authenticate(ctx)
		if !ok {
			return
		}
		requestContext := a.newRequestContext(ctx, entry.Key, principal.User)
		var (
			page     engine.Page
			instance *pageInstance
		)

		switch route.Mode {
		case engine.RouteModeRender:
			if !a.authorizeAccessGroup(ctx, principal, access.PageAccessGroupCode(entry.Key)) {
				return
			}
			page = entry.Factory()
			instanceID, err := a.instances.NewID()
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			requestContext.PageInstanceID = instanceID
		case engine.RouteModeEvent:
			instanceID := requestContext.Query[engine.PageInstanceParam]
			if instanceID == "" {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": engine.PageInstanceParam + " is required"})
				return
			}
			requestContext.PageInstanceID = instanceID
			var err error
			instance, err = a.instances.Acquire(instanceID, entry.Key, principal.ID)
			if err != nil {
				status := http.StatusNotFound
				if errors.Is(err, ErrPageInstanceExpired) {
					status = http.StatusGone
				}
				ctx.JSON(status, gin.H{"error": err.Error()})
				return
			}
			defer a.instances.Release(instance)
			page = instance.Page
			if route.AccessGroupCode != "" && !a.authorizeAccessGroup(ctx, principal, route.AccessGroupCode) {
				return
			}
		default:
			page = entry.Factory()
		}

		result, err := route.Handler(requestContext, page)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if route.Mode == engine.RouteModeRender {
			if err := a.applyDSLAccess(ctx, principal, result); err != nil {
				ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "access check failed"})
				return
			}
		}
		if route.Mode == engine.RouteModeRender {
			if err := a.instances.Add(requestContext.PageInstanceID, entry.Key, principal.ID, page); err != nil {
				ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
				return
			}
		}
		if result != nil && !ctx.Writer.Written() {
			ctx.JSON(http.StatusOK, result)
		}
	}
}

func (a *Application) applyDSLAccess(ctx *gin.Context, principal authentication.Principal, result any) error {
	render, ok := result.(*engine.RenderResult)
	if !ok || render == nil {
		return nil
	}
	requestContext := access.WithBearerToken(ctx.Request.Context(), principal.Token)
	filtered, err := (access.DSLPermissionResolver{Authorizer: a.config.AccessAuthorizer}).
		Apply(requestContext, principal.ID, principal.User, render.DSL)
	if err != nil {
		return err
	}
	render.DSL = filtered
	return nil
}

func (a *Application) authorizeAccessGroup(ctx *gin.Context, principal authentication.Principal, code string) bool {
	if a.config.AccessAuthorizer == nil {
		return true
	}
	requestContext := access.WithBearerToken(ctx.Request.Context(), principal.Token)
	allowed, err := a.config.AccessAuthorizer.HasAccess(requestContext, principal.ID, principal.User, code)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "access check failed"})
		return false
	}
	if !allowed {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "permission denied", "accessGroup": code})
		return false
	}
	return true
}

func (a *Application) deletePageInstance(pageKey string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		principal, ok := a.authenticate(ctx)
		if !ok {
			return
		}
		instanceID := ctx.Query(engine.PageInstanceParam)
		if instanceID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": engine.PageInstanceParam + " is required"})
			return
		}
		if !a.instances.Delete(instanceID, pageKey, principal.ID) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": ErrPageInstanceNotFound.Error()})
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

func (a *Application) authenticate(ctx *gin.Context) (authentication.Principal, bool) {
	if a.config.Authenticator == nil {
		return authentication.Principal{User: engine.User{}}, true
	}
	token, err := bearerToken(ctx.GetHeader("Authorization"))
	if err != nil {
		logAuthFailure(ctx, err)
		ctx.Header("WWW-Authenticate", "Bearer")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return authentication.Principal{}, false
	}
	principal, err := a.config.Authenticator.Authenticate(ctx.Request.Context(), token)
	if err != nil {
		logAuthFailure(ctx, err)
		ctx.Header("WWW-Authenticate", "Bearer")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return authentication.Principal{}, false
	}
	if principal.ID == "" {
		logAuthFailure(ctx, errors.New("authenticator returned empty principal ID"))
		ctx.Header("WWW-Authenticate", "Bearer")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return authentication.Principal{}, false
	}
	if principal.User == nil {
		logAuthFailure(ctx, errors.New("authenticator returned nil user claims"))
		ctx.Header("WWW-Authenticate", "Bearer")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return authentication.Principal{}, false
	}
	principal.Token = token
	return principal, true
}

func logAuthFailure(ctx *gin.Context, err error) {
	if err == nil {
		return
	}
	sdklog.WriteLn("AUTH FAILED " + ctx.Request.Method + " " + ctx.Request.URL.String() + ": " + err.Error())
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("bearer token is required")
	}
	return parts[1], nil
}

func (a *Application) newRequestContext(ctx *gin.Context, pageKey string, user engine.User) *engine.RequestContext {
	body, _ := io.ReadAll(ctx.Request.Body)
	query := queryParams(ctx)
	return &engine.RequestContext{
		PageKey: pageKey,
		Module:  a.config.Module,
		Params:  requestParams(ctx, query),
		Query:   query,
		User:    user,
		System:  engine.SystemKeys{},
		Body:    body,
	}
}

func requestParams(ctx *gin.Context, query engine.Params) engine.Params {
	params := make(engine.Params, len(query)+len(ctx.Params))
	for key, value := range query {
		params[key] = value
	}
	for _, param := range ctx.Params {
		params[param.Key] = param.Value
	}
	return params
}

func queryParams(ctx *gin.Context) engine.Params {
	values := ctx.Request.URL.Query()
	params := make(engine.Params, len(values))
	for key := range values {
		params[key] = ctx.Query(key)
	}
	return params
}
