package engine

// Page — интерфейс, который должна реализовывать каждая page.
// Page stateless: создаётся на каждый request, уничтожается после ответа.
type Page interface {
	// Init вызывается один раз при каждом request.
	// Здесь происходит DSL-сборка, загрузка данных, настройка полей.
	Init(ctx *BuildContext) error

	// GetEngine возвращает движок, к которому привязана page.
	// Используется Application для auto routing и rendering.
	GetEngine() Engine
}

// PageFactory — функция-фабрика, создающая новый экземпляр Page.
// Регистрируется в Manifest. Вызывается на каждый request.
type PageFactory func() Page

// DSLPage can be implemented by pages that expose their built DSL to an engine.
type DSLPage interface {
	DSL() any
}
