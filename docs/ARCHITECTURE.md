# Item Catalog Architecture

## Domain Scope
The `item-catalog` module manages a unified catalog of services and products for the Velty ecosystem. It serves as the source of truth for other modules, such as `appointment-booking`.

## Entities
- **CatalogItem**: Represents a service (`S`) or product (`P`). Contains details like SKU, name, description, price, and currency.
- **Agreement**: Represents a billing/insurer agreement (convenio) associated with a catalog item. A catalog item can have multiple agreements, each specifying an insurer (e.g., FONASA, Isapre X), code, price, and active status.

## Patterns
- **Reusable-module harness**: the module is coupled only to published contracts, not to concrete
  infrastructure — see `AGENTS.md` (this repo's root) for the full whitelist/blacklist this module
  must hold to:
    - `orm.DB` for storage (backend-agnostic — wraps whatever `storage.Conn` the app injects).
    - `ddl.CreateTable`/`ddl.Sync` (over `db.RawConn()`) for the module's own schema migration in
      `New()` — replaces the removed `orm.DB.CreateTable`.
    - `router.OpModule` (`ModelName()` + `MountOps(reg router.OpRegistry)`) for transport — the module
      never sees `router.Router`/`router.APIModule`, and never imports `tinywasm/mcp`.
    - `model.IDGenerator` for identity (`Deps.IDs`, required — the module never builds its own).
    - `events.Publisher` for event-driven updates (`Deps.Publisher`, optional — `nil` disables
      publishing silently).
    - `view.Presenter` (`NewView(caller router.Caller) view.Presenter`) for UI, built with only
      `view`+`model`+`router` — the app chooses the renderer.
    - Tests run against `storage/mem`, never `tinywasm/sqlite` — see the open item in `docs/PLAN.md`.
- **Multi-tenancy**: Every item and agreement is associated with a `tenant_id`. SKU uniqueness is enforced per tenant.
- **Typed events**: every published event carries a `model.Encodable` payload (`&CatalogItem`/`&Agreement`),
  never a bare `map`.

## Ops (via `MountOps`)
| Op | Action | Resource | Description |
|-----------|--------|----------|-------------|
| `list_catalog_items` | `r` | `catalog_item` | List items for a tenant |
| `get_catalog_item` | `r` | `catalog_item` | Get item by ID |
| `find_item_by_sku` | `r` | `catalog_item` | Find item by SKU |
| `create_catalog_item` | `c` | `catalog_item` | Create new item |
| `update_catalog_item` | `u` | `catalog_item` | Update existing item |
| `upsert_catalog_item` | `c` | `catalog_item` | Create or update item |
| `deactivate_catalog_item` | `u` | `catalog_item` | Soft-delete item |
| `delete_catalog_item` | `d` | `catalog_item` | Hard-delete item |
| `list_agreements` | `r` | `catalog_agreement` | List agreements of a catalog item |
| `upsert_agreement` | `c` | `catalog_agreement` | Create or update an agreement |
| `delete_agreement` | `d` | `catalog_agreement` | Delete an agreement |

## Composition Root Example
```go
cat, _ := itemcatalog.New(db, itemcatalog.Deps{
    IDs:       idGenerator,   // model.IDGenerator
    Publisher: eventPublisher, // events.Publisher, nil disables publishing
})
cat.MountOps(opRegistry) // router.OpRegistry
view := cat.NewView(caller) // router.Caller -> view.Presenter
// Catalog implements CatalogReader interface for other modules
appt := appointmentbooking.New(db, appointmentbooking.Deps{Catalog: cat, ...})
```
