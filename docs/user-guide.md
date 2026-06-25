# Руководство пользователя pageSDK

Это руководство предназначено для Go-разработчиков, которые подключают
`pageSDK` как библиотеку и создают на ней server-driven страницы.

Frontend-разработчикам нужен отдельный документ:
[Client Event Protocol](client-events.md).

Подробная архитектура хранения и lifecycle:
[Page instances](page-instances.md).

## 1. Что представляет собой pageSDK

`pageSDK` — backend-библиотека для server-driven UI.

Обычный frontend хранит структуру страницы и большую часть UI-логики у себя. В
pageSDK структура страницы создается backend-кодом:

1. Go page строит декларативный DSL.
2. Frontend получает DSL по HTTP и отрисовывает его.
3. Backend сохраняет созданный Page как in-memory instance.
4. Пользователь вызывает событие.
5. Frontend отправляет текущее состояние на event URL этого instance.
6. Backend выполняет handler на том же объекте Page без повторного `Init`.
7. Handler возвращает явные mutations, navigation или dialogs.
8. Frontend применяет response.

```text
PageFactory -> Page.Init(request context)
                   │
                   ▼
          Render DSL + instanceId ─────────► Frontend renderer
                   │                            │
                   │ Page сохраняется           │ event URL содержит
                   │ in-memory                  │ pageInstanceId query
                   ▼                            ▼
          Page instance ◄────────────── Event HTTP request
                   │
                   └── handler без Init ──► mutations/navigation/dialogs
```

Backend хранит объект `Page`, собранный DSL и зарегистрированные handlers между
render и event-запросами. Текущие browser values, selection, navigation stack и
визуальное состояние по-прежнему принадлежат frontend и передаются в payload.

## 2. Что входит в библиотеку

Основные части:

| Пакет | Назначение |
|---|---|
| `pageSDK` | `Application`, `Manifest`, `Page`, `New()` |
| `engine` | общие contracts, contexts, mutations, navigation, dialogs |
| `engine/formengine` | построение и runtime forms |
| `engine/tableengine` | построение и runtime tables |
| `form` | структуры Form DSL и payload |
| `table` | структуры Table DSL, constants и builders |
| `manifest` | registry страниц |

Встроенный `Application` использует Gin как HTTP transport. Пользователь
библиотеки работает с `Application`, `Page`, одним из engines и typed runtime
contexts; прямой доступ к `gin.Context` страницам не передается.

## 3. Установка и первый запуск

Установка:

```bash
go get github.com/BekkkEvrika/pageSDK
```

Минимальный `main.go`:

```go
package main

import (
	"time"

	pagesdk "github.com/BekkkEvrika/pageSDK"
)

func main() {
	app := pagesdk.New(pagesdk.Config{
		PageInstanceTTL:  30 * time.Minute,
		MaxPageInstances: 10_000,
	})
	if err := app.Run(registerPages, ":8080"); err != nil {
		panic(err)
	}
}

func registerPages(app *pagesdk.Application) {
	app.Manifest().Register("users.edit", NewUsersEditPage)
	app.Manifest().Register("users.list", NewUsersPage)
}
```

`Run` обрабатывает CLI-команды и без аргументов запускает `Bootstrap`.
`Bootstrap` выполняет три операции:

1. вызывает функцию регистрации страниц;
2. получает routes от engine каждой зарегистрированной page;
3. регистрирует routes и запускает HTTP server.

## 4. CLI и access-команды

Если приложение запускается через `app.Run(...)`, собранный сервис понимает
обычный server mode и команды для access manifest:

```bash
./service
./service serve
./service access generate
./service access validate
./service access diff
./service access sync --dry-run
./service access sync
```

Команды:

| Команда | Что делает |
|---|---|
| `serve` | Запускает HTTP server. То же самое происходит, если запустить сервис без аргументов. |
| `access generate` | Регистрирует pages, собирает access manifest и записывает `sfp.access.yaml` или путь из `Config.AccessManifestPath`. |
| `access validate` | Читает manifest и проверяет дубли, parent links и stale references. |
| `access diff` | Сравнивает текущий DSL с manifest и печатает новые/пропавшие legacy `resources`. |
| `access sync --dry-run` | Валидирует manifest и показывает, сколько `accessGroups` будет синхронизировано, без запросов изменения в Keycloak. |
| `access sync` | Валидирует manifest и синхронизирует `accessGroups` с Keycloak как UMA resources. Роли SDK не создаёт. |

`access generate` создаёт два слоя:

- `accessGroups` — новая SFP-модель доступов. Именно они синхронизируются в
  Keycloak как UMA resources.
- `resources` — legacy слой для старых event/table/button ключей. Он пока
  сохраняется для совместимости и local diff.

Page groups создаются автоматически для каждой зарегистрированной page:

```yaml
accessGroups:
  - code: page.users.edit
    name: Page users.edit
    type: page
    enabled: true
```

UI/action groups регистрируются в центральном Go registry. Это источник истины
для списка access groups:

```go
var ClientCardEditing = pagesdk.AccessGroup{
	Code: "client.card.editing",
	Name: "Редактирование карточки клиента",
}

func RegisterAccess(app *pagesdk.Application) {
	_ = app.RegisterAccessGroup(ClientCardEditing)
}
```

`Type`, `Enabled` и `ParentCode` можно не указывать в коде примера:
SDK заполнит `Type` как `ui_group` и `Enabled: true`. `ParentCode` нужен
только если вы хотите явно отразить иерархию access groups в manifest.

Pages только используют уже объявленные группы:

```go
func (p *ClientCardPage) Init(ctx *engine.BuildContext) error {
	p.Text("client.name").
		Access(accessdefs.ClientCardEditing, pagesdk.NoAccessReadonly)

	p.Button("save").
		Access(accessdefs.ClientCardEditing, pagesdk.NoAccessHidden)

	return nil
}
```

Для TableEngine доступны table, columns и actions:

```go
func (p *ClientsListPage) Init(ctx *engine.BuildContext) error {
	p.Table("clients").
		Access(accessdefs.ClientTableActions, pagesdk.NoAccessHidden).
		Columns(
			p.Column("name").
				Header("Name").
				Access(accessdefs.ClientTableActions, pagesdk.NoAccessHidden),
			p.Column("status").
				Header("Status").
				AddActionBuilder(
					p.Action("approve", p.onApprove).
						Access(accessdefs.ClientTableActions, pagesdk.NoAccessHidden),
				),
		).
		ToolbarActions(
			p.Action("export", p.onExport).
				Access(accessdefs.ClientTableActions, pagesdk.NoAccessHidden),
		).
		RowActions(
			p.Action("delete", p.onDelete).
				Access(accessdefs.ClientTableActions, pagesdk.NoAccessRemove),
		).
		SelectedActions(
			p.Action("archive", p.onArchive).
				Access(accessdefs.ClientTableActions, pagesdk.NoAccessHidden),
		)

	return nil
}
```

Пример готового manifest с группировками вынесен в отдельный файл:
[docs/examples/sfp.access.example.yaml](examples/sfp.access.example.yaml).

В нём видно, как группировать access groups:

- `accessGroups` отвечают на вопрос “какая бизнес-группа доступа существует?”;
- `elements` генерируются SDK из `.Access(...)` в DSL и отвечают на вопрос
  “какие UI элементы меняются при отсутствии этой группы?”;
- роли, policies и назначение пользователей остаются в Keycloak и создаются
  администратором, не SDK.

Если page ссылается на группу, которой нет в registry, `access generate`
завершится ошибкой:

```text
unknown access group "client.card.edting" referenced by element "save"
```

Это защищает от опечаток: SDK не создаёт новую access group из случайной
строки в UI element.

Для `access sync` нужны Keycloak-настройки в `pagesdk.Config` или env:

```text
KEYCLOAK_BASE_URL=http://IP:8081
KEYCLOAK_REALM=sfp
KEYCLOAK_CLIENT_ID=gateway
KEYCLOAK_CLIENT_SECRET=...
KEYCLOAK_SYNC_ENABLED=true
```

`access sync` получает service account token через token endpoint и создаёт
или обновляет только access groups в Keycloak. UI elements из `elements`, роли
и назначения пользователей в Keycloak не отправляются.

## 5. Manifest и page keys

Manifest связывает стабильный ключ страницы с фабрикой:

```go
app.Manifest().Register("users.edit", NewUsersEditPage)
```

Ключ участвует в URL:

```text
GET /page/users.edit
```

Рекомендации для ключей:

- используйте стабильные domain-oriented имена: `users.list`, `users.edit`,
  `admin.roles`;
- не включайте ID пользователя, язык, дату или фильтр;
- не меняйте ключ без миграции frontend/navigation links;
- не регистрируйте один ключ дважды.

Повторная регистрация вызывает panic во время startup. Это намеренное поведение:
duplicate page key является ошибкой конфигурации.

Фабрика обязана возвращать новый экземпляр:

```go
func NewUsersEditPage() engine.Page {
	return &UsersEditPage{
		FormEngine: &formengine.FormEngine{},
	}
}
```

Нельзя возвращать один глобальный Page:

```go
// Не делайте так.
var shared = &UsersEditPage{FormEngine: &formengine.FormEngine{}}

func NewUsersEditPage() engine.Page {
	return shared
}
```

## 6. Контракт Page

Каждая page реализует:

```go
type Page interface {
	Init(ctx *engine.BuildContext) error
	GetEngine() engine.Engine
}
```

Обычно `GetEngine()` получается через embedding:

```go
type UsersEditPage struct {
	*formengine.FormEngine
}
```

или:

```go
type UsersPage struct {
	*tableengine.TableEngine
}
```

`Init` должен:

- построить DSL;
- зарегистрировать event handlers;
- задать initial data/state;
- при необходимости использовать `BuildContext`.

`Init` не предназначен для runtime mutations. Он вызывается:

- на временном sample Page во время bootstrap для route discovery;
- на новом пользовательском Page во время каждого render-запроса.

На event-запросах `Init` не вызывается: Application находит сохранённый Page по
`pageInstanceId` и использует уже зарегистрированные handlers и DSL.

Следствие: route topology должна быть детерминированной. Не регистрируйте
handlers только для конкретного пользователя или query param:

```go
// Плохо: route может отсутствовать во время bootstrap.
if ctx.Params["can_save"] == "true" {
	p.Button("save").OnClick(onSave)
}
```

Лучше всегда зарегистрировать route, а authorization проверить в handler или
изменить доступность/visibility DSL-элемента.

## 7. Lifecycle

### 7.1 Bootstrap

Во время bootstrap:

```text
Application.Bootstrap
  -> registerPages
  -> Manifest.All
  -> PageFactory
  -> Page.Init(empty BuildContext)
  -> Engine.Routes
  -> register routes in Gin
```

### 7.2 Render request

```text
GET /page/{pageKey}
  -> новая Page
  -> новый Engine
  -> Page.Init(request BuildContext)
  -> создать криптографически случайный instanceId
  -> привязать instanceId к опубликованным event URLs
  -> сохранить Page в in-memory manager
  -> Engine.Render
  -> RenderResult
```

### 7.3 Event request

```text
POST /event/...?pageInstanceId={instanceId}
  -> найти Page instance по instanceId и pageKey
  -> проверить TTL
  -> заблокировать instance на время event
  -> найти handler, зарегистрированный при render
  -> создать typed RuntimeContext
  -> вызвать handler
  -> RuntimeResult
  -> обновить время последней активности
```

`Init` при event отсутствует намеренно. Поэтому request-specific DSL,
созданный для одного пользователя, не пересобирается и не заменяется DSL
другого пользователя.

### 7.4 Закрытие и expiration

Render response содержит `instanceUrl`. Frontend должен вызвать его методом
`DELETE`, когда страница окончательно закрыта:

```http
DELETE /page/users.edit/instance?pageInstanceId={instanceId}
```

Если явное закрытие не пришло, instance удаляется лениво после периода
бездействия. Проверка expired instances выполняется при следующем обращении к
manager, отдельного фонового cleanup goroutine нет.

Defaults:

```go
pagesdk.Config{
	PageInstanceTTL:  30 * time.Minute,
	MaxPageInstances: 10_000,
}
```

Нулевые или отрицательные значения заменяются defaults.

### 7.5 Что хранится на backend

В instance хранятся:

- конкретный `Page` и его `Engine`;
- DSL, построенный этим Page;
- form/table handlers;
- время создания и последней активности;
- mutex, сериализующий events одного instance.

Не используйте поля Page как замену browser state для:

- выбранные строки;
- текущие значения полей;
- pagination state;
- данные открытого dialog;
- navigation stack.

Эти значения могут изменяться на frontend без обращения к backend, поэтому
авторитетным источником event state остаётся payload клиента. В полях Page
допустимо хранить request-specific immutable configuration и зависимости,
созданные для данного render. Не храните там открытые transactions или другие
ресурсы, требующие короткого request scope.

### 7.6 Что изменилось относительно прежнего lifecycle

Раньше Application создавал новый Page и повторно вызывал `Init` на каждом
event. Это приводило к повторной загрузке данных, пересборке DSL и потере
связи с конкретным render пользователя.

Теперь:

| Раньше | Теперь |
|---|---|
| Page создавался на каждый HTTP request | Page создаётся на каждый render |
| `Init` вызывался на render и event | `Init` вызывается на sample bootstrap Page и на render |
| Event заново собирал DSL | Event использует DSL и handlers сохранённого instance |
| URL содержал только статический path | Path прежний, instance добавлен через query |
| Не было явного close lifecycle | Render возвращает `instanceUrl` для `DELETE` |
| Events не координировались на уровне instance | Events одного instance сериализуются mutex-ом |

Access manifest не меняется из-за instance ID: collector анализирует только
стабильные route paths, а `pageInstanceId` появляется позже в runtime URL.

Для backend-кода основной migration обычно не требуется: factories, `Init` и
handlers сохраняют прежние signatures. Frontend обязан перейти на полные URL
из DSL и обрабатывать `instanceId`/`instanceUrl`.

## 8. BuildContext

`Page.Init` получает:

```go
type BuildContext struct {
	User   engine.User
	System engine.SystemKeys
	Params engine.Params
}
```

- `User` — claims аутентифицированного пользователя;
- `System` — стабильные системные ключи;
- `Params` — route/query/page parameters.

Пример initial value из query:

```go
func (p *UsersEditPage) Init(ctx *engine.BuildContext) error {
	p.Text("name").
		Label("Name").
		DefaultValue(ctx.Params["name"])
	return nil
}
```

Текущий встроенный `Application` создает пустые `User` и `System`. Если
приложению нужна authentication integration, transport layer должен заполнить
эти значения до вызова engine.

## 9. FormEngine: начало работы

Импорты:

```go
import (
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/formengine"
	inputs "github.com/BekkkEvrika/pageSDK/form"
)
```

Page:

```go
type UsersEditPage struct {
	*formengine.FormEngine
}

func NewUsersEditPage() engine.Page {
	return &UsersEditPage{
		FormEngine: &formengine.FormEngine{},
	}
}
```

### 9.1 Fluent builder

Предпочтительный стиль:

```go
func (p *UsersEditPage) Init(ctx *engine.BuildContext) error {
	p.Text("name").
		Label("User name").
		Placeholder("Enter user name").
		DefaultValue(ctx.Params["name"]).
		OnChange(onNameChange)

	p.Text("email").
		Label("Email").
		DataType("string")

	p.Checkbox("active").
		Label("Active").
		DefaultValue(true)

	p.Text("status").
		Label("Status").
		ReadOnly(true)

	p.Button("save").
		Label("Save").
		Variant("primary").
		OnClick(onSave)

	return nil
}
```

Builder изменяет тот же DSL, что и старые `Set...` methods. Оба стиля
совместимы:

```go
name := p.Text("name")
name.SetLabel("User name")
name.SetPlaceholder("Enter user name")
name.SetOnChange(onNameChange)
```

### 9.2 Фабрики controls

`FormEngine` предоставляет:

```go
p.Text("name")
p.Select("role")
p.Date("birthday")
p.Datetime("createdAt")
p.Number("age")
p.Checkbox("active")
p.Label("description")
p.Search("manager")
p.Textarea("comment")
p.Hidden("userId")
p.File("avatar")
p.Button("save")
```

Все controls, кроме специального `Label`, поддерживают общий fluent API:

```go
Name(string)
Label(string)
ActionID(string)
Variant(string)
FromName(string)
ReadOnly(bool)
Placeholder(string)
Validation(*inputs.FieldValidation)
MetaData(string)
MetaKey(string)
Format(string)
Options(inputs.ComboItems)
Visible(bool)
FieldActions([]inputs.FieldAction)
FileConfig(*inputs.FileConfig)
ColSpan(int)
Hint(string)
SearchName(string)
DefaultValue(any)
Search(string)
DataType(string)
OnChange(formengine.ChangeListener)
OnClick(formengine.ClickListener)
```

Для обычных input controls используйте `OnChange`; для button — `OnClick`.
API технически общий, но frontend должен поддерживать выбранную комбинацию
control type и trigger.

`Label` намеренно предоставляет только `Input`, `SetLabel` и fluent `Label`.
У label нет runtime value.

### 9.3 Select options

```go
p.Select("role").
	Label("Role").
	Options(inputs.ComboItems{
		{ID: "admin", Text: "Administrator"},
		{ID: "editor", Text: "Editor"},
	})
```

Runtime handler может заменить options уже существующего select:

```go
func onCountryChange(ctx *formengine.RuntimeContext) {
	country, err := ctx.GetSelectById("country")
	if err != nil {
		return
	}
	city, err := ctx.GetSelectById("city")
	if err != nil {
		return
	}

	city.SetOptions(loadCities(country.Element().Value))
	city.SetValue("")
}
```

Frontend получит две mutations:

```json
{
  "mutations": [
    {
      "type": "update",
      "path": "controls.city.options",
      "value": [
        {"value": "dushanbe", "label": "Dushanbe"},
        {"value": "khujand", "label": "Khujand"}
      ]
    },
    {
      "type": "update",
      "path": "controls.city.value",
      "value": ""
    }
  ]
}
```

Полный runnable example зарегистрирован как `controls.combos` в
[`cmd/pagesdk-example/main.go`](../cmd/pagesdk-example/main.go).

### 9.4 Validation

```go
minLength := 3
maxLength := 100

p.Text("name").
	Label("Name").
	Validation(&inputs.FieldValidation{
		MinLength: &minLength,
		MaxLength: &maxLength,
		Message:   "Name is invalid",
	})
```

Validation является частью DSL. Ее исполнение зависит от frontend-клиента.
Критические бизнес-правила все равно проверяйте в backend handler.

### 9.5 File configuration

```go
p.File("documents").
	Label("Documents").
	FileConfig(&inputs.FileConfig{
		Accept:       ".pdf,.png",
		MaxSizeBytes: 10 * 1024 * 1024,
		MaxFiles:     5,
		UploadURL:    "/api/files",
	})
```

### 9.6 Полный Form DSL

Builder удобен для default container. Для сложной вложенной layout-структуры
можно передать полный DSL:

```go
p.CreateForm(inputs.Form{
	Containers: &[]inputs.Container{
		{
			Key:         "main",
			Direction:   "vertical",
			Gap:         16,
			GridColumns: 2,
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
save.OnClick(onSave)
```

`SetForm` является alias-подходом к замене текущего DSL. `Container` добавляет
top-level container, а `Field` добавляет raw input в default container.

### 9.7 Typed build-time getters

Для уже существующего DSL:

```go
p.GetSelectById(id)
p.GetDateById(id)
p.GetDatetimeById(id)
p.GetTextById(id)
p.GetNumberById(id)
p.GetCheckboxById(id)
p.GetLabelById(id)
p.GetSearchById(id)
p.GetTextareaById(id)
p.GetHiddenById(id)
p.GetFileById(id)
p.GetButtonById(id)
```

Getter возвращает error, если:

- ID отсутствует;
- control имеет другой type.

Не игнорируйте такую ошибку в `Init`: это schema/configuration error.

## 10. Form events и routes

`OnClick` и `OnChange` одновременно:

- регистрируют handler;
- создают статический route;
- добавляют route metadata в `dsl.actions`.

Пример:

```go
p.Button("save").OnClick(onSave)
p.Text("name").OnChange(onNameChange)
```

Routes:

```text
POST /event/{pageKey}/button/save?pageInstanceId={instanceId}
POST /event/{pageKey}/text/name?pageInstanceId={instanceId}
```

Frontend не должен строить эти URL из соглашений. Он использует `url` и
`method`, опубликованные в action metadata. `pageInstanceId` находится только
в query-параметре: path остаётся стабильным для access control и route
discovery.

## 11. Form RuntimeContext

Handler signature:

```go
func onSave(ctx *formengine.RuntimeContext)
```

Runtime context содержит:

```go
PageKey
PageInstanceID
User
System
Params
Extra
FormState
Sender
Mutations
Navigation
Dialogs
```

### 11.1 Чтение runtime values

```go
func onSave(ctx *formengine.RuntimeContext) {
	name, err := ctx.GetTextById("name")
	if err != nil {
		return
	}

	value := name.Value
	element := name.Element()
	props := name.Props

	_, _, _ = value, element, props
}
```

`Element()` объединяет DSL metadata и присланное frontend runtime state.
Если frontend не прислал value, используется `DefaultValue` DSL-контрола.

Runtime getters имеют те же typed варианты, что и build-time getters.

### 11.2 Mutations

Backend не вычисляет diff автоматически. Изменения нужно записывать явно:

```go
status, err := ctx.GetTextById("status")
if err != nil {
	return
}

status.SetValue("Saved")
status.SetLabel("Current status")
status.SetHint("Updated just now")
status.SetVisibility(true)
```

Response:

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

Динамическое изменение структуры DSL через events больше не является основной
моделью. Предпочитайте фиксированную структуру, созданную в `Init`, и mutations
существующих controls. `Add`/`Remove` пока остаются для обратной совместимости:

Добавление control:

```go
ctx.Form().Add(inputs.Input{
	Id:    "note",
	Type:  inputs.InputTypeText,
	Label: "Note",
})
```

Удаление существующего control:

```go
ctx.Remove("note")
// или
ctx.Form().Remove("note")
```

Runtime mutation существующего control проверяет, что control присутствует в
DSL. Ошибка сохраняется в context и event request завершается ошибкой.

### 11.3 Ошибки handler

```go
func onSave(ctx *formengine.RuntimeContext) {
	if !canSave(ctx.User) {
		ctx.SetError(errors.New("permission denied"))
		return
	}
}
```

Context сохраняет первую ошибку. После handler engine вернет ее transport
layer. Встроенный `Application` отвечает HTTP `500` с JSON:

```json
{"error":"permission denied"}
```

Сейчас библиотека не разделяет domain errors на `4xx` и internal errors:
пользователь интеграции должен учитывать это при проектировании API.

## 12. Navigation

Form handler может:

```go
ctx.OpenPage("users.details")
ctx.OpenDialog("users.picker")
ctx.OpenTab("admin.roles")
ctx.Close()
ctx.CloseWithResult(map[string]any{"saved": true})
```

Параметры открытия передаются через `OpenOptions`:

```go
ctx.OpenDialog("users.picker", formengine.OpenOptions{
	Extra: map[string]any{
		"group_id": 10,
	},
	Callback: onUserSelected,
})
```

Frontend:

1. открывает указанную page;
2. хранит callback URL;
3. передает `extra` как page/query params;
4. при закрытии дочерней page отправляет result на callback URL.

Callback:

```go
func onUserSelected(ctx *formengine.RuntimeContext) {
	selected, err := ctx.GetTextById("selectedUser")
	if err != nil {
		return
	}
	selected.SetValue(ctx.Extra["user_id"])
}
```

Имена callback routes выводятся из имен Go-функций. Для стабильности
используйте именованные package-level functions, а не динамические closures.

## 13. Dialogs

Простые helpers:

```go
ctx.ShowMessage("Message", "Operation completed")
ctx.ShowWarning("Warning", "Check entered values")
ctx.ShowError("Error", "Operation failed")
ctx.ShowSuccess("Saved", "User was saved")
```

Dialogs с ответом:

```go
ctx.ShowYesNo("Delete", "Delete selected user?", func(value string) {
	if value == "yes" {
		// Выполнить действие.
	}
})

ctx.ShowOKCancel("Confirm", "Continue?", func(value string) {
	// value: "ok" или "cancel"
})
```

Custom dialog:

```go
ctx.ShowDialog(engine.Dialog{
	Title:       "Choose action",
	Description: "What should happen next?",
	Level:       engine.DialogWarning,
	Actions: []engine.DialogAction{
		{Name: "Retry", Value: "retry"},
		{Name: "Ignore", Value: "ignore"},
	},
}, func(value string) {
	// Handle selected action.
})
```

Dialog callback в текущей реализации хранится в process memory, его URL
содержит `pageInstanceId`, а handler удаляется после первого вызова. Для
нескольких application replicas потребуется sticky routing: сам Page instance
и callback registry находятся в памяти конкретного процесса.

## 14. TableEngine: начало работы

Импорты:

```go
import (
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/tableengine"
)
```

Page:

```go
type UsersPage struct {
	*tableengine.TableEngine
}

func NewUsersPage() engine.Page {
	return &UsersPage{
		TableEngine: &tableengine.TableEngine{},
	}
}
```

Полный базовый пример:

```go
func (p *UsersPage) Init(ctx *engine.BuildContext) error {
	p.Table("users").
		Title("Users").
		RowIDKey("id").
		Columns(
			p.Column("id").
				Header("ID").
				DataType(tableengine.TableColumnDataTypeNumber).
				Hidden(true).
				Hideable(false),
			p.Column("name").
				Header("Name").
				Searchable(true).
				Sortable(true),
			p.Column("email").
				Header("Email").
				Searchable(true),
			p.Column("status").
				Header("Status").
				CellType(tableengine.TableColumnCellTypeBadge).
				ValueStyle("active", tableengine.TableCellVariantSuccess).
				ValueStyle("inactive", tableengine.TableCellVariantDanger).
				Filterable(true),
		).
		Data(loadUsers(0, 20)).
		Features(tableengine.TableFeatureConfig{
			Sorting:      true,
			Filtering:    true,
			Pagination:   true,
			RowSelection: true,
		}).
		Selection(tableengine.TableSelectionSchema{
			Mode:     tableengine.TableSelectionModeMultiple,
			Checkbox: true,
		}).
		OnReload(onUsersReload).
		OnFilter(onUsersFilter).
		OnPagination(onUsersPagination)

	return nil
}
```

## 15. Table builder

Основные методы table builder:

```go
Title(string)
RequestURL(string)
RowIDKey(string)
EmptyMessage(string)
SubRowsKey(string)
SubRowsRequestURL(string)
Columns(...*ColumnBuilder)
Features(TableFeatureConfig)
Selection(TableSelectionSchema)
State(TableStateConfig)
Data(any)
OnReload(handler)
OnFilter(handler)
OnPagination(handler)
ToolbarAction(schema, handler)
ToolbarActions(...*ActionBuilder)
RowAction(schema, handler)
SelectedAction(schema, handler)
```

`Table("users")` устанавливает stable table ID. Он используется:

- в DSL;
- в event URLs;
- в mutation path `tables.users.data`;
- frontend state registry.

## 16. Table columns

Создание accessor column:

```go
p.Column("name").
	Header("User name").
	AccessorKey("name").
	Sortable(true).
	Filterable(true).
	Searchable(true).
	Resizable(true).
	Width(240).
	MinWidth(120).
	MaxWidth(400).
	Align(tableengine.TableColumnAlignLeft).
	DataType(tableengine.TableColumnDataTypeString).
	CellType(tableengine.TableColumnCellTypeText)
```

Display-only column:

```go
p.DisplayColumn("actions", "Actions").
	CellType(tableengine.TableColumnCellTypeActions)
```

Поддерживаемые data types:

```text
string, number, boolean, date, datetime, currency,
image, file, status, custom
```

Поддерживаемые cell types:

```text
text, number, boolean, date, datetime, currency, badge,
status, image, file, link, actions, custom
```

### 16.1 Hidden и hideable

```go
p.Column("id").
	Hidden(true).
	Hideable(false)
```

- `Hidden(true)` задает initial visibility;
- `Hideable(false)` запрещает показывать column в UI управления видимостью.

Hidden column остается в schema и row data. Ее можно использовать как row ID и
в action payload.

### 16.2 Value styles

```go
p.Column("status").
	CellType(tableengine.TableColumnCellTypeBadge).
	ValueStyle("active", tableengine.TableCellVariantSuccess).
	ValueStyle("inactive", tableengine.TableCellVariantDanger).
	ValueStyle("pending", tableengine.TableCellVariantWarning)
```

### 16.3 Format

```go
p.Column("amount").
	DataType(tableengine.TableColumnDataTypeCurrency).
	Format(tableengine.TableColumnFormat{
		Type:     tableengine.TableColumnFormatTypeCurrency,
		Currency: "USD",
		Locale:   "en-US",
	})
```

## 17. Table data и state

```go
tableengine.TableData{
	Rows: []map[string]any{
		{"id": 1, "name": "Ada"},
		{"id": 2, "name": "Grace"},
	},
	Total:     2,
	PageIndex: 0,
	PageSize:  20,
}
```

Initial state:

```go
.State(tableengine.TableStateConfig{
	PageIndex: 0,
	PageSize:  20,
	Sorting: []tableengine.TableSortingItem{
		{ID: "name"},
	},
	ColumnVisibility: map[string]bool{
		"id": false,
	},
})
```

Features:

```go
tableengine.TableFeatureConfig{
	Reload:        true,
	Sorting:       true,
	Filtering:     true,
	GlobalSearch:  true,
	Pagination:    true,
	RowSelection:  true,
	ColumnResize:  true,
	VirtualScroll: true,
	SubRows:       true,
}
```

Регистрация `OnReload`, `OnFilter` и `OnPagination` автоматически включает
соответствующие feature flags.

## 18. Table runtime events

### 18.1 Reload, filter и pagination

```go
func onUsersPagination(ctx *tableengine.TableRuntimeContext) {
	data := loadUsers(
		ctx.EventTable.PageIndex,
		ctx.EventTable.PageSize,
	)
	ctx.Table("users").SetData(data)
}
```

Context предоставляет:

```go
ctx.EventTable.TableID
ctx.EventTable.Event
ctx.EventTable.PageIndex
ctx.EventTable.PageSize
ctx.EventTable.Filters
ctx.State
ctx.Params
ctx.Extra
ctx.User
ctx.System
```

### 18.2 Несколько toolbar actions

```go
.ToolbarActions(
	p.Action("refresh", onRefresh).
		Label("Refresh").
		Icon("refresh").
		Variant(tableengine.ActionVariantSecondary).
		Hotkey("F5"),
	p.Action("clear", onClear).
		Label("Clear").
		Icon("trash").
		Variant(tableengine.ActionVariantDanger),
)
```

Каждый action получает отдельный route:

```text
POST /event/{pageKey}/table/{tableID}/toolbar/refresh?pageInstanceId={instanceId}
POST /event/{pageKey}/table/{tableID}/toolbar/clear?pageInstanceId={instanceId}
```

Body toolbar action игнорируется. Handler получает server/request context, но
не получает table payload.

### 18.3 Row action

```go
.RowAction(tableengine.ActionSchema{
	ID:      "edit",
	Label:   "Edit",
	Icon:    "pencil",
	Variant: tableengine.ActionVariantSecondary,
}, onEdit)
```

Handler:

```go
func onEdit(ctx *tableengine.TableRuntimeContext) {
	row := ctx.EventTable.Row
	ctx.OpenDialog("users.edit", engine.Params{
		"id": fmt.Sprint(row["id"]),
	})
}
```

Frontend отправляет:

```json
{
  "row": {
    "id": 7,
    "name": "Ada"
  }
}
```

Row должен содержать значение `RowIDKey`.

### 18.4 Column action

```go
p.Column("email").
	AddAction(onNormalizeEmails, "normalize_emails")
```

Handler получает значения column по row ID:

```go
func onNormalizeEmails(ctx *tableengine.TableRuntimeContext) {
	values := ctx.EventTable.Column
	_ = values["7"]
}
```

Payload:

```json
{
  "column": {
    "7": "ADA@EXAMPLE.COM",
    "8": "GRACE@EXAMPLE.COM"
  }
}
```

### 18.5 Selected action

```go
.SelectedAction(tableengine.ActionSchema{
	ID:      "delete_selected",
	Label:   "Delete selected",
	Icon:    "trash",
	Variant: tableengine.ActionVariantDanger,
}, onDeleteSelected)
```

Handler:

```go
func onDeleteSelected(ctx *tableengine.TableRuntimeContext) {
	selected := ctx.EventTable.SelectedRows
	_ = selected
}
```

Payload:

```json
{
  "selectedRows": ["7", "8"]
}
```

## 19. Table mutations и navigation

Обновление visible data:

```go
ctx.Table("users").SetData(data)
```

Mutation:

```json
{
  "type": "update",
  "path": "tables.users.data",
  "value": {
    "rows": [],
    "total": 0,
    "pageIndex": 0,
    "pageSize": 20
  }
}
```

Navigation:

```go
ctx.OpenDialog("users.edit", engine.Params{"id": "7"})
ctx.OpenTab("users.audit", engine.Params{"id": "7"})
ctx.Close()
ctx.CloseWithResult(map[string]any{"updated": true})
```

Error:

```go
ctx.SetError(errors.New("could not load users"))
```

В текущем TableRuntimeContext dialogs отсутствуют; используйте navigation или
расширяйте engine contract при необходимости.

## 20. HTTP contract

Render response:

```json
{
  "pageKey": "users.list",
  "instanceId": "generated-instance-id",
  "instanceUrl": "/page/users.list/instance?pageInstanceId=generated-instance-id",
  "engine": "table",
  "dsl": {}
}
```

Runtime response:

```json
{
  "mutations": [],
  "navigation": [],
  "dialogs": [],
  "result": null
}
```

Пустые поля имеют `omitempty` и могут отсутствовать.

Встроенное приложение:

- отвечает `200` при успешном handler;
- отвечает `400`, если event/close request не содержит `pageInstanceId`;
- отвечает `404`, если instance не найден или принадлежит другому page key;
- отвечает `410`, если instance найден, но его idle TTL истёк;
- отвечает `503`, если достигнут `MaxPageInstances`;
- отвечает `500` и `{"error":"..."}` при ошибке;
- читает query parameters в `RequestContext.Query` и `Params`;
- не устанавливает auth claims автоматически.

Точный frontend payload/route contract описан в
[client-events.md](client-events.md).

## 21. Page instances и зависимости

Page instance живёт от render до explicit close или idle expiration.
Зависимости по-прежнему передают через factory:

```go
type UsersPage struct {
	*tableengine.TableEngine
	users UserRepository
}

func NewUsersPage(users UserRepository) engine.PageFactory {
	return func() engine.Page {
		return &UsersPage{
			TableEngine: &tableengine.TableEngine{},
			users:       users,
		}
	}
}
```

Регистрация:

```go
app.Manifest().Register("users.list", NewUsersPage(usersRepository))
```

Repository может быть shared и concurrency-safe. Events одного Page instance
сериализуются mutex-ом, но разные instances выполняются параллельно. Shared
dependencies поэтому всё равно обязаны быть concurrency-safe.

Factory должна возвращать новый Page на каждый render. Один и тот же Page
нельзя возвращать нескольким пользователям: их DSL и handlers смешаются.

Handlers могут быть methods:

```go
func (p *UsersPage) Init(ctx *engine.BuildContext) error {
	p.Table("users").
		Columns(p.Column("id"), p.Column("name")).
		OnReload(p.onReload)
	return nil
}

func (p *UsersPage) onReload(ctx *tableengine.TableRuntimeContext) {
	data, err := p.users.List(ctx.EventTable.PageIndex, ctx.EventTable.PageSize)
	if err != nil {
		ctx.SetError(err)
		return
	}
	ctx.Table("users").SetData(data)
}
```

## 22. Authorization

Не полагайтесь только на visibility кнопки или отсутствие frontend element.
Если элемент помечен через `.Access(group, ...)`, SDK проверяет эту группу не
только при render-фильтрации DSL, но и перед выполнением event handler.

Это значит:

- page group проверяется до `Init`; без доступа SDK возвращает `403` и не
  строит DSL;
- form control event с `AccessGroupCode` проверяется перед `OnClick`/`OnChange`;
- table reload/filter/pagination проверяются по access group всей таблицы;
- table toolbar/row/selected/column actions проверяются по access group самого
  action;
- если access group отсутствует, SDK возвращает `403`, и handler не
  выполняется;
- event без `.Access(...)` остаётся доступным после обычных проверок JWT,
  page access и owner `pageInstanceId`.

Handler всё равно должен проверять бизнес-правила: tenant, ownership, row ID,
актуальное состояние entity и другие данные, которые нельзя доверять frontend
payload.

### JWT authentication

pageSDK может проверять Keycloak/OIDC RS256 access tokens через realm JWKS:

```go
authenticator := pagesdk.NewKeycloakJWTAuthenticator(pagesdk.KeycloakJWTConfig{
	KeycloakURL:     "https://keycloak.example.com",
	Realm:           "main",
	Audience:        "page-api",
	AuthorizedParty: "frontend",
	ClockSkew:       30 * time.Second,
})

application := pagesdk.New(pagesdk.Config{
	Authenticator: authenticator,
})
```

Проверяются подпись RS256 и `kid` через JWKS, `iss`, обязательные `sub` и
`exp`, а также `nbf`, `iat`, настроенные `aud` и `azp`.

Проверенные claims доступны при построении страницы:

```go
func (p *UsersPage) Init(ctx *engine.BuildContext) error {
	userID, _ := ctx.User["sub"].(string)
	username, _ := ctx.User["preferred_username"].(string)
	_, _ = userID, username
	return nil
}
```

И внутри event handler через `ctx.User`.

Page instance закрепляется за `{iss}|{sub}`. Новый JWT после refresh может
использовать тот же instance, если issuer и subject не изменились. Токен
проверяется на каждом запросе и не хранится внутри instance.

```go
func (p *UsersPage) onDelete(ctx *tableengine.TableRuntimeContext) {
	if !hasPermission(ctx.User, "users.delete") {
		ctx.SetError(errors.New("permission denied"))
		return
	}
	// Delete...
}
```

Frontend payload недоверенный:

- проверяйте row ID;
- повторно загружайте entity из repository;
- проверяйте ownership/tenant;
- не доверяйте price, role, permission или status из row payload;
- валидируйте callback result.

`pageInstanceId` также нельзя считать authorization token. При настроенном
`Authenticator` transport привязывает instance к проверенному owner и
отклоняет event/delete другого пользователя как `404`. При отсутствии
`Authenticator` остаётся legacy-режим без этой защиты.

## 23. Testing

### 23.1 Обычный unit test handler

Handlers работают с typed context и могут тестироваться без HTTP:

```go
func TestOnRefresh(t *testing.T) {
	ctx := &tableengine.TableRuntimeContext{
		EventTable: &tableengine.TableEventContext{
			TableID: "users",
		},
	}

	onRefresh(ctx)

	if len(ctx.Mutations) != 1 {
		t.Fatalf("mutations = %d, want 1", len(ctx.Mutations))
	}
}
```

### 23.2 Schema test

Создайте page, вызовите `Init`, затем проверьте `DSL()`:

```go
page := &UsersPage{TableEngine: &tableengine.TableEngine{}}
if err := page.Init(&engine.BuildContext{}); err != nil {
	t.Fatal(err)
}

dsl := page.DSL().(tableengine.TableSchema)
if dsl.ID != "users" {
	t.Fatalf("table id = %q", dsl.ID)
}
```

### 23.3 Полный набор тестов библиотеки

```bash
go test ./...
```

## 24. Совместимость builder и setter API

Builder API добавлен поверх прежней модели:

```go
p.Text("name").
	Label("Name").
	Placeholder("Enter name")
```

Эквивалент:

```go
name := p.Text("name")
name.SetLabel("Name")
name.SetPlaceholder("Enter name")
```

Для tables:

```go
p.Column("name").
	Header("Name").
	Sortable(true)
```

Эквивалент:

```go
column := p.Column("name")
column.SetHeader("Name")
column.SetSortable(true)
```

Существующий код не требуется переписывать. Для нового кода рекомендуется
fluent builder, потому что он компактнее и лучше показывает структуру DSL.

## 25. Частые ошибки

### Handler route отсутствует

Причина: listener не был зарегистрирован в `Init`.

```go
p.Button("save").OnClick(onSave)
```

### Typed getter возвращает ошибку

Причина: control отсутствует или имеет другой type.

```go
checkbox, err := ctx.GetCheckboxById("active")
if err != nil {
	return
}
```

### Изменение не появилось на frontend

Изменение local Go variable не создает mutation. Используйте runtime methods:

```go
status.SetValue("Saved")
```

### Table action вызывает неправильный handler

Frontend должен использовать точный URL из DSL. Нельзя отправлять один общий
action route с динамическим `actionId`.

### Toolbar body

Toolbar handler не использует client payload. Тело запроса разрешено, но
игнорируется.

### Pagination сбрасывается

Возвращайте актуальные `PageIndex` и `PageSize` внутри `TableData`.

### Runtime state исчезает между запросами

Объект Page теперь сохраняется между render и events. Если исчезли текущие
значения controls, selection или pagination, проверьте event payload: browser
state не восстанавливается автоматически из Page.

### Event отвечает `pageInstanceId is required`

Frontend вызвал path вручную или потерял query-параметр. Используйте полный URL
из DSL, callback или dialog action, не реконструируйте его.

### Event отвечает `404` или `410`

- `404` — instance не существует, был закрыт, относится к другой page или
  процесс был перезапущен;
- `410` — idle TTL instance истёк.

Frontend должен прекратить отправку events и заново выполнить render страницы.

### Route зависит от request params

Routes обнаруживаются при bootstrap с пустым `BuildContext`. Handler topology
должна быть одинаковой для всех requests.

## 26. Рекомендованная структура приложения

```text
internal/
├── pages/
│   ├── users_list.go
│   ├── users_edit.go
│   └── roles_list.go
├── service/
│   └── users.go
├── repository/
│   └── users.go
└── bootstrap/
    └── pages.go
```

Регистрация:

```go
func RegisterPages(app *pagesdk.Application, deps Dependencies) {
	app.Manifest().Register("users.list", pages.NewUsersPage(deps.Users))
	app.Manifest().Register("users.edit", pages.NewUsersEditPage(deps.Users))
}
```

Старайтесь разделять:

- DSL construction в `Init`;
- UI event orchestration в handlers;
- бизнес-правила в services;
- persistence в repositories;
- frontend protocol implementation в клиентском renderer.

## 27. Production checklist

Перед production:

- все page keys стабильны и уникальны;
- все factories возвращают новый Page/Engine;
- route topology не зависит от request;
- `PageInstanceTTL` и `MaxPageInstances` соответствуют ожидаемой нагрузке;
- frontend сохраняет `instanceId`/`instanceUrl` для каждого открытого окна;
- frontend вызывает `instanceUrl` при окончательном закрытии страницы;
- deployment с несколькими replicas использует sticky sessions;
- handlers проверяют authorization;
- instance URL не используется как единственная проверка authorization;
- frontend использует URLs из DSL;
- client payload считается недоверенным;
- repositories и shared services concurrency-safe;
- ошибки логируются без sensitive data;
- dialogs/navigation callbacks проверены для deployment topology;
- pagination/filter payload валидируется;
- UI mutations применяются frontend в исходном порядке;
- есть schema и handler tests;
- `go test ./...` проходит.

## 28. Следующие документы

- Backend API и patterns: этот документ.
- Архитектура и lifecycle in-memory Page:
  [Page instances](page-instances.md).
- Точные URLs, payloads и frontend behavior:
  [Client Event Protocol](client-events.md).
- Рабочие примеры:
  [users_edit.go](../page/users_edit.go) и
  [users_page.go](../page/users_page.go).
