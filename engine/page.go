package engine

// Page — интерфейс, который должна реализовывать каждая page.
// A Page is created for a render request and retained in memory for its events.
type Page interface {
	// Init is called once when a page instance is rendered.
	// It builds request-specific DSL and registers handlers for that instance.
	Init(ctx *BuildContext) error

	// GetEngine возвращает движок, к которому привязана page.
	// Используется Application для auto routing и rendering.
	GetEngine() Engine
}

// PageFactory creates a new Page for each render request.
type PageFactory func() Page
