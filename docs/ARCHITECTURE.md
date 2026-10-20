# Item Catalog Architecture

## Domain Scope
The `service-catalog` module manages a unified catalog of services and products for the Velty ecosystem. It serves as the source of truth for other modules, such as `appointment-booking`.

## Entities
- **Specialty**: Taxonomy entity for the catalog, representing clinical/service specialties with a SKU prefix and public site slug.
- **Service**: Represents a service (`S`, represented by `ItemTypeService`) or product (`P`, represented by `ItemTypeProduct`). References a `Specialty` via `specialty_id`. Contains details like SKU, name, description, price, and currency.
  - *Design Note*: What other modules (like `appointment-booking`) actually need from this entity is the *capability* of being "bookable", not the literal service label itself. If a future item classification is also bookable (e.g. a session package), comparing raw type strings across modules breaks silently. We keep these semantics fully concentrated in `ServiceExists` so extending bookable types is a single-line change in one place.
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
    - `view.Presenter` (`NewView(caller router.Caller) view.Presenter`, `NewSpecialtyView(caller router.Caller) view.Presenter`) for UI, built with only
      `view`+`model`+`router` — the app chooses the renderer. The specialty is a form field loaded with `SetOptions`; no picker beside the title.
    - Tests run against `storage/mem`, never `tinywasm/sqlite`.
- **Multi-tenancy**: Every specialty, item, and agreement is associated with a `tenant_id`. SKU, prefix, and slug uniqueness are enforced per tenant.
- **Typed events**: every published event carries a `model.Encodable` payload (`&Specialty`/`&Service`/`&Agreement`),
  never a bare `map`.

## Ops (via `MountOps`)
| Op | Action | Resource | Description |
|-----------|--------|----------|-------------|
| `list_specialties` | `r` | `specialty` | List specialties for a tenant |
| `get_specialty` | `r` | `specialty` | Get specialty by ID |
| `upsert_specialty` | `c`\|`u` | `specialty` | Create or update a specialty |
| `delete_specialty` | `d` | `specialty` | Delete a specialty |
| `list_services` | `r` | `service` | List items for a tenant |
| `get_service` | `r` | `service` | Get item by ID |
| `find_item_by_sku` | `r` | `service` | Find item by SKU |
| `create_service` | `c` | `service` | Create new item |
| `update_service` | `u` | `service` | Update existing item |
| `upsert_service` | `c`\|`u` | `service` | Create or update item |
| `deactivate_service` | `u` | `service` | Soft-delete item |
| `delete_service` | `d` | `service` | Hard-delete item |
| `list_agreements` | `r` | `service_agreement` | List agreements of a catalog item |
| `upsert_agreement` | `c`\|`u` | `service_agreement` | Create or update an agreement |
| `delete_agreement` | `d` | `service_agreement` | Delete an agreement |

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

### Database Note
There is no production data. Development databases are recreated: `Migrate` creates `specialty`, `service` and `service_agreement` (in that FK order). It never drops the old `catalog_item` / `catalog_agreement` tables.
