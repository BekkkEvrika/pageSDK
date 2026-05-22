# pageSDK — Server-Driven UI Framework

## Быстрый старт

```bash
go mod tidy
go run main.go
```

---

## Архитектура

```
main()
  └── app.New()
  └── application.Bootstrap(projectInitial, ":8080")
        ├── projectInitial(app)        — регистрация pages в Manifest
        ├── registerRoutes()           — auto route generation из Manifest
        └── gin.Run(":8080")           — старт HTTP сервера
```

---

## Структура пакетов

```
pageSDK/
├── app/
│   └── application.go     — Application: orchestrator, bootstrap, route registration
├── engine/
│   ├── engine.go          — Engine interface + RouteDefinition + RouteHandler
│   ├── page.go            — Page interface + PageFactory
│   ├── context.go         — BuildContext + RuntimeContext
│   ├── mutation.go        — explicit mutation protocol
│   ├── navigation.go      — explicit navigation protocol
│   ├── form_engine.go     — FormEngine: routing semantics для форм
│   └── table_engine.go    — TableEngine: routing semantics для таблиц
├── manifest/
│   └── manifest.go        — Manifest: реестр pages (key → PageFactory)
├── form/
│   ├── form.go            — Form DSL struct
│   ├── container.go       — Container DSL
│   ├── input.go           — Input DSL
│   ├── formField.go       — FieldValidation, FileConfig, FieldAction
│   ├── input_actions.go   — FormAction, FormActionConfig
│   └── visibility.go      — Rule, RuleOperator
├── page/
│   ├── users_list.go      — UsersListPage (TableEngine)
│   ├── users_edit.go      — UsersEditPage (FormEngine)
│   └── admin_roles.go     — AdminRolesPage (TableEngine)
└── main.go                — Bootstrap entry point
```

---

## Ключевые концепции

### Application
- Единственный orchestrator
- Хранит Manifest
- Запускает bootstrap lifecycle
- Получает routes от Engine и регистрирует их в Gin
- **НЕ знает** routing details конкретного Engine, DSL, UI логику, бизнес-логику

### Manifest
```go
m.Register("users.list", page.NewUsersListPage)
m.Register("users.edit", page.NewUsersEditPage)
m.Register("admin.roles", page.NewAdminRolesPage)
```
- Ключ — стабильный runtime identifier (`users.list`, `admin.roles`)
- Используется для routing, lookup, UMA/API Gateway
- Заполняется **один раз** в `projectInitial`, не изменяется после Bootstrap

### Engine
```go
type Engine interface {
    ID() string
    Routes(pageKey string, page Page) []RouteDefinition
    Render(ctx *RequestContext, page Page) (*RenderResult, error)
    Handle(ctx *RequestContext, page Page) (*RuntimeResult, error)
}

type RouteDefinition struct {
    Method  string
    Path    string
    Handler RouteHandler
}
```
- Runtime behavior provider для конкретного типа Page
- Отвечает за routing semantics, DSL generation, runtime event handling
- Не хранит runtime state, page instances или sessions
- `FormEngine` → `GET /page/{key}` + `POST /event/{key}/:component/:action`
- `TableEngine` → `GET /page/{key}` + `POST /event/{key}/:component/:action`

### Page
```go
type Page interface {
    Init(ctx *engine.BuildContext) error
    GetEngine() Engine
}
```
- **Stateless**: создаётся на каждый request, уничтожается после ответа
- Embedding Engine даёт `GetEngine()` автоматически
- `Init()` — только сборка DSL/runtime model
- Event handlers используют `RuntimeContext`, а не `BuildContext`

### BuildContext / RuntimeContext
```go
type BuildContext struct {
    User   User
    System SystemKeys
    Params Params
}

type RuntimeContext struct {
    User       User
    System     SystemKeys
    Params     Params
    Mutations  *MutationWriter
    Navigation *NavigationWriter
}
```
- `BuildContext` доступен только в `Init()` и не содержит mutation/navigation APIs
- `RuntimeContext` доступен только в event handlers
- Backend не делает diff: все изменения идут через explicit mutations/navigation

### FormState
Для формы state не универсальный. Frontend может прислать только конкретный state формы:

```go
type FormState struct {
    Fields       map[string]FieldState `json:"fields,omitempty"`
    ActionID     string                `json:"actionId,omitempty"`
    Trigger      FormActionTrigger     `json:"trigger,omitempty"`
    ChangedField string                `json:"changedField,omitempty"`
}

type FieldState struct {
    Value any `json:"value,omitempty"`
}
```

- `Fields` — значения inputs, ключ строго равен `Input.Id`
- `ActionID` — `FormAction.ID`, например `save`
- `Trigger` — `FormAction.Trigger`, например `click` или `change`
- `ChangedField` — `Input.Id` поля, которое вызвало событие

### Bootstrap flow
```go
for _, entry := range manifest.All() {
    samplePage := entry.Factory()
    engine := samplePage.GetEngine()

    for _, route := range engine.Routes(entry.Key, samplePage) {
        registerRouteInGin(route)
    }
}
```
- Application проходит manifest
- Engine сам возвращает routes и runtime handlers
- Application только регистрирует `Method/Path/Handler` в Gin

### Создание новой Page
```go
// 1. Определить struct с embedding нужного Engine
type MyPage struct {
    *engine.FormEngine
    dsl any
}

// 2. Реализовать Init()
func (p *MyPage) Init(ctx *engine.BuildContext) error {
    p.dsl = myDSL
    return nil
}

func (p *MyPage) DSL() any {
    return p.dsl
}

func (p *MyPage) HandleEvent(ctx *engine.RuntimeContext, event engine.Event) error {
    ctx.Mutations.Update("status", "ok")
    return nil
}

// 3. Создать фабрику
func NewMyPage() engine.Page {
    return &MyPage{FormEngine: &engine.FormEngine{}}
}

// 4. Зарегистрировать в projectInitial
m.Register("my.page", page.NewMyPage)
```

---

## Auto-generated Routes

| Manifest Key   | Engine | Routes                                              |
|----------------|--------|-----------------------------------------------------|
| `users.list`   | Table  | `GET /page/users.list`, `POST /event/users.list/:component/:action` |
| `users.edit`   | Form   | `GET /page/users.edit`, `POST /event/users.edit/:component/:action` |
| `admin.roles`  | Table  | `GET /page/admin.roles`, `POST /event/admin.roles/:component/:action` |

---

## Архитектурные правила

- ✅ Разделение responsibilities: Application / Engine / Page / Manifest
- ✅ Stateless backend runtime — нет state между requests
- ✅ Deterministic bootstrap — routes генерируются один раз при старте
- ✅ No dynamic route registration during requests
- ✅ No hidden lifecycle — весь flow явный и предсказуемый
- ✅ UMA-friendly keys — стабильные dot-notation identifiers
