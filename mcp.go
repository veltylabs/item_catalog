//go:build !wasm

package itemcatalog

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/tinywasm/context"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/orm"
	"github.com/tinywasm/unixid"
)

var ErrNotFound = errors.New("item not found")
var ErrAlreadyExists = errors.New("item already exists")

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
	u, err := unixid.NewUnixID()
	if err != nil {
		return nil, err
	}
	return &Module{db: db, uid: u, ui: deps.UI, pub: deps.Publisher}, nil
}

// Service methods

func (m *Module) GetItem(tenantID, id string) (CatalogItem, error) {
	var item CatalogItem
	qb := m.db.Query(&item).Where(CatalogItem_.ID).Eq(id).Where(CatalogItem_.TenantID).Eq(tenantID)
	_, err := ReadOneCatalogItem(qb, &item)
	if err != nil {
		if errors.Is(err, orm.ErrNotFound) {
			return CatalogItem{}, ErrNotFound
		}
		return CatalogItem{}, err
	}
	return item, nil
}

func (m *Module) FindBySKU(tenantID, sku string) (CatalogItem, error) {
	var item CatalogItem
	qb := m.db.Query(&item).Where(CatalogItem_.SKU).Eq(sku).Where(CatalogItem_.TenantID).Eq(tenantID)
	_, err := ReadOneCatalogItem(qb, &item)
	if err != nil {
		if errors.Is(err, orm.ErrNotFound) {
			return CatalogItem{}, ErrNotFound
		}
		return CatalogItem{}, err
	}
	return item, nil
}

func (m *Module) ListItems(tenantID string, filter ItemFilter) ([]CatalogItem, error) {
	var item CatalogItem
	qb := m.db.Query(&item).Where(CatalogItem_.TenantID).Eq(tenantID)
	if filter.Type != "" {
		qb = qb.Where(CatalogItem_.Type).Eq(filter.Type)
	}
	if filter.ActiveOnly {
		qb = qb.Where(CatalogItem_.IsActive).Eq(true)
	}
	if filter.Limit > 0 {
		qb = qb.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		qb = qb.Offset(filter.Offset)
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
	existing, err := m.FindBySKU(item.TenantID, item.SKU)
	if err == nil && existing.ID != "" {
		return CatalogItem{}, ErrAlreadyExists
	}

	item.ID = m.uid.GetNewID()
	item.UpdatedAt = time.Now().Unix()
	if err := m.db.Create(&item); err != nil {
		return CatalogItem{}, err
	}
	m.publish("catalog.item.created", item)
	return item, nil
}

func (m *Module) UpdateItem(item CatalogItem) (CatalogItem, error) {
	// Verify item exists and belongs to tenant
	_, err := m.GetItem(item.TenantID, item.ID)
	if err != nil {
		return CatalogItem{}, err
	}

	item.UpdatedAt = time.Now().Unix()
	if err := m.db.Update(&item, orm.Eq(CatalogItem_.ID, item.ID)); err != nil {
		return CatalogItem{}, err
	}
	m.publish("catalog.item.updated", item)
	return item, nil
}

func (m *Module) DeactivateItem(tenantID, id string) error {
	item, err := m.GetItem(tenantID, id)
	if err != nil {
		return err
	}
	item.IsActive = false
	item.UpdatedAt = time.Now().Unix()
	if err := m.db.Update(&item, orm.Eq(CatalogItem_.ID, item.ID)); err != nil {
		return err
	}
	m.publish("catalog.item.deactivated", map[string]string{"tenant_id": tenantID, "id": id})
	return nil
}

func (m *Module) DeleteItem(tenantID, id string) error {
	item, err := m.GetItem(tenantID, id)
	if err != nil {
		return err
	}
	if err := m.db.Delete(&item, orm.Eq(CatalogItem_.ID, item.ID)); err != nil {
		return err
	}
	m.publish("catalog.item.deleted", map[string]string{"tenant_id": tenantID, "id": id})
	return nil
}

func (m *Module) ServiceExists(tenantID, serviceID string) (bool, error) {
	item, err := m.GetItem(tenantID, serviceID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return item.Type == "S" && item.IsActive, nil
}

func (m *Module) publish(event string, payload any) {
	if m.pub != nil {
		_ = m.pub.Publish(event, payload) // fire-and-forget
	}
}

// UI methods

func (m *Module) RenderList(tenantID, filter string) string {
	if m.ui == nil {
		return ""
	}
	items, _ := m.ListItems(tenantID, ItemFilter{Type: filter, ActiveOnly: true})
	return m.ui.RenderItemList(items, filter)
}

func (m *Module) RenderForm(tenantID, id string) string {
	if m.ui == nil {
		return ""
	}
	if id == "" {
		return m.ui.RenderItemForm(nil)
	}
	item, _ := m.GetItem(tenantID, id)
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
			Name:        "list_catalog_items",
			Description: "List catalog items for a tenant",
			Resource:    "catalog_item",
			Action:      'r',
			Execute:     m.mcpListItems,
		},
		{
			Name:        "get_catalog_item",
			Description: "Get a catalog item by ID",
			Resource:    "catalog_item",
			Action:      'r',
			Execute:     m.mcpGetItem,
		},
		{
			Name:        "find_item_by_sku",
			Description: "Find a catalog item by SKU",
			Resource:    "catalog_item",
			Action:      'r',
			Execute:     m.mcpFindBySKU,
		},
		{
			Name:        "create_catalog_item",
			Description: "Create a new catalog item",
			Resource:    "catalog_item",
			Action:      'c',
			Execute:     m.mcpCreateItem,
		},
		{
			Name:        "update_catalog_item",
			Description: "Update an existing catalog item",
			Resource:    "catalog_item",
			Action:      'u',
			Execute:     m.mcpUpdateItem,
		},
		{
			Name:        "deactivate_catalog_item",
			Description: "Deactivate (soft-delete) a catalog item",
			Resource:    "catalog_item",
			Action:      'u',
			Execute:     m.mcpDeactivateItem,
		},
		{
			Name:        "delete_catalog_item",
			Description: "Hard delete a catalog item",
			Resource:    "catalog_item",
			Action:      'd',
			Execute:     m.mcpDeleteItem,
		},
	}
}

func (m *Module) mcpListItems(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args struct {
		TenantID string `json:"tenant_id"`
		ItemFilter
	}
	if err := json.Unmarshal([]byte(req.Params.Arguments), &args); err != nil {
		return nil, err
	}
	items, err := m.ListItems(args.TenantID, args.ItemFilter)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	res, _ := json.Marshal(items)
	return mcp.Text(string(res)), nil
}

func (m *Module) mcpGetItem(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args struct {
		TenantID string `json:"tenant_id"`
		ID       string `json:"id"`
	}
	if err := json.Unmarshal([]byte(req.Params.Arguments), &args); err != nil {
		return nil, err
	}
	item, err := m.GetItem(args.TenantID, args.ID)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	res, _ := json.Marshal(item)
	return mcp.Text(string(res)), nil
}

func (m *Module) mcpFindBySKU(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args struct {
		TenantID string `json:"tenant_id"`
		SKU      string `json:"sku"`
	}
	if err := json.Unmarshal([]byte(req.Params.Arguments), &args); err != nil {
		return nil, err
	}
	item, err := m.FindBySKU(args.TenantID, args.SKU)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	res, _ := json.Marshal(item)
	return mcp.Text(string(res)), nil
}

func (m *Module) mcpCreateItem(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var item CatalogItem
	if err := json.Unmarshal([]byte(req.Params.Arguments), &item); err != nil {
		return nil, err
	}
	created, err := m.CreateItem(item)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	res, _ := json.Marshal(created)
	return mcp.Text(string(res)), nil
}

func (m *Module) mcpUpdateItem(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var item CatalogItem
	if err := json.Unmarshal([]byte(req.Params.Arguments), &item); err != nil {
		return nil, err
	}
	updated, err := m.UpdateItem(item)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	res, _ := json.Marshal(updated)
	return mcp.Text(string(res)), nil
}

func (m *Module) mcpDeactivateItem(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args struct {
		TenantID string `json:"tenant_id"`
		ID       string `json:"id"`
	}
	if err := json.Unmarshal([]byte(req.Params.Arguments), &args); err != nil {
		return nil, err
	}
	if err := m.DeactivateItem(args.TenantID, args.ID); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text("item deactivated"), nil
}

func (m *Module) mcpDeleteItem(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args struct {
		TenantID string `json:"tenant_id"`
		ID       string `json:"id"`
	}
	if err := json.Unmarshal([]byte(req.Params.Arguments), &args); err != nil {
		return nil, err
	}
	if err := m.DeleteItem(args.TenantID, args.ID); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text("item deleted"), nil
}
