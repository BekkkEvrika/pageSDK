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
- Хранит DSL/runtime state внутри конкретного per-request Engine instance
- Хранит root container, DSL tree, registered components, handler registry и generated event route metadata
- UI source of truth — container hierarchy; registry используется только для fast lookup/path resolution
- Не является shared singleton и не хранит state между requests
- `FormEngine` → `GET /page/{key}` + static `POST /event/{key}/{component}/{actionID}` routes from registered listeners
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
- `Init()` — declarative layer: вызывает DSL methods embedded Engine
- Не хранит controls, handlers registry или DSL tree
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
    State      map[string]any
    Mutations  []Mutation
    Navigation []NavigationItem
}
```
- `BuildContext` доступен только в `Init()` и не содержит mutation/navigation APIs
- `RuntimeContext` доступен только в event handlers
- Backend не делает diff: все изменения идут только через explicit `RuntimeContext` operations

### Runtime mutations
```go
func OnSave(ctx *engine.RuntimeContext) {
    ctx.Text("title").SetText("Saved")
    ctx.SetState("loading", false)
    ctx.Form().Add(ctx.Text("dynamic_text"))
    ctx.Remove("old_button")
    ctx.OpenDialog("users.edit")
}
```

Mutation protocol поддерживает только `update`, `add`, `remove`:

```json
{"type":"update","path":"controls.title.text","value":"Saved"}
{"type":"add","path":"form.controls","value":{"id":"dynamic_text","type":"text"}}
{"type":"remove","path":"controls.old_button"}
```

Navigation хранится отдельно от mutations: `OpenDialog`, `OpenTab`, `Close`, `CloseWithResult`.

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
- Для каждой entry создаётся sample Page, вызывается `Init()`, Engine собирает DSL/components/handlers/routes
- Engine сам возвращает deterministic route definitions и runtime handlers
- Application только регистрирует `Method/Path/Handler` в Gin
- Runtime handler на каждый request работает со свежими `Page` и `Engine` из `entry.Factory()`

### Request lifecycle
```text
Request
↓
Create Page
↓
Create Engine
↓
Init()
↓
DSL built inside Engine
↓
Handle request
↓
Dispose
```

### Создание новой Page
```go
// 1. Определить struct с embedding нужного Engine
type MyPage struct {
    *engine.FormEngine
}

// 2. Реализовать Init()
func (p *MyPage) Init(ctx *engine.BuildContext) error {
    p.CreateForm(inputs.Form{
        Containers: &[]inputs.Container{
            {
                Key: "main",
                Fields: []inputs.Input{
                    {Id: "name", Type: "text", Label: "Name"},
                    {Id: "save", Type: "button", Label: "Save"},
                },
            },
        },
    })
    button, err := p.GetButtonById("save")
    if err != nil {
        return err
    }
    button.SetOnClick(OnSave)
    return nil
}

func OnSave(ctx *engine.RuntimeContext) {
    ctx.Text("status").SetText("Saved")
    ctx.SetState("saved", true)
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
| `users.edit`   | Form   | `GET /page/users.edit`, `POST /event/users.edit/button/save` |
| `admin.roles`  | Table  | `GET /page/admin.roles`, `POST /event/admin.roles/:component/:action` |

---

## Архитектурные правила

- ✅ Разделение responsibilities: Application / Engine / Page / Manifest
- ✅ Stateless backend runtime — нет state между requests
- ✅ DSL принадлежит конкретному Engine instance, а не Page
- ✅ UI хранится как tree hierarchy от root container
- ✅ Component registry является индексом поверх tree, а не source of truth
- ✅ Engine instance создаётся на каждый request и уничтожается после ответа
- ✅ Deterministic bootstrap — routes генерируются один раз при старте
- ✅ No dynamic route registration during requests
- ✅ No global DSL storage, shared engines или stateful singleton engines
- ✅ No hidden lifecycle — весь flow явный и предсказуемый
- ✅ UMA-friendly keys — стабильные dot-notation identifiers
