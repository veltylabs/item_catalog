//go:build !wasm

package itemcatalog

import (
	"github.com/tinywasm/events"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/model"
	"github.com/tinywasm/orm"
	"github.com/tinywasm/router"
	"github.com/tinywasm/time"
)

var ErrNotFound = fmt.Err("item not found")
var ErrAlreadyExists = fmt.Err("item already exists")

const (
	OpListItems      = "list_catalog_items"
	OpGetItem        = "get_catalog_item"
	OpFindItemBySKU  = "find_item_by_sku"
	OpCreateItem     = "create_catalog_item"
	OpUpdateItem     = "update_catalog_item"
	OpUpsertItem     = "upsert_catalog_item"
	OpDeactivateItem = "deactivate_catalog_item"
	OpDeleteItem     = "delete_catalog_item"

	OpListAgreements  = "list_agreements"
	OpUpsertAgreement = "upsert_agreement"
	OpDeleteAgreement = "delete_agreement"

	TopicItemCreated      = "catalog.item.created"
	TopicItemUpdated      = "catalog.item.updated"
	TopicItemDeactivated  = "catalog.item.deactivated"
	TopicItemDeleted      = "catalog.item.deleted"
	TopicAgreementCreated = "catalog.agreement.created"
	TopicAgreementUpdated = "catalog.agreement.updated"
	TopicAgreementDeleted = "catalog.agreement.deleted"
)

type Deps struct {
	IDs       model.IDGenerator // requerido — el módulo NUNCA construye un generador
	Publisher events.Publisher  // opcional — nil desactiva la publicación de eventos
}

type Module struct {
	db  *orm.DB
	ids model.IDGenerator
	pub events.Publisher
}

func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err("item_catalog: Deps.IDs is required")
	}
	if err := db.CreateTable(&CatalogItem{}); err != nil {
		return nil, err
	}
	if err := db.CreateTable(&Agreement{}); err != nil {
		return nil, err
	}
	return &Module{db: db, ids: deps.IDs, pub: deps.Publisher}, nil
}

// Service methods

func (m *Module) GetItem(tenantId, id string) (CatalogItem, error) {
	var item CatalogItem
	qb := m.db.Query(&item).Where(CatalogItem_.Id).Eq(id).Where(CatalogItem_.TenantId).Eq(tenantId)
	_, err := ReadOneCatalogItem(qb, &item)
	if err != nil {
		if err == orm.ErrNotFound {
			return CatalogItem{}, ErrNotFound
		}
		return CatalogItem{}, err
	}
	return item, nil
}

func (m *Module) FindBySKU(tenantId, sku string) (CatalogItem, error) {
	var item CatalogItem
	qb := m.db.Query(&item).Where(CatalogItem_.Sku).Eq(sku).Where(CatalogItem_.TenantId).Eq(tenantId)
	_, err := ReadOneCatalogItem(qb, &item)
	if err != nil {
		if err == orm.ErrNotFound {
			return CatalogItem{}, ErrNotFound
		}
		return CatalogItem{}, err
	}
	return item, nil
}

func (m *Module) ListItems(tenantId string, filter ItemFilter) ([]CatalogItem, error) {
	var item CatalogItem
	qb := m.db.Query(&item).Where(CatalogItem_.TenantId).Eq(tenantId)
	if filter.Type != "" {
		qb = qb.Where(CatalogItem_.Type).Eq(filter.Type)
	}
	if filter.ActiveOnly {
		qb = qb.Where(CatalogItem_.IsActive).Eq(true)
	}
	if filter.Limit > 0 {
		qb = qb.Limit(int(filter.Limit))
	}
	if filter.Offset > 0 {
		qb = qb.Offset(int(filter.Offset))
	}
	results, err := ReadAllCatalogItem(qb)
	if err != nil {
		return nil, err
	}
	items := make([]CatalogItem, len(results))
	for i, r := range results {
		items[i] = *r
	}
	return items, nil
}

func (m *Module) CreateItem(item CatalogItem) (CatalogItem, error) {
	// Validate SKU uniqueness per tenant
	existing, err := m.FindBySKU(item.TenantId, item.Sku)
	if err == nil && existing.Id != "" {
		return CatalogItem{}, ErrAlreadyExists
	}

	item.Id = m.ids.NewID()
	item.UpdatedAt = time.Now()
	if err := m.db.Create(&item); err != nil {
		return CatalogItem{}, err
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicItemCreated, Payload: &item})
	}
	return item, nil
}

func (m *Module) UpdateItem(item CatalogItem) (CatalogItem, error) {
	// Verify item exists and belongs to tenant
	_, err := m.GetItem(item.TenantId, item.Id)
	if err != nil {
		return CatalogItem{}, err
	}

	item.UpdatedAt = time.Now()
	if err := m.db.Update(&item, orm.Eq(CatalogItem_.Id, item.Id)); err != nil {
		return CatalogItem{}, err
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicItemUpdated, Payload: &item})
	}
	return item, nil
}

func (m *Module) DeactivateItem(tenantId, id string) error {
	item, err := m.GetItem(tenantId, id)
	if err != nil {
		return err
	}
	item.IsActive = false
	item.UpdatedAt = time.Now()
	if err := m.db.Update(&item, orm.Eq(CatalogItem_.Id, item.Id)); err != nil {
		return err
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicItemDeactivated, Payload: &item})
	}
	return nil
}

func (m *Module) DeleteItem(tenantId, id string) error {
	item, err := m.GetItem(tenantId, id)
	if err != nil {
		return err
	}
	if err := m.db.Delete(&item, orm.Eq(CatalogItem_.Id, item.Id)); err != nil {
		return err
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicItemDeleted, Payload: &item})
	}
	return nil
}

func (m *Module) ServiceExists(tenantId, serviceId string) (bool, error) {
	item, err := m.GetItem(tenantId, serviceId)
	if err != nil {
		if err == ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return item.Type == "S" && item.IsActive, nil
}

func (m *Module) ListAgreements(tenantId, catalogItemId string) ([]Agreement, error) {
	var a Agreement
	qb := m.db.Query(&a).Where(Agreement_.TenantId).Eq(tenantId)
	if catalogItemId != "" {
		qb = qb.Where(Agreement_.CatalogItemId).Eq(catalogItemId)
	}
	results, err := ReadAllAgreement(qb)
	if err != nil {
		return nil, err
	}
	items := make([]Agreement, len(results))
	for i, r := range results {
		items[i] = *r
	}
	return items, nil
}

func (m *Module) GetAgreement(tenantId, id string) (Agreement, error) {
	var a Agreement
	qb := m.db.Query(&a).Where(Agreement_.Id).Eq(id).Where(Agreement_.TenantId).Eq(tenantId)
	_, err := ReadOneAgreement(qb, &a)
	if err != nil {
		return Agreement{}, err
	}
	return a, nil
}

func (m *Module) UpsertAgreement(a Agreement) (Agreement, error) {
	a.UpdatedAt = time.Now()
	if a.Id == "" {
		a.Id = m.ids.NewID()
		if err := m.db.Create(&a); err != nil {
			return Agreement{}, err
		}
		if m.pub != nil {
			m.pub.Publish(events.Event{Topic: TopicAgreementCreated, Payload: &a})
		}
		return a, nil
	}
	if err := m.db.Update(&a, orm.Eq(Agreement_.Id, a.Id)); err != nil {
		return Agreement{}, err
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicAgreementUpdated, Payload: &a})
	}
	return a, nil
}

func (m *Module) DeleteAgreement(tenantId, id string) error {
	a, err := m.GetAgreement(tenantId, id)
	if err != nil {
		// fallback if not found or error, but let's try deletion anyway if the fetch failed?
		// No, let's fetch first as instructed so we have the agreement info to publish.
		// If fetch fails, we just try to delete a stub.
	}

	if err == nil {
		if err := m.db.Delete(&a, orm.Eq(Agreement_.Id, id)); err != nil {
			return err
		}
		if m.pub != nil {
			m.pub.Publish(events.Event{Topic: TopicAgreementDeleted, Payload: &a})
		}
		return nil
	}

	stub := Agreement{Id: id, TenantId: tenantId}
	if err := m.db.Delete(&stub, orm.Eq(Agreement_.Id, id)); err != nil {
		return err
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicAgreementDeleted, Payload: &stub})
	}
	return nil
}


func (m *Module) ModelName() string { return "item_catalog" }

func (m *Module) MountOps(reg router.OpRegistry) {
	reg.Op(OpListItems, m.opListItems).Requires("catalog_item", model.Read).Accepts(&ListItemsArgs{})
	reg.Op(OpGetItem, m.opGetItem).Requires("catalog_item", model.Read).Accepts(&GetItemArgs{})
	reg.Op(OpFindItemBySKU, m.opFindItemBySKU).Requires("catalog_item", model.Read).Accepts(&FindBySKUArgs{})
	reg.Op(OpCreateItem, m.opCreateItem).Requires("catalog_item", model.Create).Accepts(&CatalogItem{})
	reg.Op(OpUpdateItem, m.opUpdateItem).Requires("catalog_item", model.Update).Accepts(&CatalogItem{})
	reg.Op(OpUpsertItem, m.opUpsertItem).Requires("catalog_item", model.Create).Accepts(&CatalogItem{})
	reg.Op(OpDeactivateItem, m.opDeactivateItem).Requires("catalog_item", model.Update).Accepts(&DeactivateItemArgs{})
	reg.Op(OpDeleteItem, m.opDeleteItem).Requires("catalog_item", model.Delete).Accepts(&DeleteItemArgs{})
	reg.Op(OpListAgreements, m.opListAgreements).Requires("catalog_agreement", model.Read).Accepts(&ListAgreementsArgs{})
	reg.Op(OpUpsertAgreement, m.opUpsertAgreement).Requires("catalog_agreement", model.Create).Accepts(&Agreement{})
	reg.Op(OpDeleteAgreement, m.opDeleteAgreement).Requires("catalog_agreement", model.Delete).Accepts(&DeleteAgreementArgs{})
}

var _ router.OpModule = (*Module)(nil)

func (m *Module) opListItems(ctx router.Context) {
	var args ListItemsArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	filter := ItemFilter{Type: args.Type, ActiveOnly: args.ActiveOnly, Limit: args.Limit, Offset: args.Offset}
	items, err := m.ListItems(args.TenantId, filter)
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	list := make(CatalogItemList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	if err := ctx.Encode(&list); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opGetItem(ctx router.Context) {
	var args GetItemArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	item, err := m.GetItem(args.TenantId, args.Id)
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	if err := ctx.Encode(&item); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opFindItemBySKU(ctx router.Context) {
	var args FindBySKUArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	item, err := m.FindBySKU(args.TenantId, args.Sku)
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	if err := ctx.Encode(&item); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opCreateItem(ctx router.Context) {
	var item CatalogItem
	if err := ctx.Decode(&item); err != nil {
		ctx.WriteStatus(400)
		return
	}
	created, err := m.CreateItem(item)
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	if err := ctx.Encode(&created); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opUpdateItem(ctx router.Context) {
	var item CatalogItem
	if err := ctx.Decode(&item); err != nil {
		ctx.WriteStatus(400)
		return
	}
	updated, err := m.UpdateItem(item)
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	if err := ctx.Encode(&updated); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opUpsertItem(ctx router.Context) {
	var item CatalogItem
	if err := ctx.Decode(&item); err != nil {
		ctx.WriteStatus(400)
		return
	}
	var out CatalogItem
	var err error
	if item.Id == "" {
		out, err = m.CreateItem(item)
	} else {
		out, err = m.UpdateItem(item)
	}
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	if err := ctx.Encode(&out); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opDeactivateItem(ctx router.Context) {
	var args DeactivateItemArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := m.DeactivateItem(args.TenantId, args.Id); err != nil {
		ctx.WriteStatus(500)
		return
	}
	ctx.WriteStatus(200)
}

func (m *Module) opDeleteItem(ctx router.Context) {
	var args DeleteItemArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := m.DeleteItem(args.TenantId, args.Id); err != nil {
		ctx.WriteStatus(500)
		return
	}
	ctx.WriteStatus(200)
}

func (m *Module) opListAgreements(ctx router.Context) {
	var args ListAgreementsArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	items, err := m.ListAgreements(args.TenantId, args.CatalogItemId)
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	list := make(AgreementList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	if err := ctx.Encode(&list); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opUpsertAgreement(ctx router.Context) {
	var a Agreement
	if err := ctx.Decode(&a); err != nil {
		ctx.WriteStatus(400)
		return
	}
	out, err := m.UpsertAgreement(a)
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	if err := ctx.Encode(&out); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opDeleteAgreement(ctx router.Context) {
	var args DeleteAgreementArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := m.DeleteAgreement(args.TenantId, args.Id); err != nil {
		ctx.WriteStatus(500)
		return
	}
	ctx.WriteStatus(200)
}
