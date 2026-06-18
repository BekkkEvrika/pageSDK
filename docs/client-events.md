# Client Event Protocol

This document describes how a frontend client should work with pageSDK event routes.

## Core idea

The backend uses static, deterministic event routes generated during bootstrap.

The client must not invent event routes. It should use the routes exposed in the rendered DSL/action metadata.

Example routes:

```text
GET  /page/users.edit
POST /event/users.edit/button/save
POST /event/users.edit/text/name
POST /event/users.edit/dialog/:dialog
POST /event/users.edit/callback/:callback
```

Application does not know event structure. Concrete Engine defines:

- component types
- supported actions
- event route patterns
- runtime payload format
- mutation/navigation/dialog response format

## Initial render

Client loads a page with:

```http
GET /page/{manifestKey}
```

Example:

```http
GET /page/users.edit
```

Response:

```json
{
  "pageKey": "users.edit",
  "engine": "form",
  "dsl": {
    "containers": [
      {
        "id": "main",
        "direction": "vertical",
        "gap": 16,
        "fields": [
          {
            "id": "name",
            "type": "text",
            "label": "Имя пользователя"
          }
        ]
      }
    ],
    "actions": [
      {
        "id": "save",
        "trigger": "click",
        "config": {
          "type": "apiCall",
          "url": "/event/users.edit/button/save",
          "method": "POST"
        }
      }
    ]
  }
}
```

The client renders the `dsl.containers` tree. Containers may contain:

- `fields`
- nested `containers`

The real UI structure is hierarchical. Field `id` values must be treated as stable node ids.

## Event route discovery

For form pages, event endpoints are defined by action metadata.

For a button click:

```json
{
  "id": "save",
  "trigger": "click",
  "config": {
    "url": "/event/users.edit/button/save",
    "method": "POST"
  }
}
```

The client should call exactly that URL.

Do not build dynamic wildcard URLs such as:

```text
/event/{page}/:component/:action
```

Those are not client-facing routes for `FormEngine`.

## Table event route discovery

For table pages, the rendered DSL exposes the table id and only the registered
runtime event routes under `dsl.events`.

Example:

```json
{
  "pageKey": "users.list",
  "engine": "table",
  "dsl": {
    "id": "users",
    "columns": [],
    "features": {
      "reload": true,
      "filtering": true,
      "pagination": true
    },
    "events": {
      "reload": {
        "url": "/event/users.list/table/users/reload",
        "method": "POST"
      },
      "filter": {
        "url": "/event/users.list/table/users/filter",
        "method": "POST"
      },
      "pagination": {
        "url": "/event/users.list/table/users/pagination",
        "method": "POST"
      }
    }
  }
}
```

Client rules:

- use `dsl.id` as the stable table id;
- call the exact `url` and `method` from `dsl.events`;
- do not construct table event URLs on the frontend;
- an event key is omitted when its backend handler is not registered;
- `features.reload`, `features.filtering`, and `features.pagination` are enabled
  automatically when the corresponding handler is registered.

## Table event payload

Table events use their own typed payload. They do not use the form event
payload and do not send form `elements` or `sender`.

```ts
type TableEventRequest = {
  state?: TableState
  pageIndex?: number
  pageSize?: number
  filters?: TableFilterState[]
  params?: Record<string, unknown>
  extra?: Record<string, unknown>
}

type TableState = {
  pageIndex?: number
  pageSize?: number
  globalFilter?: string
  sorting?: TableSortingItem[]
  filters?: TableFilterState[]
  columnVisibility?: Record<string, boolean>
  selectedRows?: string[]
  columnSizing?: Record<string, number>
}

type TableSortingItem = {
  id: string
  desc?: boolean
}

type TableFilterState = {
  id: string
  value: unknown
  operator?:
    | "eq"
    | "neq"
    | "contains"
    | "startsWith"
    | "endsWith"
    | "gt"
    | "gte"
    | "lt"
    | "lte"
    | "between"
    | "in"
    | "notIn"
}
```

Payload rules:

- every property is optional;
- `state` is the full current table state when the client already maintains it;
- top-level `pageIndex`, `pageSize`, and `filters` override the same values from
  `state`;
- `params` contains request-specific values required by the handler;
- `extra` contains optional client metadata;
- table id and event type are derived from the DSL event URL and must not be
  duplicated in the JSON body.

### Reload event

A reload may send an empty object:

```http
POST /event/users.list/table/users/reload
Content-Type: application/json
```

```json
{}
```

If the current state must be preserved, send it explicitly:

```json
{
  "state": {
    "pageIndex": 1,
    "pageSize": 20,
    "filters": [
      {
        "id": "status",
        "value": "active",
        "operator": "eq"
      }
    ]
  }
}
```

### Filter event

Send the complete active filter list. An empty array means that all filters
were cleared.

```http
POST /event/users.list/table/users/filter
Content-Type: application/json
```

```json
{
  "pageIndex": 0,
  "pageSize": 20,
  "filters": [
    {
      "id": "name",
      "value": "Ali",
      "operator": "contains"
    },
    {
      "id": "status",
      "value": ["active", "pending"],
      "operator": "in"
    }
  ],
  "params": {
    "tenantId": 17
  }
}
```

### Pagination event

`pageIndex` is zero-based. Send the requested page and current filters.

```http
POST /event/users.list/table/users/pagination
Content-Type: application/json
```

```json
{
  "pageIndex": 2,
  "pageSize": 25,
  "filters": [
    {
      "id": "status",
      "value": "active",
      "operator": "eq"
    }
  ],
  "extra": {
    "source": "pagination"
  }
}
```

Inside the backend handler these values are available through the specialized
`TableRuntimeContext`:

```go
func OnUsersPagination(ctx *tableengine.TableRuntimeContext) {
	pageIndex := ctx.EventTable.PageIndex
	pageSize := ctx.EventTable.PageSize
	filters := ctx.EventTable.Filters

	rows, total := loadUsers(pageIndex, pageSize, filters)
	ctx.Table("users").SetData(tableengine.TableData{
		Rows:      rows,
		Total:     total,
		PageIndex: pageIndex,
		PageSize:  pageSize,
	})
}
```

`SetData` returns a normal runtime mutation:

```json
{
  "mutations": [
    {
      "type": "update",
      "path": "tables.users.data",
      "value": {
        "rows": [
          {
            "id": 1,
            "name": "Alice"
          }
        ],
        "total": 42,
        "pageIndex": 2,
        "pageSize": 25
      }
    }
  ]
}
```

The client must replace the table data at the mutation path and must not treat
the response as a form mutation.

## Form event payload

For form events, send the current UI element state as one universal payload shape.
The same payload structure is used for `click`, `change`, and future form events.

```json
{
  "elements": [
    {
      "id": "name",
      "type": "text",
      "name": "name",
      "label": "Имя пользователя",
      "actionId": "",
      "variant": "",
      "fromName": "",
      "readOnly": false,
      "placeholder": "Введите имя",
      "validation": null,
      "metaData": "",
      "metaKey": "",
      "format": "",
      "options": [],
      "visibility": true,
      "fieldActions": [],
      "fileConfig": null,
      "colSpan": 0,
      "hint": "",
      "searchObject": "",
      "defaultValue": "",
      "searchSource": "",
      "dataType": "string",
      "value": "Alice"
    },
    {
      "id": "email",
      "type": "text",
      "name": "email",
      "label": "Email",
      "value": "alice@example.com"
    },
    {
      "id": "save",
      "type": "button",
      "label": "Сохранить",
      "actionId": "save",
      "value": true
    }
  ],
  "sender": {
    "id": "save",
    "type": "button",
    "label": "Сохранить",
    "actionId": "save",
    "value": true
  },
  "actionId": "save",
  "trigger": "click",
  "changedField": "save"
}
```

Fields:

- `elements`: array of all current form/UI elements. Each item may contain every property from the rendered DSL `Input` plus runtime `value`.
- `sender`: the element that triggered this event. It has the same shape as one `elements` item.
- `actionId`: action/control id, for example `save`.
- `trigger`: `click`, `change`, or another engine-supported trigger.
- `changedField`: id of the element that triggered the event.

Element item shape:

```ts
type ElementState = Input & {
  value?: unknown
  props?: Record<string, unknown>
}
```

Known `Input` properties currently include:

```ts
type Input = {
  id: string
  type: string
  name?: string
  label?: string
  actionId?: string
  variant?: string
  fromName?: string
  readOnly?: boolean
  placeholder?: string
  validation?: unknown
  metaData?: string
  metaKey?: string
  format?: string
  options?: Array<{ value: unknown; label: unknown }>
  visibility?: boolean
  fieldActions?: unknown[]
  fileConfig?: unknown
  colSpan?: number
  hint?: string
  searchObject?: string
  defaultValue?: unknown
  searchSource?: string
  dataType?: string
}
```

Extra custom element properties are allowed. Backend stores unknown properties in `ElementState.Props`.

Legacy `fields` map payload is still accepted as a fallback, but frontend should prefer `elements`.

For button click:

```http
POST /event/users.edit/button/save
Content-Type: application/json
```

```json
{
  "elements": [
    {
      "id": "name",
      "type": "text",
      "label": "Имя пользователя",
      "value": "Alice"
    },
    {
      "id": "email",
      "type": "text",
      "label": "Email",
      "value": "alice@example.com"
    },
    {
      "id": "save",
      "type": "button",
      "label": "Сохранить",
      "actionId": "save",
      "value": true
    }
  ],
  "sender": {
    "id": "save",
    "type": "button",
    "label": "Сохранить",
    "actionId": "save",
    "value": true
  },
  "actionId": "save",
  "trigger": "click",
  "changedField": "save"
}
```

For text change:

```http
POST /event/users.edit/text/name
Content-Type: application/json
```

```json
{
  "elements": [
    {
      "id": "name",
      "type": "text",
      "label": "Имя пользователя",
      "value": "Alice"
    },
    {
      "id": "email",
      "type": "text",
      "label": "Email",
      "value": "alice@example.com"
    }
  ],
  "sender": {
    "id": "name",
    "type": "text",
    "label": "Имя пользователя",
    "value": "Alice"
  },
  "actionId": "name",
  "trigger": "change",
  "changedField": "name"
}
```

## Runtime response

Event response contains explicit patch mutations, navigation actions, and client-side dialogs.

```json
{
  "mutations": [
    {
      "type": "update",
      "path": "controls.status.label",
      "value": "Saved"
    },
    {
      "type": "update",
      "path": "controls.lastAction.value",
      "value": "save"
    }
  ],
  "navigation": [
    {
      "type": "open",
      "mode": "dialog",
      "page": "users.picker",
      "extra": {
        "group_id": 10
      },
      "callback": "/event/users.edit/callback/on_user_selected"
    }
  ],
  "dialogs": [
    {
      "title": "Saved",
      "description": "User was saved successfully",
      "level": "success",
      "actions": [
        {
          "name": "OK",
          "value": "ok"
        }
      ]
    }
  ]
}
```

Handlers do not return custom result objects. They only enqueue explicit runtime operations through `RuntimeContext`.

## Patch Response Structure

Runtime event response:

```ts
type RuntimeResult = {
  mutations?: Mutation[]
  navigation?: NavigationAction[]
  dialogs?: Dialog[]
  result?: unknown
}
```

Mutation patch item:

```ts
type Mutation = {
  type: "update" | "add" | "remove"
  path: string
  value?: unknown
}
```

Navigation action:

```ts
type NavigationAction = {
  type: "open" | "close"
  mode?: "page" | "dialog" | "tab"
  page?: string
  extra?: Record<string, unknown>
  callback?: string
  result?: unknown
}
```

Client-side dialog:

```ts
type Dialog = {
  title: string
  description?: string
  level: "info" | "warning" | "error" | "success"
  actions?: DialogAction[]
}

type DialogAction = {
  name: string
  value: string
  url?: string
  method?: string
}
```

## Dialog Protocol

`dialogs` is a separate runtime response block. It is not a mutation and it is not navigation.

The frontend should render every dialog item in order. Dialog `level` controls the visual state:

- `info` - ordinary message
- `warning` - warning message
- `error` - error message
- `success` - success message

Each `actions` item describes one button. The frontend should use:

- `name` as the visible button text
- `value` as the semantic action value
- `url` and `method` only when present

Simple dialogs such as message/warning/error/success usually contain an `OK` action without `url`. In that case, clicking the button only closes the client-side dialog.

Dialogs that require backend handling include `url` and `method` on each action:

```json
{
  "title": "Confirm",
  "description": "Continue?",
  "level": "info",
  "actions": [
    {
      "name": "Yes",
      "value": "yes",
      "url": "/event/users.edit/dialog/dialog-1",
      "method": "POST"
    },
    {
      "name": "No",
      "value": "no",
      "url": "/event/users.edit/dialog/dialog-1",
      "method": "POST"
    }
  ]
}
```

When the user clicks a dialog action with `url`, the frontend sends the clicked action value to that exact URL:

```http
POST /event/users.edit/dialog/dialog-1
Content-Type: application/json

{
  "value": "yes"
}
```

The response is a normal `RuntimeResult`, so the frontend should process `mutations`, `navigation`, and `dialogs` from that response the same way as for form events.

Do not construct dialog callback URLs on the frontend. The route pattern is universal:

```text
POST /event/{pageKey}/dialog/:dialog
```

But the concrete `:dialog` id is runtime-generated by the backend and must be read from `action.url`.

Form backend helpers:

```go
ctx.ShowMessage("Message", "Plain message")
ctx.ShowWarning("Warning", "Check this")
ctx.ShowError("Error", "Something failed")
ctx.ShowSuccess("Saved", "User was saved")
ctx.ShowYesNo("Confirm", "Continue?", func(value string) {
	fmt.Println(value)
})
ctx.ShowOKCancel("Edit", "Save changes?", func(value string) {
	fmt.Println(value)
})

ctx.ShowDialog(engine.Dialog{
	Title:       "Custom",
	Description: "Choose action",
	Level:       engine.DialogWarning,
	Actions: []engine.DialogAction{
		{Name: "Retry", Value: "retry"},
		{Name: "Ignore", Value: "ignore"},
	},
}, func(value string) {
	fmt.Println(value)
})
```

## Mutation Protocol

Supported mutation types:

- `update`
- `add`
- `remove`

Apply mutations in the exact order received.

### update

```json
{
  "type": "update",
  "path": "controls.title.label",
  "value": "Saved"
}
```

Client should resolve `path` and update that property.

Common paths:

```text
controls.{id}.label
controls.{id}.value
controls.{id}.visibility
```

### add

```json
{
  "type": "add",
  "path": "form.controls",
  "value": {
    "id": "dynamic_text",
    "type": "text",
    "label": "Dynamic"
  }
}
```

Client should add the node to the target container/tree location.

Current minimal path:

```text
form.controls
```

Later versions may support deeper paths such as:

```text
containers.main.controls
containers.main.containers.details.controls
```

### remove

```json
{
  "type": "remove",
  "path": "controls.old_button"
}
```

Client should remove the node with that id from the rendered tree.

## Navigation protocol

Navigation is not a mutation. It is applied separately after or alongside mutations, depending on frontend policy.

Supported action types:

- `open`
- `close`

For `open`, `mode` tells the frontend how to present the page:

- `page`
- `dialog`
- `tab`

Examples:

```json
{
  "type": "open",
  "mode": "dialog",
  "page": "users.picker",
  "extra": {
    "group_id": 10
  },
  "callback": "/event/users.edit/callback/on_user_selected"
}
```

```json
{
  "type": "open",
  "mode": "tab",
  "page": "analytics.dashboard"
}
```

```json
{
  "type": "close"
}
```

```json
{
  "type": "close",
  "result": {
    "user_id": 77
  }
}
```

For `open`, the frontend owns opened pages and stores:

- opened page key
- presentation mode
- `extra` params sent to the opened page
- `callback` route, if present

When loading the opened page, send `extra` as query params:

```http
GET /page/users.picker?group_id=10
```

The backend exposes these values to `Page.Init` through `BuildContext.Params`.

For `close` with `result`, the frontend closes the current dialog/tab/page. If the closed page was opened with a `callback` route, the frontend calls that exact route and sends result data:

```http
POST /event/users.edit/callback/on_user_selected
Content-Type: application/json

{
  "result": {
    "user_id": 77
  }
}
```

The response from callback route is a normal `RuntimeResult`, so process `mutations`, `navigation`, and `dialogs` exactly like a form event response.

Do not construct navigation callback URLs on the frontend. The route pattern is universal:

```text
POST /event/{pageKey}/callback/:callback
```

But the concrete `callback` URL is generated by the backend and must be read from `NavigationAction.callback`.

Backend does not store navigation stack. The frontend owns opened pages, callback routes, and extra/result params.

## Client responsibilities

The client must:

- render container hierarchy recursively
- preserve stable node ids
- call only backend-provided event URLs
- call only backend-provided navigation callback URLs
- send form state payload for form events
- apply mutations in order
- keep navigation state separately from UI mutations
- store opened pages, navigation callback routes, and extra/result params client-side
- protect against double-click/race conditions when needed
- handle `4xx/5xx` event failures without corrupting UI state

## Backend guarantees

The backend guarantees:

- event routes are static after startup
- routes are deterministic for a given manifest/page implementation
- navigation callback route pattern is registered once per form page as a universal handler
- handlers are registered inside Engine, not Page
- event handlers can mutate only existing DSL controls through runtime getters
- runtime controls cannot register new event handlers
- Page and Engine are recreated per request
- no diff engine is used
- all UI changes are explicit mutations/navigation/dialog actions

## Important errors

If a Page registers a listener for a missing component:

```go
button, err := p.GetButtonById("save")
if err != nil {
    return err
}
button.SetOnClick(OnSave)
```

and `save` does not exist in the DSL tree, bootstrap fails with an explicit error.

This is intentional. The framework should not silently create missing elements.
