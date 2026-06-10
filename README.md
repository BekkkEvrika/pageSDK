# pageSDK — Server-Driven UI Framework

## Установка как библиотеки

```bash
go get github.com/behzod/pageSDK
```

## Быстрый старт в своём приложении

```go
package main

import (
	pagesdk "github.com/behzod/pageSDK"
	"github.com/behzod/pageSDK/page"
)

func main() {
	application := pagesdk.New()
	if err := application.Bootstrap(projectInitial, ":8080"); err != nil {
		panic(err)
	}
}

func projectInitial(a *pagesdk.Application) {
	a.Manifest().Register("users.edit", page.NewUsersEditPage)
}
```

## Локальный пример

```bash
go mod tidy
go run ./cmd/pagesdk-example
```

## Клиентский протокол

- [Client Event Protocol](docs/client-events.md) — как frontend получает DSL, вызывает event routes и применяет mutations/navigation.

---

## Архитектура

```
main()
  └── pagesdk.New()
  └── application.Bootstrap(projectInitial, ":8080")
        ├── projectInitial(app)        — регистрация pages в Manifest
        ├── registerRoutes()           — auto route generation из Manifest
        └── gin.Run(":8080")           — старт HTTP сервера
```

---

## Структура пакетов

```
pageSDK/
├── pagesdk.go             — public library entry point
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
└── cmd/
    └── pagesdk-example/   — runnable example application
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
    FormState  *inputs.FormState
    Sender     *inputs.ElementState
    Mutations  []Mutation
    Navigation []NavigationItem
}
```
- `BuildContext` доступен только в `Init()` и не содержит mutation/navigation APIs
- `RuntimeContext` доступен только в event handlers
- Backend не делает diff: все изменения идут только через explicit `RuntimeContext` operations
- Runtime handlers получают существующие controls через `ctx.GetTextById`, `ctx.GetButtonById` и похожие методы. Если элемента нет в DSL или тип не совпадает, event request завершается ошибкой.
- Runtime control заполняется данными, пришедшими от frontend: `control.Value`, `control.Props` и `control.Element()` доступны внутри handler.
- Runtime controls не имеют `SetOnClick`/`SetOnChange`: обработчики назначаются только во время `Init()`.

### Runtime mutations
```go
func OnSave(ctx *engine.RuntimeContext) {
    title, err := ctx.GetTextById("title")
    if err != nil {
        return
    }
    loading, err := ctx.GetTextById("loading")
    if err != nil {
        return
    }
    title.SetLabel("Saved")
    loading.SetValue(false)
    ctx.Form().Add(inputs.Input{Id: "dynamic_text", Type: inputs.InputTypeText})
    ctx.Remove("old_button")
    ctx.OpenDialog("users.edit")
}
```

`SetValue`, `SetLabel` и `SetVisibility` пишут patch в response. `Value` читает runtime value из event payload:

```go
func OnNameChange(ctx *engine.RuntimeContext) {
    name, err := ctx.GetTextById("name")
    if err != nil {
        return
    }
    currentValue := name.Value
    _ = currentValue
}
```

Mutation protocol поддерживает только `update`, `add`, `remove`:

```json
{"type":"update","path":"controls.title.label","value":"Saved"}
{"type":"add","path":"form.controls","value":{"id":"dynamic_text","type":"text"}}
{"type":"remove","path":"controls.old_button"}
```

Navigation хранится отдельно от mutations: `OpenDialog`, `OpenTab`, `Close`, `CloseWithResult`.

### FormState
Frontend присылает универсальный runtime state элементов формы:

```go
type FormState struct {
    Elements     []ElementState        `json:"elements,omitempty"`
    Sender       *ElementState         `json:"sender,omitempty"`
    Fields       map[string]FieldState `json:"fields,omitempty"` // legacy fallback
    Form         *Form                 `json:"form,omitempty"`
    ActionID     string                `json:"actionId,omitempty"`
    Trigger      FormActionTrigger     `json:"trigger,omitempty"`
    ChangedField string                `json:"changedField,omitempty"`
}

type ElementState struct {
    Input
    Value any            `json:"value,omitempty"`
    Props map[string]any `json:"props,omitempty"`
}
```

- `Elements` — текущие элементы формы/DSL с runtime `value`
- `Sender` — элемент, который вызвал событие
- `ActionID` — `FormAction.ID`, например `save`
- `Trigger` — `FormAction.Trigger`, например `click` или `change`
- `ChangedField` — `Input.Id` элемента, который вызвал событие

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
    status, err := ctx.GetTextById("status")
    if err != nil {
        return
    }
    saved, err := ctx.GetTextById("saved")
    if err != nil {
        return
    }
    status.SetLabel("Saved")
    saved.SetValue(true)
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
