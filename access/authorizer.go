package access

import (
	"context"
	"sync"
	"time"
)

type bearerTokenContextKey struct{}

func WithBearerToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, bearerTokenContextKey{}, token)
}

func bearerTokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(bearerTokenContextKey{}).(string)
	return token
}

type AccessGroupSource interface {
	UserAccessGroups(ctx context.Context, userID string, user map[string]any) ([]string, error)
}

type CachedAuthorizer struct {
	source AccessGroupSource
	ttl    time.Duration
	now    func() time.Time
	mu     sync.Mutex
	cache  map[string]cachedAccessGroups
}

type cachedAccessGroups struct {
	expiresAt time.Time
	groups    map[string]struct{}
}

func NewCachedAuthorizer(source AccessGroupSource, ttl time.Duration) *CachedAuthorizer {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &CachedAuthorizer{
		source: source,
		ttl:    ttl,
		now:    time.Now,
		cache:  map[string]cachedAccessGroups{},
	}
}

func (a *CachedAuthorizer) UserAccessGroups(ctx context.Context, userID string, user map[string]any) ([]string, error) {
	groups, err := a.load(ctx, userID, user)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(groups))
	for group := range groups {
		result = append(result, group)
	}
	return result, nil
}

func (a *CachedAuthorizer) HasAccess(ctx context.Context, userID string, user map[string]any, accessGroupCode string) (bool, error) {
	groups, err := a.load(ctx, userID, user)
	if err != nil {
		return false, err
	}
	_, ok := groups[accessGroupCode]
	return ok, nil
}

func (a *CachedAuthorizer) Invalidate() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cache = map[string]cachedAccessGroups{}
}

func (a *CachedAuthorizer) load(ctx context.Context, userID string, user map[string]any) (map[string]struct{}, error) {
	now := a.now()
	a.mu.Lock()
	if cached, ok := a.cache[userID]; ok && now.Before(cached.expiresAt) {
		groups := cached.groups
		a.mu.Unlock()
		return groups, nil
	}
	a.mu.Unlock()

	values, err := a.source.UserAccessGroups(ctx, userID, user)
	if err != nil {
		return nil, err
	}
	groups := make(map[string]struct{}, len(values))
	for _, value := range values {
		groups[value] = struct{}{}
	}

	a.mu.Lock()
	a.cache[userID] = cachedAccessGroups{expiresAt: now.Add(a.ttl), groups: groups}
	a.mu.Unlock()
	return groups, nil
}

type StaticAuthorizer struct {
	Groups map[string][]string
}

func (a StaticAuthorizer) UserAccessGroups(_ context.Context, userID string, _ map[string]any) ([]string, error) {
	return append([]string(nil), a.Groups[userID]...), nil
}

func (a StaticAuthorizer) HasAccess(_ context.Context, userID string, _ map[string]any, accessGroupCode string) (bool, error) {
	for _, group := range a.Groups[userID] {
		if group == accessGroupCode {
			return true, nil
		}
	}
	return false, nil
}

func (a StaticAuthorizer) Invalidate() {}
