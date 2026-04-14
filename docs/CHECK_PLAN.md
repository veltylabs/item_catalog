# Build item-catalog module

## Context

New module for the Velty ecosystem that manages a unified catalog of services (`S`) and products (`P`).
It is the source of truth that other modules reference — notably `appointment-booking` via its `CatalogReader` interface.

Reference implementation: `client-monjitaschillan-platform/modules/catalog` (uses memorydb + tinywasm/components directly).
This module replaces that coupling with injectable ports for storage (`tinywasm/orm`), events (`tinywasm/sse`-compatible interface), and UI (`UIAdapter` interface).

---

## Architecture Decisions

### Single table, type discriminator
One `CatalogItem` table with `type: "S" | "P"`. Services and products share the same data shape at the catalog level. Behavioral differences (duration, scheduling) live in modules that consume the catalog (e.g., `appointment-booking.EmployeeServiceConfig`).

### Multi-tenant
Every entity has `tenant_id`. SKU uniqueness is scoped per tenant: composite `(tenant_id, sku)` — enforced in the service layer since SQLite does not support composite UNIQUE without an explicit index (add `db:"unique"` on the composite via ormc tags or enforce in `CreateItem`).

### Currency per item
`price float` + `currency string` (ISO 4217) per item. Industry standard (Stripe, Shopify, SAP). Enables mixed-currency catalogs without schema migration.

### UIAdapter port
The module defines a `UIAdapter` interface. The host app (which knows `tinywasm/dom`, `tinywasm/components`, etc.) implements it and injects it. The module calls the adapter for any view-layer operation, making UI logic testable with a mock renderer that returns plain strings.

### EventPublisher port
Compatible with `tinywasm/sse` API but not coupled to it. The module defines its own `EventPublisher` interface. Passing `nil` disables events (safe for tests and CLI tools).

### CatalogReader compatibility
`*Module` implements the `CatalogReader` interface defined in `appointment-booking`:
```go
type CatalogReader interface {
    ServiceExists(tenantID, serviceID string) (bool, error)
}
```
This allows the composition root to wire:
```go
cat := itemcatalog.New(db, deps)
appt := appointmentbooking.New(db, appointmentbooking.Deps{Catalog: cat, ...})
```

---

## Data Model

**File**: `model.go`

```go
// CatalogItem represents a service (S) or product (P) in the catalog.
type CatalogItem struct {
    ID          string  `db:"pk"`              // unixid — timestamp-ordered
    TenantID    string  `db:"not_null"`
    SKU         string  `db:"not_null"`        // unique per tenant — enforced in service layer
    Name        string  `db:"not_null"`
    Description string                          // optional — aids LLM context in MCP tools
    Type        string  `db:"not_null"`        // "S" = Service | "P" = Product
    Price       float64 `db:"not_null"`        // base price (can be overridden in appointment-booking)
    Currency    string  `db:"not_null"`        // ISO 4217: "CLP", "USD", etc.
    IsActive    bool    `db:"not_null"`        // false = soft-deleted; preserved for referential integrity
    UpdatedAt   int64   `db:"not_null"`        // Unix UTC — managed by service layer before db.Create/Update
}

func (c *CatalogItem) ModelName() string { return "catalog_item" }
```

**Interfaces** (also in `model.go`, build-tag-free):

```go
// UIAdapter — port for presentation. Implemented by the host app.
// The module calls these methods; it never imports tinywasm/dom or components directly.
type UIAdapter interface {
    RenderItemList(items []CatalogItem, activeFilter string) string
    RenderItemForm(item *CatalogItem) string  // nil = empty create form
    RenderFilterSelector(current string) string
}

// EventPublisher — compatible with tinywasm/sse but not coupled to it.
// Pass nil to disable event publishing.
type EventPublisher interface {
    Publish(event string, payload any) error
}

// CatalogService — the core business interface. Implemented by *Module.
type CatalogService interface {
    GetItem(tenantID, id string) (CatalogItem, error)
    FindBySKU(tenantID, sku string) (CatalogItem, error)
    ListItems(tenantID string, filter ItemFilter) ([]CatalogItem, error)
    CreateItem(item CatalogItem) (CatalogItem, error)
    UpdateItem(item CatalogItem) (CatalogItem, error)
    DeactivateItem(tenantID, id string) error
    DeleteItem(tenantID, id string) error
    ServiceExists(tenantID, serviceID string) (bool, error) // implements appointment-booking.CatalogReader
}

// ItemFilter for list queries.
type ItemFilter struct {
    Type     string // "S" | "P" | "" (all)
    ActiveOnly bool
    Limit    int
    Offset   int
}
```

---

## Module Deps

**File**: `mcp.go`

> El tag `//go:build !wasm` va en este archivo porque importa `tinywasm/orm` y `tinywasm/unixid` — paquetes incompatibles con WASM. `model.go` no lleva tag porque solo define structs e interfaces que deben estar disponibles en ambos entornos.

```go
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
```

---

## Stage 1 — go.mod dependencies

Add to `go.mod`:

```
github.com/tinywasm/fmt     v0.22.2
github.com/tinywasm/orm     v0.6.0
github.com/tinywasm/sqlite  (latest — test only)
github.com/tinywasm/unixid  (latest)
```

Run `go mod tidy` after editing.

---

## Stage 2 — `model.go`

Create `model.go` (no build tag — interfaces and struct must be available in WASM too):

- `CatalogItem` struct with `db:` tags as shown above
- `UIAdapter` interface
- `EventPublisher` interface
- `CatalogService` interface
- `ItemFilter` struct

---

## Stage 3 — `model_orm.go` (auto-generated)

Run `ormc` after `model.go` is created:

```bash
go install github.com/tinywasm/orm/cmd/ormc@latest
ormc
```

Expected output:
- `ModelName() string`
- `Schema() []fmt.Field` with `DB: &fmt.FieldDB{PK: true}` for `id`, `NotNull: true` for required fields
- `Pointers() []any`
- `var CatalogItem_ = struct{...}` meta struct
- `ReadOneCatalogItem(qb *orm.QB, model *CatalogItem) (*CatalogItem, error)`
- `ReadAllCatalogItem(qb *orm.QB) ([]*CatalogItem, error)`

> No `Values()` — not part of `fmt.Model`.

---

## Stage 4 — `mcp.go` (server-side logic + MCP tools)

**File**: `mcp.go`

Este archivo lleva `//go:build !wasm` únicamente porque importa `tinywasm/orm` (`*orm.DB`) y `tinywasm/unixid`, que son dependencias que no corren en WASM. El tag no va porque sea un archivo "MCP" sino porque tiene código que depende de paquetes backend. La regla general: solo los archivos que importan paquetes incompatibles con WASM reciben el tag.

### 4.1 — Service methods

Implement all `CatalogService` methods on `*Module`:

| Method | Notes |
|--------|-------|
| `GetItem(tenantID, id)` | read by PK; return `ErrNotFound` if not found or wrong tenant |
| `FindBySKU(tenantID, sku)` | filter by `tenant_id` + `sku`; return `ErrNotFound` if none |
| `ListItems(tenantID, filter)` | filter by tenant; optionally by type and `is_active`; apply limit/offset |
| `CreateItem(item)` | validate SKU uniqueness per tenant; set `ID = uid.GetNewID()`, `UpdatedAt = now`; `db.Create` |
| `UpdateItem(item)` | verify item exists and belongs to tenant; set `UpdatedAt = now`; `db.Update` |
| `DeactivateItem(tenantID, id)` | set `IsActive = false`, `UpdatedAt = now`; `db.Update` |
| `DeleteItem(tenantID, id)` | hard delete; verify tenant ownership first |
| `ServiceExists(tenantID, serviceID)` | returns true if item exists, is type `"S"`, and `is_active = true` |

Publish events after each successful mutation:
```go
func (m *Module) publish(event string, payload any) {
    if m.pub != nil {
        _ = m.pub.Publish(event, payload)  // fire-and-forget; error never fails the operation
    }
}
```

Events:
- `catalog.item.created` — payload: full `CatalogItem`
- `catalog.item.updated` — payload: full `CatalogItem`
- `catalog.item.deactivated` — payload: `{tenant_id, id}`
- `catalog.item.deleted` — payload: `{tenant_id, id}`

### 4.2 — MCP ToolProvider

Implement `Tools() []mcp.Tool` on `*Module`:

| Tool name | Action | RBAC resource | Handler |
|-----------|--------|---------------|---------|
| `list_catalog_items` | `'r'` | `catalog_item` | `m.mcpListItems` |
| `get_catalog_item` | `'r'` | `catalog_item` | `m.mcpGetItem` |
| `find_item_by_sku` | `'r'` | `catalog_item` | `m.mcpFindBySKU` |
| `create_catalog_item` | `'c'` | `catalog_item` | `m.mcpCreateItem` |
| `update_catalog_item` | `'u'` | `catalog_item` | `m.mcpUpdateItem` |
| `deactivate_catalog_item` | `'u'` | `catalog_item` | `m.mcpDeactivateItem` |
| `delete_catalog_item` | `'d'` | `catalog_item` | `m.mcpDeleteItem` |

Handler signature: `func(ctx *context.Context, req mcp.Request) (*mcp.Result, error)`

Decode arguments: `json.Unmarshal([]byte(req.Params.Arguments), &args)`

Result helpers:
```go
mcp.Text(string)                               // success
&mcp.Result{IsError: true, Content: "msg"}    // error
```

### 4.3 — UI methods (on `*Module`)

Only called when `m.ui != nil`:

```go
func (m *Module) RenderList(tenantID, filter string) string {
    items, _ := m.ListItems(tenantID, ItemFilter{Type: filter, ActiveOnly: true})
    return m.ui.RenderItemList(items, filter)
}

func (m *Module) RenderForm(tenantID, id string) string {
    if id == "" {
        return m.ui.RenderItemForm(nil)
    }
    item, _ := m.GetItem(tenantID, id)
    return m.ui.RenderItemForm(&item)
}

func (m *Module) RenderFilter(current string) string {
    return m.ui.RenderFilterSelector(current)
}
```

---

## Stage 5 — Tests

**File**: `tests/setup_test.go` + `tests/catalog_test.go`

Use `tinywasm/sqlite` in-memory for realistic tests. Mock `UIAdapter` returns plain strings. `EventPublisher` mock records calls.

Test coverage:
- `CreateItem` — success, duplicate SKU error
- `FindBySKU` — found, not found, wrong tenant isolation
- `ListItems` — by type, active-only filter
- `UpdateItem` — success, wrong tenant rejection
- `DeactivateItem` — sets `is_active=false`, emits event
- `DeleteItem` — hard delete, verifies row gone
- `ServiceExists` — true for active service, false for product or inactive
- MCP tools — one test per tool validating JSON decode + result shape
- `UIAdapter` — mock verifies `RenderItemList` called with correct items

---

## Stage 6 — Documentation

Create:

- `docs/ARCHITECTURE.md` — domain scope, entities, patterns (Hexagonal ports), MCP tools table, composition root example
- `docs/SKILL.md` — LLM-friendly summary for agents consuming this module
- `docs/diagrams/database.md` — Mermaid ERD for `catalog_item` table
- `README.md` — quick start, MCP tools table, key files table

Update `appointment-booking/docs/ARCHITECTURE.md` Section 7 (Composition Root) to show `itemcatalog.New(db, deps)` as the `CatalogReader` implementation.

---

## Verification

```bash
go build ./...
go test ./...
```

Expected: zero errors, all tests pass.
