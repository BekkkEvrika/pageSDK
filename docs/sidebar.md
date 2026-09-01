# Публикация sidebar

pageSDK хранит описание sidebar и строит полное дерево, но не зависит от
RabbitMQ или другого транспорта. Конкретный способ публикации предоставляет
приложение через интерфейс `SidebarPublisher`.

## Объявление меню

Sidebar nodes объявляются как стабильные package variables рядом с access
groups:

```go
var (
	OrdersViewing = pagesdk.AccessGroup{
		Code: "orders.viewing",
		Name: "Просмотр заказов",
	}
	OrdersListViewing = pagesdk.AccessGroup{
		Code: "orders.list.viewing",
		Name: "Просмотр списка заказов",
	}

	OrdersMenu = pagesdk.SidebarNode{
		Key:         "orders",
		Title:       "Заказы",
		AccessGroup: OrdersViewing,
		Order:       10,
	}
	OrdersListMenu = pagesdk.SidebarNode{
		Key:         "orders.list",
		Title:       "Список заказов",
		ParentKey:   OrdersMenu.Key,
		AccessGroup: OrdersListViewing,
		Order:       10,
	}
)
```

Групповой node не закрепляется за page. Его `target` остаётся пустым, а дерево
строится по `ParentKey`.

Access groups и sidebar nodes регистрируются один раз на Application:

```go
app := pagesdk.New(pagesdk.Config{
	Module: "orders",
	Sidebar: pagesdk.SidebarConfig{
		ServiceID:  "orders-service",
		SectionKey: "sales",
		Publisher:  publisher,
	},
}, func(app *pagesdk.Application) error {
	for _, group := range []pagesdk.AccessGroup{OrdersViewing, OrdersListViewing} {
		if err := app.RegisterAccessGroup(group); err != nil {
			return err
		}
	}
	return app.RegisterSidebarNodes(OrdersMenu, OrdersListMenu)
})
```

`ServiceID` должен быть стабильным идентификатором сервиса. `SectionKey`
указывает на заранее созданную секцию в sidebar registry. Один service ID
следует использовать только для одной секции.

## Привязка к page

Страница прикрепляет зарегистрированный node через общий для FormEngine и
TableEngine метод `Sidebar`:

```go
func (p *OrdersListPage) Init(ctx *engine.BuildContext) error {
	p.Sidebar(OrdersListMenu)

	p.Table("orders").Columns(
		p.Column("id").Header("ID"),
	)
	return nil
}
```

SDK строит `target` из module и page key. Для `Module: "orders"` и page
`orders.list` получится:

```text
/orders/page/orders.list
```

Sidebar metadata должна быть статичной. Не делай вызов `Sidebar(...)`
зависимым от текущего пользователя или request params: при bootstrap SDK
создаёт sample page с пустым `BuildContext` для сбора metadata.

## Publisher

Транспортный контракт состоит из одного метода:

```go
type SidebarPublisher interface {
	PublishSidebar(
		ctx context.Context,
		action pagesdk.SidebarAction,
		event pagesdk.SidebarEvent,
	) error
}
```

Поддерживаются действия:

- `pagesdk.SidebarRegistration`;
- `pagesdk.SidebarRefresh`;
- `pagesdk.SidebarUnregister`.

При обычном `Application.Run`/`Bootstrap` SDK автоматически публикует
`registration` перед запуском HTTP server, если зарегистрирован хотя бы один
sidebar node. Ошибка publisher останавливает запуск и возвращается вызывающему
коду.

Полный актуальный snapshot можно опубликовать явно:

```go
err := app.PublishSidebar(ctx, pagesdk.SidebarRefresh)
```

При окончательном выводе сервиса из эксплуатации:

```go
err := app.PublishSidebar(ctx, pagesdk.SidebarUnregister)
```

SDK намеренно не отправляет `unregister` при обычной остановке или рестарте,
чтобы меню не исчезало между экземплярами сервиса.

## Реализация RabbitMQ-адаптера

Адаптер сериализует полученный `SidebarEvent` и преобразует action в требования
конкретной инфраструктуры. Для sfpRegistry он должен публиковать:

```text
Exchange: VKEvents
Routing key: ""
Content-Type: application/json
Headers:
  version: "1.0.0"
  registration|refresh|unregister: "1"
```

pageSDK не содержит этих значений и не импортирует RabbitMQ client. Поэтому
тот же механизм можно подключить к RabbitMQ, Kafka, HTTP или тестовому in-memory
publisher без изменений SDK.

Для небольших адаптеров доступен `SidebarPublisherFunc`:

```go
publisher := pagesdk.SidebarPublisherFunc(
	func(ctx context.Context, action pagesdk.SidebarAction, event pagesdk.SidebarEvent) error {
		return transport.Publish(ctx, string(action), event)
	},
)
```

## Валидация

Перед `registration` и `refresh` SDK проверяет:

- непустые `ServiceID` и `SectionKey`;
- наличие хотя бы одного node;
- уникальные и непустые node keys;
- непустые title и access group;
- регистрацию каждой access group в Application;
- существование каждого `ParentKey` и отсутствие циклов;
- существование node, указанного страницей;
- что leaf node закреплён за page;
- что один node не закреплён за разными pages.

Nodes на каждом уровне сортируются сначала по `Order`, затем по `Key`.
Публикуемый `accessKey` равен `AccessGroup.Code`, а `children` всегда содержит
массив, включая пустой массив у leaf nodes.
