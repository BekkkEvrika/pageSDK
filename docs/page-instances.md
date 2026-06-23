# Page instances: lifecycle и архитектура

Этот документ описывает lifecycle объекта `Page` после перехода от
«новая Page на каждый request» к in-memory instances.

Практическое использование backend API описано в
[руководстве пользователя](user-guide.md), frontend-протокол — в
[Client Event Protocol](client-events.md).

## Зачем понадобился instance

`Page.Init(ctx)` может строить разный DSL в зависимости от query parameters,
claims пользователя и системного контекста. Например, два render-запроса одной
страницы могут получить разные initial values, visibility и data:

```text
GET /page/users.edit?id=10
GET /page/users.edit?id=20
```

Если event создаёт новый Page и повторно вызывает `Init`, backend:

- повторно загружает данные;
- может построить DSL уже с другим контекстом;
- теряет тот объект, на котором были зарегистрированы handlers;
- не работает с конкретным render, который видит пользователь.

Поэтому render теперь создаёт самостоятельный Page instance, а events
обращаются именно к нему.

## Два уровня объектов

Manifest по-прежнему хранит фабрики, а не пользовательские Page:

```text
Manifest
└── pageKey -> PageFactory
```

In-memory manager хранит уже открытые экземпляры:

```text
PageInstanceManager
└── instanceId
    ├── pageKey
    ├── Page
    ├── Engine + DSL + handlers
    ├── createdAt
    ├── lastAccess
    └── mutex
```

Один глобальный Page в Manifest использовать нельзя. Иначе разные
пользователи и вкладки будут изменять один DSL и один набор handlers.

## Render lifecycle

```text
GET /page/{pageKey}
  1. Manifest находит PageFactory.
  2. Factory создаёт новый Page и Engine.
  3. Application создаёт случайный instanceId.
  4. Engine вызывает Page.Init(request BuildContext).
  5. Init строит request-specific DSL и регистрирует handlers.
  6. Engine добавляет pageInstanceId в query всех runtime URLs.
  7. Application сохраняет Page в in-memory manager.
  8. Frontend получает DSL, instanceId и instanceUrl.
```

Пример ответа:

```json
{
  "pageKey": "users.edit",
  "instanceId": "Hq8W...generated-id",
  "instanceUrl": "/page/users.edit/instance?pageInstanceId=Hq8W...generated-id",
  "engine": "form",
  "dsl": {
    "actions": [
      {
        "id": "save",
        "config": {
          "url": "/event/users.edit/button/save?pageInstanceId=Hq8W...generated-id",
          "method": "POST"
        }
      }
    ]
  }
}
```

`instanceId` генерируется из 24 cryptographically random bytes и кодируется
через URL-safe base64 без padding.

## Event lifecycle

```text
POST /event/{stable-path}?pageInstanceId={instanceId}
  1. Application читает instanceId из query.
  2. Manager находит instance.
  3. Manager проверяет pageKey и idle TTL.
  4. Instance блокируется mutex-ом.
  5. Engine находит handler, зарегистрированный во время render.
  6. Handler выполняется без Page.Init.
  7. RuntimeResult возвращается frontend.
  8. Instance разблокируется.
```

Path event route не содержит instance ID. Это сохраняет стабильность:

- Gin route topology;
- access manifest keys;
- API gateway policies;
- route-level authorization rules;
- observability grouping.

Instance передаётся только через query:

```text
/event/users.edit/button/save?pageInstanceId=...
```

## Почему `Init` всё ещё вызывается во время bootstrap

Engines должны заранее зарегистрировать статические Gin routes. Для route
discovery Application создаёт временный sample Page и вызывает его `Init` с
пустым `BuildContext`.

Sample Page:

- не сохраняется в instance manager;
- не обслуживает пользовательские events;
- нужен только для обнаружения route topology и access resources.

Поэтому набор handlers/routes должен быть детерминированным. Request-specific
DSL может отличаться, но event route не должен существовать только у одного
пользователя.

## Concurrency

Manager использует два уровня синхронизации:

- общий mutex защищает map instances и timestamps;
- отдельный mutex каждого instance сериализует его events.

Следствия:

- два events одного открытого окна выполняются последовательно;
- разные tabs/users/instances могут выполняться параллельно;
- shared repositories и services всё равно должны быть concurrency-safe;
- handler не должен бесконечно блокироваться, потому что следующий event этого
  instance будет ждать.

## TTL и limits

Конфигурация:

```go
pagesdk.Config{
	PageInstanceTTL:  30 * time.Minute,
	MaxPageInstances: 10_000,
}
```

Defaults применяются, если значение нулевое или отрицательное:

- idle TTL: 30 минут;
- общий limit: 10 000 instances на процесс.

TTL считается от `LastAccess`. Успешный acquire для event обновляет
`LastAccess`.

Cleanup ленивый:

- expired target удаляется при event acquire;
- все expired instances очищаются перед добавлением нового instance;
- отдельного background sweeper сейчас нет.

## Явное закрытие

Frontend получает готовый `instanceUrl` и вызывает:

```http
DELETE /page/{pageKey}/instance?pageInstanceId={instanceId}
```

Успешный ответ — `204 No Content`.

Явное закрытие важно, когда TTL большой или пользователи часто открывают
тяжёлые страницы. Если закрытие не было отправлено, TTL остаётся fallback.

## HTTP errors

| Status | Причина |
|---|---|
| `400` | query не содержит `pageInstanceId` |
| `404` | instance отсутствует, закрыт или имеет другой page key |
| `410` | instance найден, но idle TTL истёк |
| `503` | достигнут `MaxPageInstances` во время render |
| `500` | ошибка `Init`, handler или runtime engine |

После `404`/`410` frontend должен выполнить новый render. Старый instance ID
повторно использовать нельзя.

## Что хранить в Page

Подходящие данные:

- request-specific DSL;
- зарегистрированные handlers;
- immutable configuration render-а;
- references на shared services/repositories;
- вычисленные данные, которые нужны handlers этого instance.

Не следует считать Page источником истины для:

- текущих значений inputs;
- table selection;
- pagination/filter state;
- navigation stack;
- данных, изменяемых в browser без event.

Эти значения frontend отправляет в event payload.

Не храните в Page долгоживущую database transaction, request body stream или
другой ресурс, рассчитанный на завершение одного HTTP request.

## Form, table и callbacks

Instance query добавляется во все backend-generated runtime URLs:

- form click/change actions;
- table reload/filter/pagination;
- toolbar/row/column/selected actions;
- dialog callbacks;
- navigation callbacks.

Navigation callback принадлежит parent instance. Когда child page закрывается,
frontend вызывает callback URL, сохранённый вместе с navigation action. Query
в этом URL указывает на parent, а не на child.

## Deployment limitations

Page содержит Go functions и pointers, поэтому instance нельзя напрямую
сериализовать в Redis или database.

Текущая реализация рассчитана на один процесс. При нескольких replicas нужны
sticky sessions, чтобы render и все последующие events попали в один process.

После restart/deploy все in-memory instances исчезают. Frontend получит `404`
и должен заново render-ить страницу.

`pageInstanceId` не является authorization token. Встроенный Application пока
не сохраняет owner ID, а default `User` пуст. Gateway/auth integration должны
проверять доступ к page/event routes и не допускать использование чужого URL.

## Migration checklist

Backend:

- factories продолжают возвращать новый Page;
- `Init` продолжает строить DSL и регистрировать handlers;
- handlers больше не должны рассчитывать на повторный `Init` перед event;
- route topology остаётся детерминированной;
- выбрать подходящие `PageInstanceTTL` и `MaxPageInstances`.

Frontend:

- сохранять `instanceId` и `instanceUrl` отдельно для каждого render;
- вызывать полный URL из DSL, не теряя query;
- не переносить event URLs между tabs/instances;
- вызывать `instanceUrl` при окончательном закрытии;
- на `404`/`410` discard-ить instance и render-ить заново;
- не retry-ить автоматически non-idempotent event после expiration.

Operations:

- использовать sticky sessions при нескольких replicas;
- учитывать memory cost открытых Page instances;
- отслеживать количество `404`, `410` и `503`;
- ожидать invalidation всех instances при restart/deploy.
