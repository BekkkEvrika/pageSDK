package formengine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/BekkkEvrika/pageSDK/engine"
)

// DialogHandler handles a client-side dialog action value.
type DialogHandler func(value string)

var (
	dialogCallbackSeq      uint64
	dialogCallbackHandlers sync.Map
)

type dialogCallbackPayload struct {
	Value       string `json:"value,omitempty"`
	ActionValue string `json:"actionValue,omitempty"`
	Name        string `json:"name,omitempty"`
}

func bindDialogHandler(pageKey string, dialog engine.Dialog, handler DialogHandler, module ...string) engine.Dialog {
	if handler == nil {
		return dialog
	}
	id := "dialog-" + strconv.FormatUint(atomic.AddUint64(&dialogCallbackSeq, 1), 10)
	dialogCallbackHandlers.Store(id, handler)

	moduleName := ""
	instanceID := ""
	if len(module) > 0 {
		moduleName = module[0]
	}
	if len(module) > 1 {
		instanceID = module[1]
	}
	url := engine.PageInstanceURL(dialogEventRoutePath(moduleName, pageKey, id), instanceID)
	for i := range dialog.Actions {
		dialog.Actions[i].URL = url
		dialog.Actions[i].Method = http.MethodPost
	}
	return dialog
}

func handleDialogCallback(ctx *engine.RequestContext) (*engine.RuntimeResult, error) {
	id := ctx.Params["dialog"]
	value, err := dialogActionValue(ctx)
	if err != nil {
		return nil, err
	}
	handlerValue, ok := dialogCallbackHandlers.LoadAndDelete(id)
	if !ok {
		return nil, fmt.Errorf("form engine: dialog handler %q not found", id)
	}
	handler, ok := handlerValue.(DialogHandler)
	if !ok {
		return nil, fmt.Errorf("form engine: dialog handler %q has unexpected type", id)
	}
	handler(value)
	return &engine.RuntimeResult{}, nil
}

func dialogActionValue(ctx *engine.RequestContext) (string, error) {
	var payload dialogCallbackPayload
	if len(ctx.Body) > 0 {
		if err := json.Unmarshal(ctx.Body, &payload); err != nil {
			return "", err
		}
	}
	switch {
	case payload.Value != "":
		return payload.Value, nil
	case payload.ActionValue != "":
		return payload.ActionValue, nil
	case payload.Name != "":
		return payload.Name, nil
	case ctx.Params["value"] != "":
		return ctx.Params["value"], nil
	case ctx.Query["value"] != "":
		return ctx.Query["value"], nil
	default:
		return "", nil
	}
}

func dialogEventRoutePath(args ...string) string {
	var module, pageKey, dialogID string
	switch len(args) {
	case 2:
		pageKey, dialogID = args[0], args[1]
	case 3:
		module, pageKey, dialogID = args[0], args[1], args[2]
	default:
		return ""
	}
	if pageKey == "" {
		pageKey = "{page}"
	}
	return engine.RoutePath(module, "/event/"+pageKey+"/dialog/"+dialogID)
}
