# Item Catalog Architecture

## Domain Scope
The `item-catalog` module manages a unified catalog of services and products for the Velty ecosystem. It serves as the source of truth for other modules, such as `appointment-booking`.

## Entities
- **CatalogItem**: Represents a service (`S`) or product (`P`). Contains details like SKU, name, description, price, and currency.

## Patterns
- **Hexagonal Architecture**: Uses ports (interfaces) for external concerns:
    - `orm.DB` for storage.
    - `UIAdapter` for UI rendering.
    - `EventPublisher` for event-driven updates.
- **Multi-tenancy**: Every item is associated with a `tenant_id`. SKU uniqueness is enforced per tenant.
- **MCP Integration**: Provides a set of tools for AI agents to interact with the catalog.

## MCP Tools
| Tool Name | Action | Resource | Description |
|-----------|--------|----------|-------------|
| `list_catalog_items` | `r` | `catalog_item` | List items for a tenant |
| `get_catalog_item` | `r` | `catalog_item` | Get item by ID |
| `find_item_by_sku` | `r` | `catalog_item` | Find item by SKU |
| `create_catalog_item` | `c` | `catalog_item` | Create new item |
| `update_catalog_item` | `u` | `catalog_item` | Update existing item |
| `deactivate_catalog_item` | `u` | `catalog_item` | Soft-delete item |
| `delete_catalog_item` | `d` | `catalog_item` | Hard-delete item |

## Composition Root Example
```go
cat := itemcatalog.New(db, itemcatalog.Deps{
    UI: uiAdapter,
    Publisher: eventPublisher,
})
// Catalog implements CatalogReader interface for other modules
appt := appointmentbooking.New(db, appointmentbooking.Deps{Catalog: cat, ...})
```
