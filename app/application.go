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
	"github.com/gin-gonic/gin"
)

// InitFunc — функция инициализации проекта.
// Вызывается один раз при старте: регистрирует pages в манифесте.
type InitFunc func(app *Application)

// Application — центральный orchestrator framework.
// Хранит manifest, запускает bootstrap, регистрирует routes в Gin.
// НЕ знает о DSL, UI логике, бизнес-логике.
type Application struct {
	manifest    *manifest.Manifest
	router      *gin.Engine
	config      Config
	syncer      access.AccessSyncProvider
	access      *access.Registry
	instances   *pageInstanceManager
	initialized bool
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
	// Authenticator enables Bearer authentication for all page, event and
	// instance lifecycle routes. Nil preserves the legacy unauthenticated mode.
	Authenticator    authentication.Authenticator
	AccessAuthorizer access.AccessAuthorizer
}

// New создаёт новый Application.
func New(config ...Config) *Application {
	a := &Application{
		manifest: manifest.New(),
		access:   access.NewRegistry(),
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

	// Шаг 2: auto route generation из manifest
	a.registerRoutes()

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
func (a *Application) registerRoutes() {
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
		a.router.DELETE(
			engine.RoutePath(a.config.Module, "/page/"+entry.Key+"/instance"),
			a.deletePageInstance(entry.Key),
		)
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
