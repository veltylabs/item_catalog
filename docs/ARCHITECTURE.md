# Service Catalog Architecture

## Domain Scope
The `service-catalog` module manages the catalog of services and their specialties for the Velty ecosystem (products and supplies live in `pharmacy_manager` and `supply_manager`). It serves as the source of truth for other modules, such as `appointment-booking`.

## Entities
- **Specialty**: taxonomía con un código de referencia de dos letras (prefijo de los SKU).
- **Service**: A service the business provides. References a `Specialty` via `specialty_id`. Contains SKU, name, description, price, currency and an optional `tax_treatment` (`TAXED` or `EXEMPT`; the rate is not stored here).
  - *Design Note*: other modules (like `appointment-booking`) need the *capability* of being bookable, kept in one place: `ServiceExists`.
- **Agreement**: Represents a billing/insurer agreement (convenio) associated with a service. A service can have multiple agreements, each specifying an insurer (e.g., FONASA, Isapre X), code, price, and active status.

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
- **Multi-tenancy**: Every specialty, service, and agreement is associated with a `tenant_id`. SKU and prefix uniqueness are enforced per tenant.
- **Accepted Currencies**: `Deps.Currencies` sets the currencies the module accepts (ISO 4217). If empty, it defaults to CLP.
- **Typed events**: every published event carries a `model.Encodable` payload (`&Specialty`/`&Service`/`&Agreement`),
  never a bare `map`.

## Ops (via `MountOps`)
| Op | Action | Resource | Description |
|-----------|--------|----------|-------------|
| `list_specialties` | `r` | `specialty` | List specialties for a tenant |
| `get_specialty` | `r` | `specialty` | Get specialty by ID |
| `upsert_specialty` | `c`\|`u` | `specialty` | Create or update a specialty |
| `delete_specialty` | `d` | `specialty` | Delete a specialty |
| `list_services` | `r` | `service` | List services for a tenant |
| `get_service` | `r` | `service` | Get service by ID |
| `find_service_by_sku` | `r` | `service` | Find service by SKU |
| `create_service` | `c` | `service` | Create new service |
| `update_service` | `u` | `service` | Update existing service |
| `upsert_service` | `c`\|`u` | `service` | Create or update service |
| `deactivate_service` | `u` | `service` | Soft-delete service |
| `delete_service` | `d` | `service` | Hard-delete service |
| `list_agreements` | `r` | `service_agreement` | List agreements of a service |
| `upsert_agreement` | `c`\|`u` | `service_agreement` | Create or update an agreement |
| `delete_agreement` | `d` | `service_agreement` | Delete an agreement |

## Composition Root Example
```go
cat, _ := servicecatalog.New(db, servicecatalog.Deps{
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

## Supply recipe

A service has a **recipe**: the supplies one performance uses (`service_supply`: service, `item_id` of `supply_manager` as a
soft reference, quantity > 0, each supply once per service). `ServiceRecipe(tenantId, serviceId) (itemIds []string,
quantities []float64, err error)` returns it with plain types, so `clinical_encounter` declares it in a narrow interface of
its own and the application passes this module. This module never imports `supply_manager`; the supply picker of the
service screen comes with the screens that follow the module-guidance rules.
