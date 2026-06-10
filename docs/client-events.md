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
```

Application does not know event structure. Concrete Engine defines:

- component types
- supported actions
- event route patterns
- runtime payload format
- mutation/navigation response format

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
      "visibilityRules": [],
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
  visibilityRules?: unknown[]
  fieldActions?: unknown[]
  fileConfig?: unknown
  colSpan?: number
  hint?: string
  searchObject?: string
  defaultValue?: string
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

Event response contains explicit patch mutations and navigation actions.

```json
{
  "mutations": [
    {
      "type": "update",
      "path": "controls.status.text",
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
      "type": "openDialog",
      "page": "users.edit",
      "params": {
        "id": "42"
      }
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
  navigation?: NavigationItem[]
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

Navigation item:

```ts
type NavigationItem = {
  type: "openPage" | "closePage" | "openDialog" | "openTab" | "closeWithResult"
  page?: string
  params?: Record<string, string>
  result?: unknown
}
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
  "path": "controls.title.text",
  "value": "Saved"
}
```

Client should resolve `path` and update that property.

Common paths:

```text
controls.{id}.text
controls.{id}.label
controls.{id}.value
controls.{id}.visible
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

Supported types:

- `openDialog`
- `openTab`
- `closePage`
- `closeWithResult`

Examples:

```json
{
  "type": "openDialog",
  "page": "users.edit",
  "params": {
    "id": "42"
  }
}
```

```json
{
  "type": "openTab",
  "page": "analytics.dashboard"
}
```

```json
{
  "type": "closePage"
}
```

```json
{
  "type": "closeWithResult",
  "result": {
    "saved": true
  }
}
```

For `closeWithResult`, the frontend owns navigation stack and callback routes. Backend remains stateless.

## Client responsibilities

The client must:

- render container hierarchy recursively
- preserve stable node ids
- call only backend-provided event URLs
- send form state payload for form events
- apply mutations in order
- keep navigation state separately from UI mutations
- protect against double-click/race conditions when needed
- handle `4xx/5xx` event failures without corrupting UI state

## Backend guarantees

The backend guarantees:

- event routes are static after startup
- routes are deterministic for a given manifest/page implementation
- handlers are registered inside Engine, not Page
- event handlers can mutate only existing DSL controls through runtime getters
- runtime controls cannot register new event handlers
- Page and Engine are recreated per request
- no diff engine is used
- all UI changes are explicit mutations/navigation actions

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
