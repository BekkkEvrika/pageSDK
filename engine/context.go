package engine

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

// BuildContext creates a build-only context from a request snapshot.
func (r *RequestContext) BuildContext() *BuildContext {
	return &BuildContext{
		User:   r.User,
		System: r.System,
		Params: r.Params,
	}
}
