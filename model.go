package itemcatalog

import "github.com/tinywasm/model"

var CatalogItemModel = model.Definition{
	Name: "catalog_item",
	Fields: model.Fields{
		{Name: "id", Type: model.FieldText, DB: &model.FieldDB{PK: true}},
		{Name: "tenant_id", Type: model.FieldText, NotNull: true},
		{Name: "sku", Type: model.FieldText, NotNull: true},
		{Name: "name", Type: model.FieldText, NotNull: true},
		{Name: "description", Type: model.FieldText},
		{Name: "category", Type: model.FieldText},
		{Name: "type", Type: model.FieldText, NotNull: true},
		{Name: "price", Type: model.FieldFloat, NotNull: true},
		{Name: "currency", Type: model.FieldText, NotNull: true},
		{Name: "is_active", Type: model.FieldBool, NotNull: true},
		{Name: "updated_at", Type: model.FieldInt, NotNull: true},
	},
}

// UIAdapter — port for presentation. Implemented by the host app.
// The module calls these methods; it never imports tinywasm/dom or components directly.
type UIAdapter interface {
	RenderItemList(items []CatalogItem, activeFilter string) string
	RenderItemForm(item *CatalogItem) string // nil = empty create form
	RenderFilterSelector(current string) string
}

// EventPublisher — compatible with tinywasm/sse but not coupled to it.
// Pass nil to disable event publishing.
type EventPublisher interface {
	Publish(event string, payload any) error
}

// CatalogService — the core business interface. Implemented by *Module.
type CatalogService interface {
	GetItem(tenantId, id string) (CatalogItem, error)
	FindBySKU(tenantId, sku string) (CatalogItem, error)
	ListItems(tenantId string, filter ItemFilter) ([]CatalogItem, error)
	CreateItem(item CatalogItem) (CatalogItem, error)
	UpdateItem(item CatalogItem) (CatalogItem, error)
	DeactivateItem(tenantId, id string) error
	DeleteItem(tenantId, id string) error
	ServiceExists(tenantId, serviceId string) (bool, error) // implements appointment-booking.CatalogReader
}

var ItemFilterModel = model.Definition{
	Name: "item_filter",
	Fields: model.Fields{
		{Name: "type", Type: model.FieldText},
		{Name: "active_only", Type: model.FieldBool},
		{Name: "limit", Type: model.FieldInt},
		{Name: "offset", Type: model.FieldInt},
	},
}

var ListItemsArgsModel = model.Definition{
	Name: "list_items_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.FieldText},
		{Name: "type", Type: model.FieldText},
		{Name: "active_only", Type: model.FieldBool},
		{Name: "limit", Type: model.FieldInt},
		{Name: "offset", Type: model.FieldInt},
	},
}

var GetItemArgsModel = model.Definition{
	Name: "get_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.FieldText},
		{Name: "id", Type: model.FieldText},
	},
}

var FindBySKUArgsModel = model.Definition{
	Name: "find_by_sku_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.FieldText},
		{Name: "sku", Type: model.FieldText},
	},
}

var DeactivateItemArgsModel = model.Definition{
	Name: "deactivate_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.FieldText},
		{Name: "id", Type: model.FieldText},
	},
}

var DeleteItemArgsModel = model.Definition{
	Name: "delete_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.FieldText},
		{Name: "id", Type: model.FieldText},
	},
}
