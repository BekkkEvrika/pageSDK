// Package pagesdk provides the public entry points for building server-driven
// UI applications with pageSDK.
package pagesdk

import (
	"time"

	"github.com/BekkkEvrika/pageSDK/access"
	"github.com/BekkkEvrika/pageSDK/app"
	"github.com/BekkkEvrika/pageSDK/authentication"
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/manifest"
	"github.com/gin-gonic/gin"
)

// Application is the framework orchestrator.
type Application = app.Application
type Config = app.Config
type Route = app.Route

type AccessManifest = access.Manifest
type AccessConfig = access.Config
type AccessGroup = access.AccessGroup
type AccessElement = access.AccessElement
type AccessGroupType = access.AccessGroupType
type ElementType = access.ElementType
type NoAccessBehavior = access.NoAccessBehavior
type AccessResource = access.Resource
type PermissionGroup = access.PermissionGroup
type AccessSyncProvider = access.AccessSyncProvider
type AccessSyncOptions = access.SyncOptions
type AccessDiff = access.Diff
type AccessAuthorizer = access.AccessAuthorizer
type CachedAuthorizer = access.CachedAuthorizer
type RPTClaimSource = access.RPTClaimSource
type JWTAuthorizationClaimSource = access.JWTAuthorizationClaimSource
type KeycloakUMAAccessAuthorizer = access.KeycloakUMAAccessAuthorizer
type KeycloakUMAProvider = access.KeycloakUMAProvider
type Authenticator = authentication.Authenticator
type AuthenticatorFunc = authentication.AuthenticatorFunc
type Principal = authentication.Principal
type KeycloakJWTConfig = authentication.KeycloakJWTConfig
type KeycloakJWTAuthenticator = authentication.KeycloakJWTAuthenticator

// InitFunc registers application pages during bootstrap.
type InitFunc = app.InitFunc

// Manifest is the registry of pages exposed by an application.
type Manifest = manifest.Manifest

// Page is the interface implemented by every page.
type Page = engine.Page

// PageFactory creates a new stateless page instance per request.
type PageFactory = engine.PageFactory

// Engine is implemented by page runtime engines.
type Engine = engine.Engine

// Dialog describes a client-side message dialog requested by runtime code.
type Dialog = engine.Dialog

// DialogLevel enumerates client-side dialog severity levels.
type DialogLevel = engine.DialogLevel

// DialogAction describes one action button in a client-side dialog.
type DialogAction = engine.DialogAction

const (
	DialogInfo    = engine.DialogInfo
	DialogWarning = engine.DialogWarning
	DialogError   = engine.DialogError
	DialogSuccess = engine.DialogSuccess
)

const (
	AccessGroupPage   = access.AccessGroupPage
	AccessGroupUI     = access.AccessGroupUI
	AccessGroupAction = access.AccessGroupAction

	ElementButton  = access.ElementButton
	ElementInput   = access.ElementInput
	ElementTable   = access.ElementTable
	ElementColumn  = access.ElementColumn
	ElementBlock   = access.ElementBlock
	ElementSection = access.ElementSection
	ElementMenu    = access.ElementMenu
	ElementTab     = access.ElementTab
	ElementAction  = access.ElementAction
	ElementCustom  = access.ElementCustom

	NoAccessHidden   = access.NoAccessHidden
	NoAccessDisabled = access.NoAccessDisabled
	NoAccessReadonly = access.NoAccessReadonly
	NoAccessRemove   = access.NoAccessRemove
)

// New creates a new pageSDK application.
// New creates an application and executes setup callbacks in declaration order.
func New(config Config, setups ...func(app *Application) error) *Application {
	application := app.New(config)
	callbacks := make([]app.SetupFunc, 0, len(setups))
	for _, setup := range setups {
		callbacks = append(callbacks, app.SetupFunc(setup))
	}
	application.Configure(callbacks...)
	return application
}

// PrincipalFromContext returns the verified identity available to a custom
// route handler registered through Application.RegisterRoute or
// Application.RegisterRoutes.
func PrincipalFromContext(ctx *gin.Context) (Principal, bool) {
	return app.PrincipalFromContext(ctx)
}

func NewKeycloakJWTAuthenticator(config KeycloakJWTConfig) *KeycloakJWTAuthenticator {
	return authentication.NewKeycloakJWTAuthenticator(config)
}

func NewKeycloakUMAProvider(config AccessConfig) *KeycloakUMAProvider {
	return access.NewKeycloakUMAProvider(config)
}

func NewKeycloakUMAAccessAuthorizer(config AccessConfig) *KeycloakUMAAccessAuthorizer {
	return access.NewKeycloakUMAAccessAuthorizer(config)
}

func NewCachedAuthorizer(source access.AccessGroupSource, ttl time.Duration) *CachedAuthorizer {
	return access.NewCachedAuthorizer(source, ttl)
}
