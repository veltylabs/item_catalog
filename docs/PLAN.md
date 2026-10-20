---
PLAN: "feat!: service_catalog v0.1.0 — item_catalog becomes the services-and-specialties catalog: no product type, tables service / service_agreement / specialty, optional tax treatment"
TAG: v0.1.0
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 2505385731027524456
PR: https://github.com/veltylabs/service_catalog/pull/12
REVIEWED_COMMIT: 45e2ee6cc49d0821572498054d6c65d91cca4468
---

> The GitHub repository is already named `veltylabs/service_catalog`; its `go.mod` still declares the old module path until Stage 1.

> Read AGENTS.md at the repo root before any change.

## Tools

```
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/ormc/cmd/ormc@latest
go install webtyp.com/lang/cmd/langc@latest
```

## Why

The module mixes in one table (`catalog_item`, a `type` of `S` or `P`) what the clinic **provides** (services) and what it
**holds** (products). Nothing in production uses a product (no row exists, and the only readers, `ServiceExists` and
`ServiceCard`, already filter `type == S`). Products and supplies are now two modules of their own (`pharmacy_manager`,
`supply_manager`). This module keeps **services and specialties** and takes the name of what it holds.

## Ecosystem search (done before writing this plan)

| Found | What it offers | Decision |
|---|---|---|
| `veltylabs/pharmacy_manager`, `veltylabs/supply_manager` | the products and supplies this module no longer holds | the reason for the split; **never imported** here |
| `webtyp/money` | `Money`, `Fact` | **not used in this version**: a service's list price stays `price` (float) + `currency` because `appointment_booking` snapshots it as a float today; moving prices to whole pesos is a later, separate wave |
| `appointment_booking`, `staff_manager`, `clinical_encounter`, `mjosefa-cms` | import `github.com/veltylabs/item_catalog` | consumers; each gets its own plan after this one is published (this plan does not touch them) |

## Design gate

1. **Prior art:** the module itself; FHIR separates the service definition from the product; Odoo/ERPNext keep one master
   with a flag (rejected for this clinic: master plan `CATALOG_SPLIT_MASTER_PLAN.md`, §4).
2. **Novice-name test:** `service_catalog.Service`, `Specialty`, `ServiceAgreement`. A reader guesses each.
3. **Complexity ledger:** concepts −1 (`type`), fields −1 on the form (the Type radio), ways to hold a product −1 (none here).
4. **Where it belongs:** unchanged; the module keeps its place in the DAG (a leaf).
5. **What it deletes:** `ItemTypeService`, `ItemTypeProduct`, `itemType()`, the `type` field, the `type` filter in every args
   model, the `type` radio of the form, the tests of the product type, and every name containing `item` or `catalog_item`
   (renamed below). The old table names `catalog_item` and `catalog_agreement` are no longer created.

## The contract

### Rename (module, package, tables, wire names)

| Before | After |
|---|---|
| module `github.com/veltylabs/item_catalog`, package `itemcatalog` | module `github.com/veltylabs/service_catalog`, package `servicecatalog` |
| table / resource `catalog_item` (`CatalogItemModel`, type `CatalogItem`) | table / resource `service` (`ServiceModel`, type `Service`) |
| table / resource `catalog_agreement` (`AgreementModel`, type `Agreement`, FK `catalog_item_id`) | table / resource `service_agreement` (`ServiceAgreementModel`, type `ServiceAgreement`, FK `service_id`) |
| table / resource `specialty` | unchanged |
| `ui.ID = "catalog_item"` | `ui.ID = "service_catalog"` (`ui.Label` stays `"Catálogo"`); the group's children stay `Servicios` and `Especialidades` |
| ops `list_catalog_items`, `get_catalog_item`, `find_item_by_sku`, `create_catalog_item`, `update_catalog_item`, `upsert_catalog_item`, `deactivate_catalog_item`, `delete_catalog_item` | `list_services`, `get_service`, `find_service_by_sku`, `create_service`, `update_service`, `upsert_service`, `deactivate_service`, `delete_service` (constants `OpListServices`, `OpGetService`, `OpFindServiceBySKU`, `OpCreateService`, `OpUpdateService`, `OpUpsertService`, `OpDeactivateService`, `OpDeleteService`) |
| ops `list_agreements`, `upsert_agreement`, `delete_agreement`; specialty ops | unchanged names |
| topics `catalog.item.*`, `catalog.agreement.*`, `catalog.specialty.*` | `catalog.service.created`, `.updated`, `.deactivated`, `.deleted` (`TopicService*`); `catalog.agreement.*` → `catalog.service_agreement.*`; specialty topics unchanged |
| args models `ListItemsArgs`, `GetItemArgs`, … `…Item…` | `ListServicesArgs`, `GetServiceArgs`, … `…Service…`; `ItemFilterModel` → `ServiceFilterModel` |
| error prefix `item_catalog: ` and the message `"item not found"` | `service_catalog: ` and `"service not found"` |
| `MigrateSpecialtiesAndItems` | `MigrateSpecialtiesAndServices` (same behaviour: seeds the canonical specialties and maps services to their specialty by the SKU prefix) |

The cross-module **readers keep their names and behaviour**: `ServiceExists(tenantId, id string)` and
`ServiceCard(...)` (their `type == S` filter disappears with the field: every row is a service). Their signatures do not
change, so `appointment_booking` keeps compiling against the new module path with only its import changed.

### The service (`ServiceModel`)

Same fields as `CatalogItemModel` **minus `type`**, plus one optional field:

| Field | Type / widget | Rules |
|---|---|---|
| `tax_treatment` | `input.Select()` of `TaxTaxed = "TAXED"`, `TaxExempt = "EXEMPT"`, Label "Tax treatment", `OmitEmpty` | optional; empty = not declared; any other value → `ValidationError`. The rate is not stored here |

Nothing in this module reads `tax_treatment`; it is carried so that a business that does issue a tax document for a
service can declare it. Everything else (`specialty_id` required, `sku` rules, price and currency, `is_active`) is
unchanged.

### Database note

There is no production data. Development databases are recreated: `Migrate` creates `specialty`, `service` and
`service_agreement` (in that FK order). It never drops the old `catalog_item` / `catalog_agreement` tables; say so in
`docs/ARCHITECTURE.md`.

## Stages

1. **Module path and package.** `go.mod` module path; package clause `servicecatalog` in every root file; update every
   import in `ui/`, `seed/`, `web/`, `migrate/` and `tests/`. Remove every `replace` from `go.mod` (they pointed at local
   checkouts); require the published versions (`go get` the latest published `form`, `components`, `widget`, `layout`).
2. **Model and ops.** `model.go` per the contract (delete the `type` machinery, add `tax_treatment`, rename); run `ormc` to
   regenerate `model_orm.go` (never edit it); `mcp.go` renames, filters and error text; `migration.go` rename.
3. **UI, seed, demo.** `ui/module.go` ID and icons (`IconCatalog = svg.Icon("service_catalog")`,
   `IconServices = svg.Icon("service_catalog_services")`, `IconSpecialties = svg.Icon("service_catalog_specialties")`, and
   the matching `ui/svg.go` sprite ids); `ui/browser.go` with no type radio and no `Type` in `NewRecord`; `seed/seed.go`
   creates services only; `web/client.go`.
4. **Tests** (`tests/`, same layout as today). Rename everything; **delete** the tests whose subject is the product type
   and the type filter; **add** `tax_treatment`: accepted values, empty accepted, an unknown value → `ValidationError`.
   `service_card_test.go` keeps asserting `ServiceCard` and `ServiceExists`.
5. **Translations.** `langc sync`; keep the existing Spanish; add the Spanish of `Tax treatment` ("Tratamiento tributario"),
   `Taxed` ("Afecto") and `Exempt` ("Exento"); remove the keys of the deleted radio (`Product`, `Service` as type labels) if
   `langc sync` reports them unused.
6. **Docs.** `README.md`, `docs/ARCHITECTURE.md` (scope: services and specialties only; the database note), `docs/diagrams/database.md`,
   and the `AGENTS.md` domain notes. `docs/ACCORDION_NAV_MASTER_PLAN.md` mentions the old name: leave it untouched (history).

## Delivery check

```
gotest                                                                          # vet, race, tests and wasm green
grep -rn "ItemType\|catalog_item\|itemcatalog\|item_catalog" --include=*.go --include=*.json . | grep -v "_orm.go" # → empty
grep -n '^replace' go.mod                                                       # → empty
git diff --stat main                                                            # only the files of the stages
```

## Stages table

| # | Stage | Files |
|---|---|---|
| 1 | module path, package, `go.mod` | every `*.go`, `go.mod`, `go.sum` |
| 2 | model and ops | `model.go`, `model_orm.go` (generated), `mcp.go`, `migration.go`, `view.go`, `migrate/migrate.go` |
| 3 | ui, seed, demo | `ui/*`, `seed/seed.go`, `web/client.go` |
| 4 | tests | `tests/*`, `migrate/migrate_test.go` |
| 5 | translations | `lang.json` |
| 6 | docs | `README.md`, `docs/ARCHITECTURE.md`, `docs/diagrams/database.md`, `AGENTS.md` |
