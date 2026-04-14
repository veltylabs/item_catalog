# item-catalog
<img src="docs/img/badges.svg">

Item catalog MCP module for the Velty ecosystem. Manages a unified catalog of services and products.

## Quick Start

```go
db, _ := sqlite.Open("catalog.db")
module, _ := itemcatalog.New(db, itemcatalog.Deps{})

// Use the module
item, _ := module.CreateItem(itemcatalog.CatalogItem{
    TenantID: "my-tenant",
    SKU:      "SKU001",
    Name:     "Initial Product",
    Type:     "P",
    Price:    100,
    Currency: "USD",
    IsActive: true,
})
```

## MCP Tools
| Tool Name | Description |
|-----------|-------------|
| `list_catalog_items` | List items for a tenant |
| `get_catalog_item` | Get item by ID |
| `find_item_by_sku` | Find item by SKU |
| `create_catalog_item` | Create new item |
| `update_catalog_item` | Update existing item |
| `deactivate_catalog_item` | Soft-delete item |
| `delete_catalog_item` | Hard-delete item |

## Key Files
| File | Purpose |
|------|---------|
| `model.go` | Data structures and interfaces |
| `mcp.go` | Core logic and MCP handlers |
| `model_orm.go` | Generated ORM helpers |
| `tests/` | Functional and unit tests |
