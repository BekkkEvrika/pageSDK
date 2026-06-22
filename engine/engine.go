package engine

import "strings"

// Engine — интерфейс, который должен реализовывать каждый движок.
// Engine хранит DSL/runtime state только внутри конкретного per-request instance.
// Он предоставляет runtime strategy: routes, DSL rendering и event handling.
type Engine interface {
	// ID возвращает стабильный identifier движка.
	ID() string

	// Routes возвращает все routes, нужные движку для конкретной page.
	// Application не знает routing semantics движка и только регистрирует результат в Gin.
	Routes(pageKey string, page Page) []RouteDefinition

	// Render создаёт DSL/runtime response для page.
	Render(ctx *RequestContext, page Page) (*RenderResult, error)

	// Handle обрабатывает runtime events для page.
	Handle(ctx *RequestContext, page Page) (*RuntimeResult, error)
}

// RequestContext — transport-neutral snapshot of an incoming runtime request.
// Application builds it from Gin, but Page and Engine do not receive Gin directly.
type RequestContext struct {
	PageKey string
	Module  string
	Params  Params
	Query   Params
	User    User
	System  SystemKeys
	Body    []byte
}

// RouteDefinition — route, предоставленный конкретным Engine.
type RouteDefinition struct {
	Method  string
	Path    string
	Handler RouteHandler
}

// RouteHandler — runtime handler route, который Application вызывает
// со свежим Page instance на каждый request.
type RouteHandler func(ctx *RequestContext, page Page) (any, error)

// RoutePath adds an optional module prefix to a framework-owned route.
func RoutePath(module, path string) string {
	module = strings.Trim(module, "/ ")
	if module == "" {
		return path
	}
	return "/" + module + "/" + strings.TrimPrefix(path, "/")
}

// RenderResult is returned for DSL requests.
type RenderResult struct {
	PageKey string `json:"pageKey"`
	Engine  string `json:"engine"`
	DSL     any    `json:"dsl"`
}

// RuntimeResult is returned for event requests.
type RuntimeResult struct {
	Mutations  []Mutation         `json:"mutations,omitempty"`
	Navigation []NavigationAction `json:"navigation,omitempty"`
	Dialogs    []Dialog           `json:"dialogs,omitempty"`
	Result     any                `json:"result,omitempty"`
}
