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
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/logging"
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
	instances   *pageInstanceManager
	initialized bool
}

type Config struct {
	Module             string
	KeycloakURL        string
	Realm              string
	ClientID           string
	ClientSecret       string
	AccessManifestPath string
	PageInstanceTTL    time.Duration
	MaxPageInstances   int
}

// New создаёт новый Application.
func New(config ...Config) *Application {
	a := &Application{
		manifest: manifest.New(),
		router:   gin.New(),
	}
	if len(config) > 0 {
		a.config = config[0]
	}
	if a.config.AccessManifestPath == "" {
		a.config.AccessManifestPath = "sfp.access.yaml"
	}
	a.instances = newPageInstanceManager(a.config.PageInstanceTTL, a.config.MaxPageInstances)
	a.syncer = access.UnsupportedKeycloakProvider{Config: a.accessConfig()}
	return a
}

// Manifest возвращает манифест приложения.
// Используется в InitFunc для регистрации pages.
func (a *Application) Manifest() *manifest.Manifest {
	return a.manifest
}

func (a *Application) Config() Config {
	return a.config
}

func (a *Application) SetAccessSyncProvider(provider access.AccessSyncProvider) {
	if provider != nil {
		a.syncer = provider
	}
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
		generated, err := access.Generate(path, a.config.Module, resources)
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "generated %s (%d resources, %d stale, %d groups)\n",
			path, len(generated.Resources), len(generated.Stale), len(generated.PermissionGroups))
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
			fmt.Fprintf(output, "dry-run: would sync %d resources and %d permission groups\n",
				len(value.Resources), len(value.PermissionGroups))
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

// registerRoute регистрирует RouteDefinition в Gin.
func (a *Application) registerRoute(entry manifest.Entry, route engine.RouteDefinition) {
	a.router.Handle(route.Method, engine.RoutePath(a.config.Module, route.Path), a.makeGinHandler(entry, route))
}

// makeGinHandler creates a Page on render and reuses the stored instance for
// subsequent events identified by the pageInstanceId query parameter.
func (a *Application) makeGinHandler(entry manifest.Entry, route engine.RouteDefinition) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestContext := a.newRequestContext(ctx, entry.Key)
		var (
			page     engine.Page
			instance *pageInstance
		)

		switch route.Mode {
		case engine.RouteModeRender:
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
			instance, err = a.instances.Acquire(instanceID, entry.Key)
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
		default:
			page = entry.Factory()
		}

		result, err := route.Handler(requestContext, page)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if route.Mode == engine.RouteModeRender {
			if err := a.instances.Add(requestContext.PageInstanceID, entry.Key, page); err != nil {
				ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
				return
			}
		}
		if result != nil && !ctx.Writer.Written() {
			ctx.JSON(http.StatusOK, result)
		}
	}
}

func (a *Application) deletePageInstance(pageKey string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		instanceID := ctx.Query(engine.PageInstanceParam)
		if instanceID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": engine.PageInstanceParam + " is required"})
			return
		}
		if !a.instances.Delete(instanceID, pageKey) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": ErrPageInstanceNotFound.Error()})
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

func (a *Application) newRequestContext(ctx *gin.Context, pageKey string) *engine.RequestContext {
	body, _ := io.ReadAll(ctx.Request.Body)
	query := queryParams(ctx)
	return &engine.RequestContext{
		PageKey: pageKey,
		Module:  a.config.Module,
		Params:  requestParams(ctx, query),
		Query:   query,
		User:    engine.User{},
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
