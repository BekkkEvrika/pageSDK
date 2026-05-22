package app

import (
	"net/http"

	"github.com/behzod/pageSDK/engine"
	"github.com/behzod/pageSDK/manifest"
	"github.com/gin-gonic/gin"
)

// InitFunc — функция инициализации проекта.
// Вызывается один раз при старте: регистрирует pages в манифесте.
type InitFunc func(app *Application)

// Application — центральный orchestrator framework.
// Хранит manifest, запускает bootstrap, регистрирует routes в Gin.
// НЕ знает о DSL, UI логике, бизнес-логике.
type Application struct {
	manifest *manifest.Manifest
	router   *gin.Engine
}

// New создаёт новый Application.
func New() *Application {
	return &Application{
		manifest: manifest.New(),
		router:   gin.New(),
	}
}

// Manifest возвращает манифест приложения.
// Используется в InitFunc для регистрации pages.
func (a *Application) Manifest() *manifest.Manifest {
	return a.manifest
}

// Bootstrap запускает lifecycle:
// 1. Вызывает initFn — проект заполняет manifest.
// 2. Генерирует routes для всех pages из manifest.
// 3. Запускает Gin на указанном адресе.
func (a *Application) Bootstrap(initFn InitFunc, addr string) error {
	// Шаг 1: проект регистрирует свои pages
	initFn(a)

	// Шаг 2: auto route generation из manifest
	a.registerRoutes()

	// Шаг 3: запуск HTTP сервера
	return a.router.Run(addr)
}

// registerRoutes итерирует manifest и делегирует регистрацию routes
// конкретному Engine каждой page. Application регистрирует только Method/Path/Handler.
func (a *Application) registerRoutes() {
	for _, entry := range a.manifest.All() {
		entry := entry // capture

		// Создаём временный экземпляр page только для получения Engine.
		// Сам page не используется для обработки request — только для метаданных.
		samplePage := entry.Factory()
		eng := samplePage.GetEngine()

		// Движок знает routing semantics (form, table, etc.) и возвращает routes.
		for _, route := range eng.Routes(entry.Key, samplePage) {
			a.registerRoute(entry, route)
		}
	}
}

// registerRoute регистрирует RouteDefinition в Gin.
func (a *Application) registerRoute(entry manifest.Entry, route engine.RouteDefinition) {
	a.router.Handle(route.Method, route.Path, a.makeGinHandler(entry, route.Handler))
}

// makeGinHandler возвращает gin.HandlerFunc для конкретной page entry.
// На каждый request: создаёт новый Page, вызывает runtime handler, уничтожает.
func (a *Application) makeGinHandler(entry manifest.Entry, handler engine.RouteHandler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		page := entry.Factory()
		result, err := handler(a.newRequestContext(ctx, entry.Key), page)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if result != nil && !ctx.Writer.Written() {
			ctx.JSON(http.StatusOK, result)
		}
	}
}

func (a *Application) newRequestContext(ctx *gin.Context, pageKey string) *engine.RequestContext {
	return &engine.RequestContext{
		PageKey: pageKey,
		Params:  routeParams(ctx),
		Query:   queryParams(ctx),
		User:    engine.User{},
		System:  engine.SystemKeys{},
	}
}

func routeParams(ctx *gin.Context) engine.Params {
	params := make(engine.Params, len(ctx.Params))
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
