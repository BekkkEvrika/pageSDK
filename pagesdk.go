// Package pagesdk provides the public entry points for building server-driven
// UI applications with pageSDK.
package pagesdk

import (
	"github.com/BekkkEvrika/pageSDK/access"
	"github.com/BekkkEvrika/pageSDK/app"
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/manifest"
)

// Application is the framework orchestrator.
type Application = app.Application
type Config = app.Config

type AccessManifest = access.Manifest
type AccessResource = access.Resource
type PermissionGroup = access.PermissionGroup
type AccessSyncProvider = access.AccessSyncProvider
type AccessSyncOptions = access.SyncOptions
type AccessDiff = access.Diff

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

// New creates a new pageSDK application.
func New(config ...Config) *Application {
	return app.New(config...)
}
