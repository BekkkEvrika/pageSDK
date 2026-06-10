// Package pagesdk provides the public entry points for building server-driven
// UI applications with pageSDK.
package pagesdk

import (
	"github.com/BekkkEvrika/pageSDK/app"
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/manifest"
)

// Application is the framework orchestrator.
type Application = app.Application

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

// New creates a new pageSDK application.
func New() *Application {
	return app.New()
}
