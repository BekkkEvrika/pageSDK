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

func bindDialogHandler(pageKey string, dialog engine.Dialog, handler DialogHandler) engine.Dialog {
	if handler == nil {
		return dialog
	}
	id := "dialog-" + strconv.FormatUint(atomic.AddUint64(&dialogCallbackSeq, 1), 10)
	dialogCallbackHandlers.Store(id, handler)

	url := dialogEventRoutePath(pageKey, id)
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

func dialogEventRoutePath(pageKey, dialogID string) string {
	if pageKey == "" {
		pageKey = "{page}"
	}
	return "/event/" + pageKey + "/dialog/" + dialogID
}
