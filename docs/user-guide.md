# Руководство пользователя pageSDK

Это руководство предназначено для Go-разработчиков, которые подключают
`pageSDK` как библиотеку и создают на ней server-driven страницы.

Frontend-разработчикам нужен отдельный документ:
[Client Event Protocol](client-events.md).

## 1. Что представляет собой pageSDK

`pageSDK` — backend-библиотека для server-driven UI.

Обычный frontend хранит структуру страницы и большую часть UI-логики у себя. В
pageSDK структура страницы создается backend-кодом:

1. Go page строит декларативный DSL.
2. Frontend получает DSL по HTTP и отрисовывает его.
3. Пользователь вызывает событие.
4. Frontend отправляет текущее состояние на статический event route.
5. Backend handler возвращает явные mutations, navigation или dialogs.
6. Frontend применяет response.

```text
Go Page.Init
    │
    ▼
Render DSL ───────────────► Frontend renderer
                               │
                               │ click/change/table event
                               ▼
Runtime handler ◄──────── Event HTTP request
    │
    └──────── mutations/navigation/dialogs ───────► Frontend
```

Библиотека не хранит browser UI state между запросами. Page и Engine являются
stateless с точки зрения жизненного цикла приложения.

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

import pagesdk "github.com/BekkkEvrika/pageSDK"

func main() {
	app := pagesdk.New()
	if err := app.Bootstrap(registerPages, ":8080"); err != nil {
		panic(err)
	}
}

func registerPages(app *pagesdk.Application) {
	app.Manifest().Register("users.edit", NewUsersEditPage)
	app.Manifest().Register("users.list", NewUsersPage)
}
```

`Bootstrap` выполняет три операции:

1. вызывает функцию регистрации страниц;
2. получает routes от engine каждой зарегистрированной page;
3. регистрирует routes и запускает HTTP server.

## 4. Manifest и page keys

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

## 5. Контракт Page

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

`Init` не предназначен для runtime mutations. Он вызывается при bootstrap для
route discovery и повторно на каждом request.

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

## 6. Lifecycle

### 6.1 Bootstrap

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

### 6.2 Render request

```text
GET /page/{pageKey}
  -> новая Page
  -> новый Engine
  -> Page.Init(request BuildContext)
  -> Engine.Render
  -> RenderResult
```

### 6.3 Event request

```text
POST /event/...
  -> новая Page
  -> новый Engine
  -> Page.Init(request BuildContext)
  -> найти handler статического route
  -> создать typed RuntimeContext
  -> вызвать handler
  -> RuntimeResult
```

Не сохраняйте между запросами:

- выбранные строки;
- текущие значения полей;
- pagination state;
- данные открытого dialog;
- request-specific repositories или transactions.

Для долговременных данных используйте database/service layer. Для UI state
используйте payload клиента и runtime context.

## 7. BuildContext

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

## 8. FormEngine: начало работы

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

### 8.1 Fluent builder

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

### 8.2 Фабрики controls

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

### 8.3 Select options

```go
p.Select("role").
	Label("Role").
	Options(inputs.ComboItems{
		{ID: "admin", Text: "Administrator"},
		{ID: "editor", Text: "Editor"},
	})
```

### 8.4 Validation

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

### 8.5 File configuration

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

### 8.6 Полный Form DSL

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

### 8.7 Typed build-time getters

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

## 9. Form events и routes

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
POST /event/{pageKey}/button/save
POST /event/{pageKey}/text/name
```

Frontend не должен строить эти URL из соглашений. Он использует `url` и
`method`, опубликованные в action metadata.

## 10. Form RuntimeContext

Handler signature:

```go
func onSave(ctx *formengine.RuntimeContext)
```

Runtime context содержит:

```go
PageKey
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

### 10.1 Чтение runtime values

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

### 10.2 Mutations

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

### 10.3 Ошибки handler

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

## 11. Navigation

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

## 12. Dialogs

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

Dialog callback в текущей реализации хранится в process memory и удаляется
после первого вызова. Для нескольких application instances потребуется sticky
routing или внешний callback registry.

## 13. TableEngine: начало работы

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

## 14. Table builder

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

## 15. Table columns

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

### 15.1 Hidden и hideable

```go
p.Column("id").
	Hidden(true).
	Hideable(false)
```

- `Hidden(true)` задает initial visibility;
- `Hideable(false)` запрещает показывать column в UI управления видимостью.

Hidden column остается в schema и row data. Ее можно использовать как row ID и
в action payload.

### 15.2 Value styles

```go
p.Column("status").
	CellType(tableengine.TableColumnCellTypeBadge).
	ValueStyle("active", tableengine.TableCellVariantSuccess).
	ValueStyle("inactive", tableengine.TableCellVariantDanger).
	ValueStyle("pending", tableengine.TableCellVariantWarning)
```

### 15.3 Format

```go
p.Column("amount").
	DataType(tableengine.TableColumnDataTypeCurrency).
	Format(tableengine.TableColumnFormat{
		Type:     tableengine.TableColumnFormatTypeCurrency,
		Currency: "USD",
		Locale:   "en-US",
	})
```

## 16. Table data и state

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

## 17. Table runtime events

### 17.1 Reload, filter и pagination

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

### 17.2 Несколько toolbar actions

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
POST /event/{pageKey}/table/{tableID}/toolbar/refresh
POST /event/{pageKey}/table/{tableID}/toolbar/clear
```

Body toolbar action игнорируется. Handler получает server/request context, но
не получает table payload.

### 17.3 Row action

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

### 17.4 Column action

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

### 17.5 Selected action

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

## 18. Table mutations и navigation

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

## 19. HTTP contract

Render response:

```json
{
  "pageKey": "users.list",
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
- отвечает `500` и `{"error":"..."}` при ошибке;
- читает query parameters в `RequestContext.Query` и `Params`;
- не устанавливает auth claims автоматически.

Точный frontend payload/route contract описан в
[client-events.md](client-events.md).

## 20. Stateless design и зависимости

Page instance живет один request, поэтому зависимости обычно передают через
factory:

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

Repository может быть shared и concurrency-safe. Request-specific mutable state
не должен сохраняться в shared dependency без синхронизации.

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

## 21. Authorization

Не полагайтесь только на visibility кнопки или отсутствие frontend element.
Authorization должна выполняться на backend в каждом чувствительном handler:

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

## 22. Testing

### 22.1 Обычный unit test handler

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

### 22.2 Schema test

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

### 22.3 Полный набор тестов библиотеки

```bash
go test ./...
```

## 23. Совместимость builder и setter API

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

## 24. Частые ошибки

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

Это ожидаемо: Page stateless. Состояние должно находиться на frontend или в
постоянном backend storage.

### Route зависит от request params

Routes обнаруживаются при bootstrap с пустым `BuildContext`. Handler topology
должна быть одинаковой для всех requests.

## 25. Рекомендованная структура приложения

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

## 26. Production checklist

Перед production:

- все page keys стабильны и уникальны;
- все factories возвращают новый Page/Engine;
- route topology не зависит от request;
- handlers проверяют authorization;
- frontend использует URLs из DSL;
- client payload считается недоверенным;
- repositories и shared services concurrency-safe;
- ошибки логируются без sensitive data;
- dialogs/navigation callbacks проверены для deployment topology;
- pagination/filter payload валидируется;
- UI mutations применяются frontend в исходном порядке;
- есть schema и handler tests;
- `go test ./...` проходит.

## 27. Следующие документы

- Backend API и patterns: этот документ.
- Точные URLs, payloads и frontend behavior:
  [Client Event Protocol](client-events.md).
- Рабочие примеры:
  [users_edit.go](../page/users_edit.go) и
  [users_page.go](../page/users_page.go).
