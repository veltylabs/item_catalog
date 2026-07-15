//go:build !wasm

package itemcatalog

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/model"
	"github.com/tinywasm/orm"
	"github.com/tinywasm/time"
	"github.com/tinywasm/unixid"
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
)

type Deps struct {
	UI        UIAdapter      // optional — nil disables UI methods
	Publisher EventPublisher // optional — nil disables events
}

type Module struct {
	db  *orm.DB
	uid *unixid.UnixID
	ui  UIAdapter
	pub EventPublisher
}

func New(db *orm.DB, deps Deps) (*Module, error) {
	if err := db.CreateTable(&CatalogItem{}); err != nil {
		return nil, err
	}
	if err := db.CreateTable(&Agreement{}); err != nil {
		return nil, err
	}
	u, err := unixid.NewUnixID()
	if err != nil {
		return nil, err
	}
	return &Module{db: db, uid: u, ui: deps.UI, pub: deps.Publisher}, nil
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

	item.Id = m.uid.GetNewID()
	item.UpdatedAt = time.Now()
	if err := m.db.Create(&item); err != nil {
		return CatalogItem{}, err
	}
	m.publish("catalog.item.created", item)
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
	m.publish("catalog.item.updated", item)
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
	m.publish("catalog.item.deactivated", map[string]string{"tenant_id": tenantId, "id": id})
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
	m.publish("catalog.item.deleted", map[string]string{"tenant_id": tenantId, "id": id})
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

func (m *Module) UpsertAgreement(a Agreement) (Agreement, error) {
	a.UpdatedAt = time.Now()
	if a.Id == "" {
		a.Id = m.uid.GetNewID()
		if err := m.db.Create(&a); err != nil {
			return Agreement{}, err
		}
		m.publish("catalog.agreement.created", a)
		return a, nil
	}
	if err := m.db.Update(&a, orm.Eq(Agreement_.Id, a.Id)); err != nil {
		return Agreement{}, err
	}
	m.publish("catalog.agreement.updated", a)
	return a, nil
}

func (m *Module) DeleteAgreement(tenantId, id string) error {
	a := Agreement{Id: id, TenantId: tenantId}
	if err := m.db.Delete(&a, orm.Eq(Agreement_.Id, id)); err != nil {
		return err
	}
	m.publish("catalog.agreement.deleted", map[string]string{"tenant_id": tenantId, "id": id})
	return nil
}

func (m *Module) publish(event string, payload any) {
	if m.pub != nil {
		_ = m.pub.Publish(event, payload) // fire-and-forget
	}
}

// UI methods

func (m *Module) RenderList(tenantId, filter string) string {
	if m.ui == nil {
		return ""
	}
	items, _ := m.ListItems(tenantId, ItemFilter{Type: filter, ActiveOnly: true})
	return m.ui.RenderItemList(items, filter)
}

func (m *Module) RenderForm(tenantId, id string) string {
	if m.ui == nil {
		return ""
	}
	if id == "" {
		return m.ui.RenderItemForm(nil)
	}
	item, _ := m.GetItem(tenantId, id)
	return m.ui.RenderItemForm(&item)
}

func (m *Module) RenderFilter(current string) string {
	if m.ui == nil {
		return ""
	}
	return m.ui.RenderFilterSelector(current)
}

// MCP ToolProvider

func (m *Module) Tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        OpListItems,
			Description: "List catalog items for a tenant",
			Args:        &ListItemsArgs{},
			Resource:    "catalog_item",
			Action:      model.Read,
			Execute:     m.mcpListItems,
		},
		{
			Name:        OpGetItem,
			Description: "Get a catalog item by ID",
			Args:        &GetItemArgs{},
			Resource:    "catalog_item",
			Action:      model.Read,
			Execute:     m.mcpGetItem,
		},
		{
			Name:        OpFindItemBySKU,
			Description: "Find a catalog item by SKU",
			Args:        &FindBySKUArgs{},
			Resource:    "catalog_item",
			Action:      model.Read,
			Execute:     m.mcpFindBySKU,
		},
		{
			Name:        OpCreateItem,
			Description: "Create a new catalog item",
			Args:        &CatalogItem{},
			Resource:    "catalog_item",
			Action:      model.Create,
			Execute:     m.mcpCreateItem,
		},
		{
			Name:        OpUpdateItem,
			Description: "Update an existing catalog item",
			Args:        &CatalogItem{},
			Resource:    "catalog_item",
			Action:      model.Update,
			Execute:     m.mcpUpdateItem,
		},
		{
			Name:        OpUpsertItem,
			Description: "Create or update a catalog item (create if id is empty)",
			Args:        &CatalogItem{},
			Resource:    "catalog_item",
			Action:      model.Create,
			Execute:     m.mcpUpsertItem,
		},
		{
			Name:        OpDeactivateItem,
			Description: "Deactivate (soft-delete) a catalog item",
			Args:        &DeactivateItemArgs{},
			Resource:    "catalog_item",
			Action:      model.Update,
			Execute:     m.mcpDeactivateItem,
		},
		{
			Name:        OpDeleteItem,
			Description: "Hard delete a catalog item",
			Args:        &DeleteItemArgs{},
			Resource:    "catalog_item",
			Action:      model.Delete,
			Execute:     m.mcpDeleteItem,
		},
		{
			Name:        OpListAgreements,
			Description: "List agreements (convenios) of a catalog item",
			Args:        &ListAgreementsArgs{},
			Resource:    "catalog_agreement",
			Action:      model.Read,
			Execute:     m.mcpListAgreements,
		},
		{
			Name:        OpUpsertAgreement,
			Description: "Create or update an agreement (create if id is empty)",
			Args:        &Agreement{},
			Resource:    "catalog_agreement",
			Action:      model.Create,
			Execute:     m.mcpUpsertAgreement,
		},
		{
			Name:        OpDeleteAgreement,
			Description: "Delete an agreement",
			Args:        &DeleteAgreementArgs{},
			Resource:    "catalog_agreement",
			Action:      model.Delete,
			Execute:     m.mcpDeleteAgreement,
		},
	}
}

func (m *Module) mcpListItems(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args ListItemsArgs
	if err := json.Decode(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	filter := ItemFilter{Type: args.Type, ActiveOnly: args.ActiveOnly, Limit: args.Limit, Offset: args.Offset}
	items, err := m.ListItems(args.TenantId, filter)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	// Convert []CatalogItem to CatalogItemList for JSON encoding
	itemList := make(CatalogItemList, len(items))
	for i := range items {
		itemList[i] = &items[i]
	}
	var res string
	if err := json.Encode(&itemList, &res); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text(res), nil
}

func (m *Module) mcpGetItem(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args GetItemArgs
	if err := json.Decode(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	item, err := m.GetItem(args.TenantId, args.Id)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	var res string
	if err := json.Encode(&item, &res); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text(res), nil
}

func (m *Module) mcpFindBySKU(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args FindBySKUArgs
	if err := json.Decode(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	item, err := m.FindBySKU(args.TenantId, args.Sku)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	var res string
	if err := json.Encode(&item, &res); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text(res), nil
}

func (m *Module) mcpCreateItem(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var item CatalogItem
	if err := json.Decode(req.Params.Arguments, &item); err != nil {
		return nil, err
	}
	created, err := m.CreateItem(item)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	var res string
	if err := json.Encode(&created, &res); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text(res), nil
}

func (m *Module) mcpUpdateItem(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var item CatalogItem
	if err := json.Decode(req.Params.Arguments, &item); err != nil {
		return nil, err
	}
	updated, err := m.UpdateItem(item)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	var res string
	if err := json.Encode(&updated, &res); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text(res), nil
}

func (m *Module) mcpUpsertItem(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var item CatalogItem
	if err := json.Decode(req.Params.Arguments, &item); err != nil {
		return nil, err
	}
	var out CatalogItem
	var err error
	if item.Id == "" {
		out, err = m.CreateItem(item)
	} else {
		out, err = m.UpdateItem(item)
	}
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	var res string
	if err := json.Encode(&out, &res); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text(res), nil
}

func (m *Module) mcpDeactivateItem(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args DeactivateItemArgs
	if err := json.Decode(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	if err := m.DeactivateItem(args.TenantId, args.Id); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text("item deactivated"), nil
}

func (m *Module) mcpDeleteItem(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args DeleteItemArgs
	if err := json.Decode(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	if err := m.DeleteItem(args.TenantId, args.Id); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text("item deleted"), nil
}

func (m *Module) mcpListAgreements(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args ListAgreementsArgs
	if err := json.Decode(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	items, err := m.ListAgreements(args.TenantId, args.CatalogItemId)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	list := make(AgreementList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	var res string
	if err := json.Encode(&list, &res); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text(res), nil
}

func (m *Module) mcpUpsertAgreement(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var a Agreement
	if err := json.Decode(req.Params.Arguments, &a); err != nil {
		return nil, err
	}
	out, err := m.UpsertAgreement(a)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	var res string
	if err := json.Encode(&out, &res); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text(res), nil
}

func (m *Module) mcpDeleteAgreement(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args DeleteAgreementArgs
	if err := json.Decode(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	if err := m.DeleteAgreement(args.TenantId, args.Id); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text("agreement deleted"), nil
}
