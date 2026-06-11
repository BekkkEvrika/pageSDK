package formengine

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unicode"

	"github.com/BekkkEvrika/pageSDK/engine"
)

var navigationCallbackHandlers sync.Map

func registerNavigationCallback(pageKey string, handler NavigationCallback) string {
	name := navigationCallbackName(handler)
	key := navigationCallbackKey(pageKey, name)
	navigationCallbackHandlers.Store(key, handler)
	return navigationCallbackRoutePath(pageKey, name)
}

// HandleCallback dispatches a navigation callback event registered by OpenDialog/OpenTab/OpenPage.
func (f *FormEngine) HandleCallback(ctx *engine.RequestContext, page engine.Page) (*engine.RuntimeResult, error) {
	if err := page.Init(ctx.BuildContext()); err != nil {
		return nil, err
	}

	name := ctx.Params["callback"]
	handlerValue, ok := navigationCallbackHandlers.Load(navigationCallbackKey(ctx.PageKey, name))
	if !ok {
		return nil, fmt.Errorf("form engine: navigation callback %q not found", name)
	}
	handler, ok := handlerValue.(NavigationCallback)
	if !ok {
		return nil, fmt.Errorf("form engine: navigation callback %q has unexpected type", name)
	}

	state, err := formState(ctx)
	if err != nil {
		return nil, err
	}
	normalizeFormStateValues(state, &f.root)

	runtimeCtx := NewRuntimeContext(ctx)
	runtimeCtx.Extra = callbackExtra(ctx)
	runtimeCtx.FormState = state
	runtimeCtx.Sender = state.Sender
	runtimeCtx.BindFormTree(&f.root)

	handler(runtimeCtx)
	if err := runtimeCtx.Error(); err != nil {
		return nil, err
	}
	return &engine.RuntimeResult{
		Mutations:  runtimeCtx.Mutations,
		Navigation: runtimeCtx.Navigation,
		Dialogs:    runtimeCtx.Dialogs,
	}, nil
}

func callbackExtra(ctx *engine.RequestContext) map[string]any {
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

func navigationCallbackName(handler NavigationCallback) string {
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
	name = strings.TrimSuffix(name, "-fm")
	return toSnakeIdentifier(name)
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

func navigationCallbackKey(pageKey, name string) string {
	return pageKey + "/" + name
}

func navigationCallbackRoutePath(pageKey, name string) string {
	return "/event/" + pageKey + "/callback/" + name
}
