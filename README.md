# pageSDK

`pageSDK` — Go-библиотека для создания server-driven UI. Backend описывает
страницы декларативным DSL, библиотека публикует HTTP routes, frontend
отрисовывает DSL и отправляет пользовательские события обратно.

Библиотека предоставляет два движка:

- `FormEngine` — формы, поля, кнопки, события `click`/`change`, mutations,
  dialogs и navigation;
- `TableEngine` — таблицы, pagination/filter/reload, toolbar actions, row
  actions, column actions и actions над выбранными строками.

`pageSDK` не является готовым frontend-компонентом. Это backend SDK и HTTP
контракт. Клиентское приложение должно уметь отрисовать полученный DSL и
обработать runtime response.

## Документация

- [Полное руководство пользователя](docs/user-guide.md)
- [Протокол frontend-событий](docs/client-events.md)
- [Lifecycle и архитектура page instances](docs/page-instances.md)
- [Пример FormEngine](page/users_edit.go)
- [Пример TableEngine](page/users_page.go)

## Требования

- Go `1.25.4` или совместимая версия;
- HTTP transport предоставляется встроенным приложением на Gin.

## Установка

```bash
go get github.com/BekkkEvrika/pageSDK
```

## Минимальное приложение

```go
package main

import (
	pagesdk "github.com/BekkkEvrika/pageSDK"
	"github.com/BekkkEvrika/pageSDK/engine"
	"github.com/BekkkEvrika/pageSDK/engine/formengine"
)

func main() {
	app := pagesdk.New(pagesdk.Config{
		Module:             "clients",
		AccessManifestPath: "sfp.access.yaml",
	})
	if err := app.Run(registerPages, ":8080"); err != nil {
		panic(err)
	}
}

func registerPages(app *pagesdk.Application) {
	app.Manifest().Register("users.edit", NewUsersEditPage)
}

type UsersEditPage struct {
	*formengine.FormEngine
}

func NewUsersEditPage() engine.Page {
	return &UsersEditPage{
		FormEngine: &formengine.FormEngine{},
	}
}

func (p *UsersEditPage) Init(ctx *engine.BuildContext) error {
	p.Text("name").
		Label("User name").
		Placeholder("Enter user name")

	p.Text("status").
		Label("Status").
		ReadOnly(true)

	p.Button("save").
		Label("Save").
		Variant("primary").
		OnClick(onSave)

	return nil
}

func onSave(ctx *formengine.RuntimeContext) {
	status, err := ctx.GetTextById("status")
	if err != nil {
		return
	}
	status.SetValue("Saved")
}
```

Запуск:

```bash
go run .
```

Получение страницы:

```bash
curl http://localhost:8080/clients/page/users.edit
```

Если `Module` пустой, старые маршруты `/page/...` и `/event/...` остаются
без изменений.

## UI access manifest

После перехода entrypoint с `Bootstrap` на `Run` собранный сервис поддерживает:

```bash
./service serve
./service access generate
./service access validate
./service access diff
./service access sync --dry-run
```

`access generate` собирает page, form event, table event/action и column-view
ключи из зарегистрированного DSL. Повторная генерация сохраняет вручную
созданные `permissionGroups` и описания, а исчезнувшие ключи переносит в
`stale`.

Для реальной синхронизации с Keycloak приложение может установить реализацию
`pagesdk.AccessSyncProvider` через `app.SetAccessSyncProvider`. Встроенный
provider намеренно поддерживает только безопасный `--dry-run` и возвращает
понятную ошибку для реальной синхронизации.

Render response содержит тип движка и DSL:

```json
{
  "pageKey": "users.edit",
  "instanceId": "generated-instance-id",
  "instanceUrl": "/clients/page/users.edit/instance?pageInstanceId=generated-instance-id",
  "engine": "form",
  "dsl": {
    "containers": [],
    "actions": []
  }
}
```

Frontend берет URL события из `dsl.actions`, отправляет состояние формы и
получает явные изменения:

Event URL сохраняет стабильный route path для access control и содержит
`pageInstanceId` только как query-параметр:

```text
/clients/event/users.edit/button/save?pageInstanceId=generated-instance-id
```

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

## Короткий пример таблицы

```go
type UsersPage struct {
	*tableengine.TableEngine
}

func NewUsersPage() engine.Page {
	return &UsersPage{
		TableEngine: &tableengine.TableEngine{},
	}
}

func (p *UsersPage) Init(ctx *engine.BuildContext) error {
	p.Table("users").
		Columns(
			p.Column("id").
				DataType(tableengine.TableColumnDataTypeNumber).
				Hidden(true),
			p.Column("name").Searchable(true),
			p.Column("status").
				CellType(tableengine.TableColumnCellTypeBadge).
				ValueStyle("active", tableengine.TableCellVariantSuccess),
		).
		Data(tableengine.TableData{
			Rows: []map[string]any{
				{"id": 1, "name": "Ada", "status": "active"},
			},
			Total:    1,
			PageSize: 20,
		}).
		OnReload(onReload).
		ToolbarActions(
			p.Action("refresh", onRefresh).
				Icon("refresh").
				Hotkey("F5"),
		)

	return nil
}

func onReload(ctx *tableengine.TableRuntimeContext) {
	ctx.Table("users").SetData(loadUsers())
}

func onRefresh(ctx *tableengine.TableRuntimeContext) {
	ctx.Table("users").SetData(loadUsers())
}
```

## Основная модель

```text
Application
  └── Manifest
      └── page key -> PageFactory
                       └── новая Page на каждый render
                           ├── Init(request context)
                           └── in-memory instance
                               └── events по pageInstanceId
```

Ключевые правила:

- `Page` и `Engine` создаются заново на каждый render-запрос;
- `Page.Init` вызывается при render, строит DSL и регистрирует handlers;
- события используют сохранённый in-memory instance и не вызывают `Init`
  повторно;
- instance удаляется после периода бездействия (по умолчанию 30 минут) или
  через `DELETE /page/{pageKey}/instance?pageInstanceId=...`;
- request-specific DSL и handlers сохраняются внутри Page instance;
- текущие browser values, selection и navigation state остаются на frontend;
- frontend не должен самостоятельно конструировать event URL;
- все event URL публикуются в DSL;
- изменения UI возвращаются явно через `mutations`, `navigation` и `dialogs`;
- frontend остается владельцем текущего визуального и navigation state.

## Структура пакетов

```text
github.com/BekkkEvrika/pageSDK
├── pageSDK                 Application и основные публичные aliases
├── engine                  базовые контракты, contexts и runtime responses
├── engine/formengine       FormEngine и form runtime API
├── engine/tableengine      TableEngine и table runtime API
├── form                    структуры Form DSL и form payload
├── table                   структуры Table DSL и builders
└── manifest                registry page key -> PageFactory
```

## Локальная проверка репозитория

```bash
go test ./...
go run ./cmd/pagesdk-example
```

Дальнейшее чтение лучше начать с
[полного руководства](docs/user-guide.md), а при разработке frontend-клиента —
с [протокола событий](docs/client-events.md).
