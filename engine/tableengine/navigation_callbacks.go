package tableengine

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unicode"

	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/table"
)

var navigationCallbackHandlers sync.Map

func registerNavigationCallback(pageKey string, handler table.NavigationCallback, module, instanceID string) string {
	name := navigationCallbackName(handler)
	navigationCallbackHandlers.Store(navigationCallbackKey(pageKey, instanceID, name), handler)
	return engine.PageInstanceURL(navigationCallbackRoutePath(module, pageKey, name), instanceID)
}

func (t *TableEngine) configureNavigationCallbacks(ctx *table.TableRuntimeContext, req *engine.RequestContext) {
	ctx.SetNavigationCallbackRegistrar(func(handler table.NavigationCallback) string {
		return registerNavigationCallback(req.PageKey, handler, req.Module, req.PageInstanceID)
	})
}

// HandleCallback dispatches a navigation callback registered by OpenDialog or OpenTab.
func (t *TableEngine) HandleCallback(ctx *engine.RequestContext, page engine.Page) (*engine.RuntimeResult, error) {
	if ctx.PageInstanceID == "" {
		if err := page.Init(ctx.BuildContext()); err != nil {
			return nil, err
		}
	}

	name := ctx.Params["callback"]
	handlerValue, ok := navigationCallbackHandlers.Load(navigationCallbackKey(ctx.PageKey, ctx.PageInstanceID, name))
	if !ok {
		return nil, fmt.Errorf("table engine: navigation callback %q not found", name)
	}
	handler, ok := handlerValue.(table.NavigationCallback)
	if !ok {
		return nil, fmt.Errorf("table engine: navigation callback %q has unexpected type", name)
	}

	state := table.TableStateConfig{}
	if t.dsl.State != nil {
		state = *t.dsl.State
	}
	params := make(map[string]any, len(ctx.Params))
	for key, value := range ctx.Params {
		params[key] = value
	}
	runtimeCtx := &table.TableRuntimeContext{
		State:  state,
		User:   ctx.User,
		System: ctx.System,
		Params: params,
		Extra:  navigationCallbackExtra(ctx),
	}
	t.configureNavigationCallbacks(runtimeCtx, ctx)
	handler(runtimeCtx)
	if err := runtimeCtx.Error(); err != nil {
		return nil, err
	}
	return &engine.RuntimeResult{
		Mutations:  runtimeCtx.Mutations,
		Navigation: runtimeCtx.Navigation,
	}, nil
}

func (t *TableEngine) handleCallbackRoute(pageKey string) engine.RouteHandler {
	return func(ctx *engine.RequestContext, page engine.Page) (any, error) {
		ctx.PageKey = pageKey
		if ctx.Params == nil {
			ctx.Params = engine.Params{}
		}
		requestEngine, ok := page.GetEngine().(*TableEngine)
		if !ok {
			return nil, fmt.Errorf("table engine: page %q returned unexpected engine %T", pageKey, page.GetEngine())
		}
		return requestEngine.HandleCallback(ctx, page)
	}
}

func navigationCallbackExtra(ctx *engine.RequestContext) map[string]any {
	extra := map[string]any{}
	if len(ctx.Body) > 0 {
		var payload map[string]any
		if err := json.Unmarshal(ctx.Body, &payload); err == nil {
			if nested, ok := payload["extra"].(map[string]any); ok {
				for key, value := range nested {
					extra[key] = value
				}
			} else if nested, ok := payload["result"].(map[string]any); ok {
				for key, value := range nested {
					extra[key] = value
				}
			} else {
				for key, value := range payload {
					extra[key] = value
				}
			}
		}
	}
	for key, value := range ctx.Query {
		extra[key] = value
	}
	return extra
}

func navigationCallbackName(handler table.NavigationCallback) string {
	value := reflect.ValueOf(handler)
	if !value.IsValid() || value.Kind() != reflect.Func {
		return "callback"
	}
	fn := runtime.FuncForPC(value.Pointer())
	if fn == nil {
		return "callback"
	}
	name := fn.Name()
	if slash := strings.LastIndex(name, "/"); slash >= 0 {
		name = name[slash+1:]
	}
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		name = name[dot+1:]
	}
	return toSnakeIdentifier(strings.TrimSuffix(name, "-fm"))
}

func toSnakeIdentifier(value string) string {
	var out []rune
	var lastUnderscore bool
	for i, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if unicode.IsUpper(r) && i > 0 && !lastUnderscore {
				out = append(out, '_')
			}
			out = append(out, unicode.ToLower(r))
			lastUnderscore = false
		default:
			if len(out) > 0 && !lastUnderscore {
				out = append(out, '_')
				lastUnderscore = true
			}
		}
	}
	result := strings.Trim(string(out), "_")
	if result == "" {
		return "callback"
	}
	return result
}

func navigationCallbackKey(pageKey, instanceID, name string) string {
	return pageKey + "/" + instanceID + "/" + name
}

func navigationCallbackRoutePath(module, pageKey, name string) string {
	return engine.RoutePath(module, "/event/"+pageKey+"/callback/"+name)
}
