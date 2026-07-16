package itemcatalog

import (
	"github.com/tinywasm/form/input"
	"github.com/tinywasm/model"
)

var (
	Text_FieldText     = input.Text()
	Textarea_FieldText = input.Textarea()
	Decimal_FieldFloat = input.Decimal()
	Checkbox_FieldBool = input.Checkbox()
	Number_FieldInt    = input.Number()

	BaseText_FieldText = model.Text()
	BaseBool_FieldBool = model.Bool()
	BaseInt_FieldInt   = model.Int()
)

// CatalogItemModel: producto o servicio. `type` == "S" (servicio) / "P" (producto).
// ⚠️ NO cambies los valores "S"/"P": appointment_booking.ServiceExists depende de type == "S".
var CatalogItemModel = model.Definition{
	Name: "catalog_item",
	Fields: model.Fields{
		{Name: "id", Type: Text_FieldText, DB: &model.FieldDB{PK: true}},
		{Name: "tenant_id", Type: Text_FieldText, NotNull: true},
		{Name: "sku", Type: Text_FieldText, NotNull: true},
		{Name: "name", Type: Text_FieldText, NotNull: true},
		{Name: "description", Type: Textarea_FieldText},
		{Name: "category", Type: Text_FieldText},
		{Name: "type", Type: Text_FieldText, NotNull: true}, // "S" servicio / "P" producto
		{Name: "price", Type: Decimal_FieldFloat, NotNull: true},
		{Name: "currency", Type: Text_FieldText, NotNull: true},
		{Name: "is_active", Type: Checkbox_FieldBool, NotNull: true},
		{Name: "updated_at", Type: Number_FieldInt, NotNull: true},
	},
}

// AgreementModel (convenio): un item tiene N convenios. Cada uno con su aseguradora, su código
// (ex-fonasa_code) y su tarifa propia. FK a catalog_item.
var AgreementModel = model.Definition{
	Name: "catalog_agreement",
	Fields: model.Fields{
		{Name: "id", Type: Text_FieldText, DB: &model.FieldDB{PK: true}},
		{Name: "tenant_id", Type: Text_FieldText, NotNull: true},
		{Name: "catalog_item_id", Type: BaseText_FieldText, NotNull: true}, // FK simple (fallback)
		{Name: "insurer", Type: Text_FieldText, NotNull: true}, // aseguradora: "FONASA", "Isapre X"
		{Name: "code", Type: Text_FieldText},                   // ex fonasa_code: código de facturación del convenio
		{Name: "price", Type: Decimal_FieldFloat},               // tarifa propia del convenio (opcional)
		{Name: "is_active", Type: Checkbox_FieldBool, NotNull: true},
		{Name: "updated_at", Type: Number_FieldInt, NotNull: true},
	},
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

	// Agreement methods
	ListAgreements(tenantId, catalogItemId string) ([]Agreement, error)
	UpsertAgreement(Agreement) (Agreement, error)
	DeleteAgreement(tenantId, id string) error
}

var ItemFilterModel = model.Definition{
	Name: "item_filter",
	Fields: model.Fields{
		{Name: "type", Type: BaseText_FieldText},
		{Name: "active_only", Type: BaseBool_FieldBool},
		{Name: "limit", Type: BaseInt_FieldInt},
		{Name: "offset", Type: BaseInt_FieldInt},
	},
}

var ListItemsArgsModel = model.Definition{
	Name: "list_items_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: BaseText_FieldText},
		{Name: "type", Type: BaseText_FieldText},
		{Name: "active_only", Type: BaseBool_FieldBool},
		{Name: "limit", Type: BaseInt_FieldInt},
		{Name: "offset", Type: BaseInt_FieldInt},
	},
}

var GetItemArgsModel = model.Definition{
	Name: "get_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: BaseText_FieldText},
		{Name: "id", Type: BaseText_FieldText},
	},
}

var FindBySKUArgsModel = model.Definition{
	Name: "find_by_sku_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: BaseText_FieldText},
		{Name: "sku", Type: BaseText_FieldText},
	},
}

var DeactivateItemArgsModel = model.Definition{
	Name: "deactivate_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: BaseText_FieldText},
		{Name: "id", Type: BaseText_FieldText},
	},
}

var DeleteItemArgsModel = model.Definition{
	Name: "delete_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: BaseText_FieldText},
		{Name: "id", Type: BaseText_FieldText},
	},
}

var ListAgreementsArgsModel = model.Definition{
	Name: "list_agreements_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: BaseText_FieldText},
		{Name: "catalog_item_id", Type: BaseText_FieldText},
	},
}

var DeleteAgreementArgsModel = model.Definition{
	Name: "delete_agreement_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: BaseText_FieldText},
		{Name: "id", Type: BaseText_FieldText},
	},
}
