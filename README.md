# pageSDK

`pageSDK` - это Go-библиотека для server-driven UI: backend описывает страницу как DSL, frontend отрисовывает этот DSL, а события пользователя отправляет обратно на backend. Backend не хранит состояние интерфейса между запросами. Любое изменение UI возвращается клиенту явно: через `mutations` и `navigation`.

Проект сейчас содержит два движка:

- `FormEngine` - формы, поля, кнопки, обработчики `click` и `change`.
- `TableEngine` - простые табличные страницы с колонками и универсальным event route.

## Установка

```bash
go get github.com/BekkkEvrika/pageSDK
```

## Минимальный запуск

```go
package main

import (
	pagesdk "github.com/BekkkEvrika/pageSDK"
	"github.com/BekkkEvrika/pageSDK/engine"
)

func main() {
	application := pagesdk.New()
	if err := application.Bootstrap(registerPages, ":8080"); err != nil {
		panic(err)
	}
}

func registerPages(a *pagesdk.Application) {
	a.Manifest().Register("users.edit", NewUsersEditPage)
}

type UsersEditPage struct {
	*engine.FormEngine
}

func NewUsersEditPage() pagesdk.Page {
	return &UsersEditPage{
		FormEngine: &engine.FormEngine{},
	}
}

func (p *UsersEditPage) Init(ctx *engine.BuildContext) error {
	name := p.Text("name")
	name.SetLabel("User name")
	name.SetPlaceholder("Enter user name")

	save := p.Button("save")
	save.SetLabel("Save")
	save.SetVariant("primary")
	save.SetOnClick(onSave)

	status := p.Text("status")
	status.SetLabel("Status")
	status.SetReadOnly(true)

	return nil
}

func onSave(ctx *engine.RuntimeContext) {
	status, err := ctx.GetTextById("status")
	if err != nil {
		return
	}
	status.SetValue("Saved")
}
```

После старта приложение будет отдавать DSL страницы:

```http
GET /page/users.edit
```

И принимать событие кнопки:

```http
POST /event/users.edit/button/save
```

## Локальный пример из репозитория

```bash
go mod tidy
mkdir -p logs
go run ./cmd/pagesdk-example
```

Проверить render:

```bash
curl http://localhost:8080/page/users.edit
```

Отправить click-событие:

```bash
curl -X POST http://localhost:8080/event/users.edit/button/save \
  -H 'Content-Type: application/json' \
  -d '{
    "elements": [
      {"id": "status", "type": "text", "label": "Status", "value": ""}
    ],
    "sender": {"id": "save", "type": "button", "label": "Save", "actionId": "save"},
    "actionId": "save",
    "trigger": "click",
    "changedField": "save"
  }'
```

Ответ будет содержать явные изменения:

```json
{
  "mutations": [
    {
      "type": "update",
      "path": "controls.status.value",
      "value": "Saved"
    }
  ]
}
```

## Как устроен проект

```text
pageSDK/
├── pagesdk.go             public entry point библиотеки
├── app/                   Application: bootstrap, manifest, регистрация routes в Gin
├── engine/                движки, lifecycle, контексты, mutations, navigation
├── form/                  структуры Form DSL и runtime payload формы
├── manifest/              реестр page key -> PageFactory
├── page/                  примерные страницы
├── docs/client-events.md  подробный протокол для frontend
└── cmd/pagesdk-example/   запускаемый пример
```

Главные сущности:

- `Application` - запускает приложение, вызывает init-функцию, читает manifest и регистрирует HTTP routes.
- `Manifest` - хранит стабильные ключи страниц и фабрики страниц.
- `Page` - stateless-описание одной страницы.
- `Engine` - отвечает за DSL, routes и обработку событий конкретного типа страницы.
- `BuildContext` - контекст для построения DSL внутри `Init`.
- `RuntimeContext` - контекст для event handlers, чтения runtime state, записи mutations и navigation.

## Lifecycle приложения

Запуск выглядит так:

```text
main()
  -> pagesdk.New()
  -> application.Bootstrap(registerPages, ":8080")
       -> registerPages(app)
       -> app.Manifest().All()
       -> для каждой page создается sample Page
       -> Engine.Routes(pageKey, samplePage)
       -> routes регистрируются в Gin
       -> gin.Run(":8080")
```

Во время bootstrap `Application` создает временный экземпляр страницы только для получения списка routes. Этот экземпляр не используется для реальных запросов.

Каждый HTTP-запрос проходит отдельный runtime lifecycle:

```text
HTTP request
  -> создать новую Page через PageFactory
  -> внутри Page создать новый Engine
  -> вызвать Page.Init(...)
  -> построить DSL или обработать event
  -> вернуть JSON response
  -> уничтожить Page и Engine
```

Из-за этого важное правило: нельзя хранить runtime-состояние в `Page` или `Engine` и ожидать, что оно переживет следующий запрос.

## Manifest

Страницы регистрируются один раз при старте:

```go
func registerPages(a *pagesdk.Application) {
	a.Manifest().Register("users.list", page.NewUsersListPage)
	a.Manifest().Register("users.edit", page.NewUsersEditPage)
	a.Manifest().Register("admin.roles", page.NewAdminRolesPage)
}
```

Ключ страницы:

- должен быть стабильным;
- используется в URL: `/page/users.edit`;
- используется frontend-клиентом и gateway/integration-слоем;
- не должен зависеть от пользователя, языка, фильтров или временных данных.

При повторной регистрации одного и того же ключа `Manifest.Register` вызывает `panic`, потому что это ошибка конфигурации приложения.

## Page

Страница реализует интерфейс:

```go
type Page interface {
	Init(ctx *engine.BuildContext) error
	GetEngine() engine.Engine
}
```

Обычно `GetEngine()` писать вручную не нужно: достаточно embedded engine.

```go
type UsersEditPage struct {
	*engine.FormEngine
}

func NewUsersEditPage() engine.Page {
	return &UsersEditPage{
		FormEngine: &engine.FormEngine{},
	}
}
```

Метод `Init` должен декларативно собрать страницу:

- создать поля, контейнеры, колонки;
- назначить обработчики событий;
- использовать данные из `BuildContext`, если DSL зависит от пользователя, query params или route params.

`Init` не должен выполнять runtime-мутации UI. Для этого есть event handlers и `RuntimeContext`.

## FormEngine

`FormEngine` генерирует:

```text
GET  /page/{pageKey}
POST /event/{pageKey}/{component}/{actionID}
```

Например:

```text
GET  /page/users.edit
POST /event/users.edit/button/save
POST /event/users.edit/text/name
```

Routes для form events статические. Они создаются во время bootstrap из зарегистрированных listeners. Frontend не должен придумывать URL самостоятельно: он должен читать URL из `dsl.actions`.

### Создание формы через helper-методы

```go
func (p *UsersEditPage) Init(ctx *engine.BuildContext) error {
	name := p.Text("name")
	name.SetLabel("User name")
	name.SetOnChange(onNameChange)

	email := p.Text("email")
	email.SetLabel("Email")

	save := p.Button("save")
	save.SetLabel("Save")
	save.SetOnClick(onSave)

	return nil
}
```

`Text` и `Button` добавляют элементы в default container. Если контейнера еще нет, `FormEngine` создаст контейнер `main`.

Доступные типы input:

- `select`
- `date`
- `datetime`
- `text`
- `number`
- `checkbox`
- `label`
- `search`
- `textarea`
- `hidden`
- `file`
- `button`

Для чтения элемента по ID есть typed getters:

```go
text, err := p.GetTextById("name")
button, err := p.GetButtonById("save")
checkbox, err := p.GetCheckboxById("enabled")
```

Getter возвращает ошибку, если элемента нет или его тип отличается от ожидаемого.

### Создание формы через полный DSL

```go
func (p *UsersEditPage) Init(ctx *engine.BuildContext) error {
	p.CreateForm(inputs.Form{
		Containers: &[]inputs.Container{
			{
				Key:       "main",
				Direction: "vertical",
				Gap:       16,
				Fields: []inputs.Input{
					{Id: "name", Type: inputs.InputTypeText, Label: "Name"},
					{Id: "email", Type: inputs.InputTypeText, Label: "Email"},
					{Id: "save", Type: inputs.InputTypeButton, Label: "Save"},
				},
			},
		},
	})

	save, err := p.GetButtonById("save")
	if err != nil {
		return err
	}
	save.SetOnClick(onSave)

	return nil
}
```

Контейнеры могут быть вложенными. Frontend должен отрисовывать дерево `containers` рекурсивно.

### Что можно делать в Init

В `Init` доступны build-time controls:

```go
name := p.Text("name")
name.SetLabel("Name")
name.SetPlaceholder("Enter name")
name.SetOnChange(onNameChange)
```

Здесь можно:

- создавать поля и контейнеры;
- задавать label, placeholder, validation, options, visibility и другие свойства DSL;
- назначать `SetOnClick` и `SetOnChange`.

Обработчики назначаются только в `Init`. Runtime controls не имеют `SetOnClick` и `SetOnChange`.

## Event handlers

Handler получает `*engine.RuntimeContext`:

```go
func onSave(ctx *engine.RuntimeContext) {
	status, err := ctx.GetTextById("status")
	if err != nil {
		return
	}

	status.SetValue("Saved")
	status.SetLabel("Current status")
}
```

В handler можно:

- получить runtime control через `ctx.GetTextById`, `ctx.GetButtonById`, `ctx.GetCheckboxById` и другие getters;
- прочитать текущее значение из `control.Value`;
- прочитать полный runtime element через `control.Element()`;
- отправить mutations через `SetValue`, `SetLabel`, `SetVisibility`, `ctx.Form().Add`, `ctx.Remove`;
- отправить navigation через `ctx.OpenDialog`, `ctx.OpenTab`, `ctx.Close`, `ctx.CloseWithResult`.

В handler нельзя:

- регистрировать новые event handlers;
- полагаться на состояние, сохраненное в предыдущем запросе;
- менять frontend неявно. Все изменения должны попасть в response как `mutations` или `navigation`.

Пример чтения значения, которое пришло от frontend:

```go
func onNameChange(ctx *engine.RuntimeContext) {
	name, err := ctx.GetTextById("name")
	if err != nil {
		return
	}

	currentValue := name.Value
	_ = currentValue

	nameChanged, err := ctx.GetCheckboxById("nameChanged")
	if err != nil {
		return
	}
	nameChanged.SetValue(true)
}
```

## Runtime payload формы

Frontend отправляет текущее состояние элементов формы:

```json
{
  "elements": [
    {
      "id": "name",
      "type": "text",
      "label": "User name",
      "value": "Alice"
    },
    {
      "id": "save",
      "type": "button",
      "label": "Save",
      "actionId": "save",
      "value": true
    }
  ],
  "sender": {
    "id": "save",
    "type": "button",
    "label": "Save",
    "actionId": "save",
    "value": true
  },
  "actionId": "save",
  "trigger": "click",
  "changedField": "save"
}
```

Поля payload:

- `elements` - все актуальные элементы формы вместе с runtime `value`;
- `sender` - элемент, который вызвал событие;
- `actionId` - ID действия или контрола;
- `trigger` - тип события, например `click` или `change`;
- `changedField` - ID элемента, который изменился или был нажат.

Если frontend передает дополнительные свойства элемента, backend сохранит их в `ElementState.Props`.

Старый формат `fields` тоже поддерживается как fallback, но новый frontend должен использовать `elements`.

## Mutations

Backend не делает diff UI. Handler явно записывает изменения, а frontend применяет их в полученном порядке.

Поддерживаются три типа mutations:

```json
{"type":"update","path":"controls.status.value","value":"Saved"}
{"type":"add","path":"form.controls","value":{"id":"extra","type":"text","label":"Extra"}}
{"type":"remove","path":"controls.oldField"}
```

Частые paths:

```text
controls.{id}.label
controls.{id}.value
controls.{id}.visibility
form.controls
```

Go API:

```go
status.SetValue("Saved")
status.SetLabel("Status")
status.SetVisibility(true)

ctx.Form().Add(inputs.Input{
	Id:    "extra",
	Type:  inputs.InputTypeText,
	Label: "Extra",
})

ctx.Remove("oldField")
```

## Navigation

Navigation не является mutation. Она возвращается отдельным массивом:

```json
{
  "navigation": [
    {
      "type": "openDialog",
      "page": "users.edit",
      "params": {
        "id": "42"
      }
    }
  ]
}
```

Go API:

```go
ctx.OpenDialog("users.edit", engine.Params{"id": "42"})
ctx.OpenTab("admin.roles")
ctx.Close()
ctx.CloseWithResult(map[string]any{"saved": true})
```

Frontend сам владеет navigation stack. Backend остается stateless.

## TableEngine

`TableEngine` генерирует:

```text
GET  /page/{pageKey}
POST /event/{pageKey}/:component/:action
```

Пример страницы:

```go
type UsersListPage struct {
	*engine.TableEngine
}

func NewUsersListPage() engine.Page {
	return &UsersListPage{
		TableEngine: &engine.TableEngine{},
	}
}

func (p *UsersListPage) Init(ctx *engine.BuildContext) error {
	p.Column("id", "ID")
	p.Column("name", "Name")
	p.Column("email", "Email")
	return nil
}
```

Для обработки table events страница может реализовать `engine.EventHandler`:

```go
func (p *UsersListPage) HandleEvent(ctx *engine.RuntimeContext, event engine.Event) error {
	switch event.Action {
	case "open":
		ctx.OpenDialog("users.edit", engine.Params{"id": event.Component})
	}
	return nil
}
```

## HTTP responses

Render response:

```json
{
  "pageKey": "users.edit",
  "engine": "form",
  "dsl": {
    "containers": [],
    "actions": []
  }
}
```

Runtime response:

```json
{
  "mutations": [],
  "navigation": [],
  "result": null
}
```

`mutations`, `navigation` и `result` могут отсутствовать, если они пустые.

## BuildContext и RuntimeContext

`BuildContext` используется только в `Page.Init`:

```go
type BuildContext struct {
	User   engine.User
	System engine.SystemKeys
	Params engine.Params
}
```

`RuntimeContext` используется только в event handlers:

```go
type RuntimeContext struct {
	User       engine.User
	System     engine.SystemKeys
	Params     engine.Params
	FormState  *inputs.FormState
	Sender     *inputs.ElementState
	Mutations  []engine.Mutation
	Navigation []engine.NavigationItem
}
```

Дополнительно `FormEngine` кладет в `ctx.Params["form.actionId"]` значение `actionId` из payload.

## Правила для frontend-клиента

Frontend должен:

- загрузить страницу через `GET /page/{pageKey}`;
- отрисовать `dsl.containers` как дерево;
- считать `id` элементов стабильными ключами;
- брать event URL из `dsl.actions`;
- на событие отправлять актуальный `elements` payload;
- применять `mutations` строго в порядке получения;
- обрабатывать `navigation` отдельно от mutations;
- корректно переживать `4xx/5xx`, не ломая локальное состояние UI.

Более подробный протокол описан в [docs/client-events.md](docs/client-events.md).

## Архитектурные правила

- `Application` не знает DSL и бизнес-логику страниц.
- `Manifest` регистрируется один раз во время bootstrap.
- `Page` создается заново на каждый request.
- `Engine` создается заново вместе с `Page` и хранит DSL только внутри текущего request.
- `Init` строит DSL и регистрирует handlers.
- Runtime handler не перестраивает приложение, а возвращает explicit mutations/navigation.
- Form event routes статические и детерминированные.
- Backend не хранит UI state между запросами.
- Frontend присылает runtime state, который нужен handler.
- Component registry внутри `FormEngine` является индексом для поиска, а source of truth - дерево контейнеров.

## Частые ошибки

### Handler не найден

Причина: в `Init` не был вызван `SetOnClick` или `SetOnChange` для нужного элемента, поэтому route не был сгенерирован.

```go
save := p.Button("save")
save.SetOnClick(onSave)
```

### Control not found

Причина: handler пытается получить элемент, которого нет в DSL, или использует неправильный getter.

```go
status, err := ctx.GetTextById("status")
if err != nil {
	return
}
```

Если `status` имеет тип `checkbox`, `GetTextById("status")` вернет ошибку.

### Значение не приходит в handler

Причина: frontend не отправил элемент в `elements` или указал другой `id`.

Handler читает runtime value так:

```go
name, err := ctx.GetTextById("name")
if err != nil {
	return
}
value := name.Value
```

### Изменение не видно на frontend

Причина: backend изменил локальную переменную или DSL-структуру, но не записал mutation.

В runtime нужно использовать методы, которые пишут mutation:

```go
field.SetValue("new value")
field.SetLabel("New label")
field.SetVisibility(false)
```

## Публичный entry point

Пакет `github.com/BekkkEvrika/pageSDK` реэкспортирует главные типы:

```go
type Application = app.Application
type InitFunc = app.InitFunc
type Manifest = manifest.Manifest
type Page = engine.Page
type PageFactory = engine.PageFactory
type Engine = engine.Engine

func New() *Application
```

Для DSL форм, движков и контекстов используются подпакеты:

```go
import (
	pagesdk "github.com/BekkkEvrika/pageSDK"
	"github.com/BekkkEvrika/pageSDK/engine"
	inputs "github.com/BekkkEvrika/pageSDK/form"
)
```
