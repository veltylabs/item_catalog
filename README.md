# service-catalog
<img src="docs/img/badges.svg">

Catalog of the **services** a business provides and the **specialties** that classify them. Designed for the Velty
ecosystem. Products and supplies are other modules (`pharmacy_manager`, `supply_manager`).

## Quick Start

```go
import (
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
	servicecatalog "github.com/veltylabs/service_catalog"
)

// In a real application, inject the database connection chosen by the composer app
db := orm.New(mem.New())
idGenerator := &MyIDGenerator{} // implements model.IDGenerator

module, _ := servicecatalog.New(db, servicecatalog.Deps{
	IDs: idGenerator,
})

// Create specialty
specialty, _ := module.UpsertSpecialty(servicecatalog.Specialty{
	TenantId: "my-tenant",
	Prefix:   "md",
	Slug:     "medicina-general",
	Name:     "Medicina General",
})

// Create a service
service, _ := module.CreateService(servicecatalog.Service{
	TenantId:    "my-tenant",
	SpecialtyId: specialty.Id,
	Sku:         "md0001",
	Name:        "General consultation",
	Price:       100,
	Currency:    "USD",
	IsActive:    true,
})
```

## Ops
| Op Name | Description |
|-----------|-------------|
| `list_specialties` | List specialties for a tenant |
| `get_specialty` | Get specialty by ID |
| `upsert_specialty` | Create or update specialty |
| `delete_specialty` | Delete specialty |
| `list_services` | List services for a tenant |
| `get_service` | Get service by ID |
| `find_service_by_sku` | Find service by SKU |
| `create_service` | Create new service |
| `update_service` | Update existing service |
| `upsert_service` | Create or update service |
| `deactivate_service` | Soft-delete service |
| `delete_service` | Hard-delete service |
| `list_agreements` | List billing agreements |
| `upsert_agreement` | Create or update agreement |
| `delete_agreement` | Delete agreement |

## Readers (Cross-Module)
- `ServiceExists(tenantId, serviceId string) (bool, error)`: Checks if a service exists and is active.
- `ServiceCard(tenantId, serviceId string) (name string, price float64, currency string, active bool, err error)`: Gets service details for billing/booking.

## View and demo

This module carries its own UI component (`ui/`), seed data (`seed/`), and runnable in-browser demo (`web/`).

- **`ui/`**: Exports `ID` (`catalog_item`), `Label` (`Catálogo`), and `Browser(caller, ids, tenantID)`. The specialty is a form field loaded with `SetOptions`; no picker beside the title.
- **`seed/`**: Exports `seed.Load(module, tenantID)`, populating 3 canonical specialties and 1 service in each.
- **`web/`**: Runnable WASM demo. Run `webtyp` at the repository root to open the demo — in-browser, in-memory, no login.

## Key Files
| File | Purpose |
|------|---------|
| `model.go` | Data structures and schemas |
| `mcp.go` | Core logic and transport operations |
| `migration.go` | SKU prefix to specialty migration |
| `model_orm.go` | Generated ORM helpers |
| `ui/` | Module view presenter and components |
| `seed/` | Demo seed data loader |
| `web/` | WASM in-browser demo entrypoint |
| `tests/` | Functional and unit tests |
