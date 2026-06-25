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
- [Пример зависимых combo boxes](cmd/pagesdk-example/main.go)

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

## JWT authentication и владельцы page instances

Для Keycloak/OIDC можно включить встроенную проверку RS256 JWT через JWKS:

```go
authenticator := pagesdk.NewKeycloakJWTAuthenticator(pagesdk.KeycloakJWTConfig{
	KeycloakURL:     "https://keycloak.example.com",
	Realm:           "main",
	Audience:        "page-api",
	AuthorizedParty: "frontend",
})

app := pagesdk.New(pagesdk.Config{
	Authenticator: authenticator,
})
```

После включения `Authenticator` все page, event, callback и instance-delete
routes требуют `Authorization: Bearer <token>`. Проверенные JWT claims
передаются в `BuildContext.User` и runtime context.

Каждый rendered page instance закрепляется за идентичностью
`{iss}|{sub}`. Event с JWT другого пользователя получает `404`, даже если
ему известен `pageInstanceId`. JWT проверяется заново на каждом запросе;
сам access token в page instance не сохраняется.

`cmd/pagesdk-example` читает Keycloak-настройки из env. Runtime auth включается
явно, чтобы пример можно было запускать локально без Keycloak:

```bash
export KEYCLOAK_BASE_URL=http://localhost:8081
export KEYCLOAK_REALM=sfp
export KEYCLOAK_CLIENT_ID=gateway
export KEYCLOAK_CLIENT_SECRET=...
export KEYCLOAK_AUTH_ENABLED=true
export KEYCLOAK_AUDIENCE=gateway

go run ./cmd/pagesdk-example serve
```

Для `access sync` используются те же `KEYCLOAK_BASE_URL`,
`KEYCLOAK_REALM`, `KEYCLOAK_CLIENT_ID`, `KEYCLOAK_CLIENT_SECRET`.

## UI access manifest

После перехода entrypoint с `Bootstrap` на `Run` собранный сервис поддерживает:

```bash
./service
./service serve
./service access generate
./service access validate
./service access diff
./service access sync --dry-run
./service access sync
```

| Команда | Назначение |
|---|---|
| `serve` или запуск без аргументов | Запускает HTTP server. |
| `access generate` | Создаёт/обновляет `sfp.access.yaml`: `accessGroups`, legacy `resources`, `stale`. |
| `access validate` | Проверяет manifest: дубли, parent links и stale references. |
| `access diff` | Сравнивает текущий DSL с manifest и печатает отличия legacy `resources`. |
| `access sync --dry-run` | Валидирует manifest и показывает план без изменений в Keycloak. |
| `access sync` | Синхронизирует `accessGroups` с Keycloak как UMA resources. Роли SDK не создаёт. |

`access generate` сохраняет legacy `resources`, а также новую модель
`accessGroups`. В Keycloak синхронизируются только `accessGroups`; конкретные
кнопки, inputs, blocks, sections и другие UI elements остаются внутри SFP.
Большой пример `sfp.access.yaml` с готовыми группировками есть в
[docs/examples/sfp.access.example.yaml](docs/examples/sfp.access.example.yaml).

Access group регистрируется один раз в central registry:

```go
_ = app.RegisterAccessGroup(pagesdk.AccessGroup{
	Code:        "client.card.editing",
	Name:        "Редактирование карточки клиента",
})
```

В sample package `page/` для этого есть helper:

```go
_ = page.RegisterAccessGroups(app)
```

UI elements не пишутся руками в registry. Они собираются из DSL annotations:

```go
p.Text("client.name").
	Access(accessdefs.ClientCardEditing, pagesdk.NoAccessReadonly)

p.Button("save").
	Access(accessdefs.ClientCardEditing, pagesdk.NoAccessHidden)
```

Для TableEngine работают table, columns и actions:

```go
p.Table("clients").
	Access(accessdefs.ClientTableActions, pagesdk.NoAccessHidden).
	Columns(
		p.Column("name").Access(accessdefs.ClientTableActions, pagesdk.NoAccessHidden),
		p.Column("status").AddActionBuilder(
			p.Action("approve", onApprove).Access(accessdefs.ClientTableActions, pagesdk.NoAccessHidden),
		),
	).
	ToolbarActions(
		p.Action("export", onExport).Access(accessdefs.ClientTableActions, pagesdk.NoAccessHidden),
	).
	RowActions(
		p.Action("delete", onDelete).Access(accessdefs.ClientTableActions, pagesdk.NoAccessRemove),
	).
	SelectedActions(
		p.Action("archive", onArchive).Access(accessdefs.ClientTableActions, pagesdk.NoAccessHidden),
	)
```

Если element ссылается на незарегистрированную group, `access generate`
завершится ошибкой и не создаст новую group из опечатки.

Встроенный `KeycloakUMAProvider` получает service account token через
`/realms/{realm}/protocol/openid-connect/token` и создаёт/обновляет UMA
resources через `/realms/{realm}/authz/protection/resource_set`. В payload
уходит только access group code/name/description/type; `Elements` не
отправляются в Keycloak.

Настройки можно передать через `pagesdk.Config` или env:

```text
KEYCLOAK_BASE_URL=http://IP:8081
KEYCLOAK_REALM=sfp
KEYCLOAK_CLIENT_ID=gateway
KEYCLOAK_CLIENT_SECRET=...
KEYCLOAK_SYNC_ENABLED=true
```

Для runtime-проверки по Keycloak RPT включите RPT authorizer:

```go
app := pagesdk.New(pagesdk.Config{
	Authenticator: pagesdk.NewKeycloakJWTAuthenticator(pagesdk.KeycloakJWTConfig{
		KeycloakURL: "http://IP:8081",
		Realm:       "sfp",
		Audience:    "gateway",
	}),
})

app.UseRPTAccessAuthorizer()
```

В этом режиме SDK не ждёт роли пользователя в JWT. Он читает access groups из
RPT:

```json
{
  "authorization": {
    "permissions": [
      { "resource_set_name": "page.clients.card" },
      { "resource_set_name": "client.card.editing" }
    ]
  }
}
```

Page group проверяется до `Init`; при отсутствии доступа SDK возвращает `403`
и не строит DSL. UI groups применяются после render по полям
`accessGroupCode` и `noAccessBehavior` внутри DSL. Эти же группы проверяются
на backend перед event handler: скрытый form/table action нельзя выполнить
ручным POST, SDK вернёт `403`.

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

В example-приложении доступна страница с combo boxes:

```text
GET /page/controls.combos
```

Она показывает статические options, initial values из query parameters и
runtime-обновление списка городов при смене страны.
