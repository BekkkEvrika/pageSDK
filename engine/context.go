package engine

import inputs "github.com/behzod/pageSDK/form"

// User contains authenticated user claims available to build and runtime code.
type User map[string]any

// SystemKeys contains stable system values extracted from the request token.
type SystemKeys map[string]string

// Params contains deterministic route/query/page parameters.
type Params map[string]string

// BuildContext is used only by Page.Init for DSL generation.
type BuildContext struct {
	User   User
	System SystemKeys
	Params Params
}

// RuntimeContext is used only by event handlers.
type RuntimeContext struct {
	User       User
	System     SystemKeys
	Params     Params
	FormState  *inputs.FormState
	Sender     *inputs.ElementState
	Mutations  []Mutation
	Navigation []NavigationItem
	formRoot   *inputs.Container
	err        error
}

// BuildContext creates a build-only context from a request snapshot.
func (r *RequestContext) BuildContext() *BuildContext {
	return &BuildContext{
		User:   r.User,
		System: r.System,
		Params: r.Params,
	}
}

// RuntimeContext creates a runtime context with explicit mutation/navigation writers.
func (r *RequestContext) RuntimeContext() *RuntimeContext {
	params := r.Params
	if params == nil {
		params = Params{}
	}
	return &RuntimeContext{
		User:   r.User,
		System: r.System,
		Params: params,
	}
}
