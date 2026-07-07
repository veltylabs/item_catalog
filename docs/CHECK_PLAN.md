# PLAN — item_catalog: migrar model.go a model.Definition

> This plan is dispatched via the CodeJob workflow. See skill: **agents-workflow**.

✅ **Desbloqueado.** `github.com/tinywasm/orm@v0.9.24` (con `fmt@v0.25.1`) genera el helper
`<Struct>_.Campo` de forma automática (always-on) para modelos con DB. Ojo con el cambio de casing
puro (`Id`/`Sku`/`TenantId`, no `ID`/`SKU`/`TenantID`) descrito en §5.3.

Eres un agente **sin contexto previo** y **solo tienes este repositorio** (`item_catalog`). Plan
autocontenido: todo contrato, regla y ejemplo está inline.

---

## 1. Qué cambia y por qué

El ecosistema tinywasm invirtió la generación de modelos: se escribe una definición tipada
(`model.Definition`) a mano, y `ormc` genera el struct concreto + plomería. Migración **mecánica**:
mismo comportamiento, mismas columnas/tabla, mismo JSON.

## 2. Contrato de `github.com/tinywasm/model` (inline)

```go
package model

type FieldType int
const (
    FieldText FieldType = iota // string
    FieldInt                   // int64
    FieldFloat                 // float64
    FieldBool                  // bool
    FieldBlob                  // []byte
    FieldStruct                // struct anidado — requiere Ref
    FieldIntSlice               // []int
    FieldStructSlice            // []T anidado — requiere Ref
    FieldRaw                    // JSON pre-serializado
)

type FieldDB struct { PK, Unique, AutoInc bool }

type Field struct {
    Name      string
    Type      FieldType
    NotNull   bool
    OmitEmpty bool
    Widget    Widget      // nil = sin UI (no usado en este módulo)
    DB        *FieldDB    // nil = sin persistencia (args/DTOs de transporte)
    Ref       *Definition
    Exclude   bool
    Permitted
}

type Fields = []Field

type Definition struct {
    Name   string
    Fields Fields
}
```

Mapeo fijo: `FieldText`→`string`, `FieldInt`→`int64`, `FieldFloat`→`float64`, `FieldBool`→`bool`.
Variable de definición debe llamarse `<Struct>Model`.

---

## 3. Estado actual (`model.go`, a portar — solo los structs con campos; las interfaces no cambian)

```go
package itemcatalog

// orm:typed_fields
// CatalogItem represents a service (S) or product (P) in the catalog.
type CatalogItem struct {
	ID          string  `db:"pk"` // unixid — timestamp-ordered
	TenantID    string  `db:"not_null"`
	SKU         string  `db:"not_null"` // unique per tenant — enforced in service layer
	Name        string  `db:"not_null"`
	Description string  // optional — aids LLM context in MCP tools
	Category    string  // optional grouping label — free text, not enforced
	Type        string  `db:"not_null"` // "S" = Service | "P" = Product
	Price       float64 `db:"not_null"`
	Currency    string  `db:"not_null"` // ISO 4217
	IsActive    bool    `db:"not_null"`
	UpdatedAt   int64   `db:"not_null"`
}

// ItemFilter for list queries.
type ItemFilter struct {
	Type       string
	ActiveOnly bool
	Limit      int
	Offset     int
}

type ListItemsArgs struct {
	TenantID   string
	Type       string
	ActiveOnly bool
	Limit      int
	Offset     int
}

type GetItemArgs struct {
	TenantID string
	ID       string
}

type FindBySKUArgs struct {
	TenantID string
	SKU      string
}

type DeactivateItemArgs struct {
	TenantID string
	ID       string
}

type DeleteItemArgs struct {
	TenantID string
	ID       string
}
```

Las interfaces `UIAdapter`, `EventPublisher`, `CatalogService` (puertos, sin campos) **NO** se tocan —
solo migran structs de datos.

**Nota de migración de tipo:** `ItemFilter.Limit/Offset` y `ListItemsArgs.Limit/Offset` son hoy `int`
(32-bit); con el mapeo fijo `FieldInt`→`int64` pasan a `int64`. Revisa el uso de estos campos (SQL
`LIMIT`/`OFFSET`) para conversiones si algún driver exige `int`.

## 4. Estado objetivo (`model.go` reescrito)

```go
package itemcatalog

import "github.com/tinywasm/model"

var CatalogItemModel = model.Definition{
	Name: "catalog_item",
	Fields: model.Fields{
		{Name: "id", Type: model.FieldText, DB: &model.FieldDB{PK: true}},
		{Name: "tenant_id", Type: model.FieldText, NotNull: true},
		{Name: "sku", Type: model.FieldText, NotNull: true},
		{Name: "name", Type: model.FieldText, NotNull: true},
		{Name: "description", Type: model.FieldText},
		{Name: "category", Type: model.FieldText},
		{Name: "type", Type: model.FieldText, NotNull: true},
		{Name: "price", Type: model.FieldFloat, NotNull: true},
		{Name: "currency", Type: model.FieldText, NotNull: true},
		{Name: "is_active", Type: model.FieldBool, NotNull: true},
		{Name: "updated_at", Type: model.FieldInt, NotNull: true},
	},
}

var ItemFilterModel = model.Definition{
	Name: "item_filter",
	Fields: model.Fields{
		{Name: "type", Type: model.FieldText},
		{Name: "active_only", Type: model.FieldBool},
		{Name: "limit", Type: model.FieldInt},
		{Name: "offset", Type: model.FieldInt},
	},
}

var ListItemsArgsModel = model.Definition{
	Name: "list_items_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.FieldText},
		{Name: "type", Type: model.FieldText},
		{Name: "active_only", Type: model.FieldBool},
		{Name: "limit", Type: model.FieldInt},
		{Name: "offset", Type: model.FieldInt},
	},
}

var GetItemArgsModel = model.Definition{
	Name: "get_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.FieldText},
		{Name: "id", Type: model.FieldText},
	},
}

var FindBySKUArgsModel = model.Definition{
	Name: "find_by_sku_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.FieldText},
		{Name: "sku", Type: model.FieldText},
	},
}

var DeactivateItemArgsModel = model.Definition{
	Name: "deactivate_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.FieldText},
		{Name: "id", Type: model.FieldText},
	},
}

var DeleteItemArgsModel = model.Definition{
	Name: "delete_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.FieldText},
		{Name: "id", Type: model.FieldText},
	},
}
```

## 5. Pasos

> **Dependencias:** `go get github.com/tinywasm/orm@v0.9.24` (arrastra `github.com/tinywasm/fmt@v0.25.1`).
> Reconstruye `ormc` desde esa versión antes de regenerar.

1. Reescribe `model.go` con el contenido de §4. Las interfaces (`UIAdapter`, `EventPublisher`,
   `CatalogService`) quedan intactas en el mismo archivo o donde estén hoy.
   **Sin ninguna anotación** — la directiva `// orm:typed_fields` **ya no existe** (removida en orm
   v0.9.24).
2. Regenera `model_orm.go` con el ormc de v0.9.24.
3. **El helper de campos tipados ahora es automático (always-on) para todo modelo con DB.** ormc emite:
   ```go
   var CatalogItem_ = struct{ Id, TenantId, Sku, Name, ... string }{ Id: "id", TenantId: "tenant_id", Sku: "sku", ... }
   ```
   ⚠️ **El casing cambió a algorítmico puro (sin diccionario de acrónimos):** `id`→`Id`, `sku`→`Sku`,
   `tenant_id`→`TenantId` (ya **no** `ID`/`SKU`/`TenantID`). Esto afecta **dos** superficies:
   - **El helper:** `CatalogItem_.SKU` → `CatalogItem_.Sku`, `.ID` → `.Id`, `.TenantID` → `.TenantId`.
   - **Los campos del struct generado:** `CatalogItem.ID/SKU/TenantID` → `CatalogItem.Id/Sku/TenantId`.
     Como `CatalogItem` es el tipo público del módulo, actualiza **todas** las referencias
     (`item.ID`→`item.Id`, `item.SKU`→`item.Sku`, etc.) en `mcp.go`, `view.go`, tests y cualquier
     consumidor. Las **claves JSON/columnas del wire NO cambian** (siguen snake).
4. Ajusta `mcp.go`/tests para el nuevo casing y para los tipos `Limit`/`Offset` que pasan a `int64`
   (mapeo fijo `FieldInt`→`int64`); añade conversiones donde el compilador lo exija.

## 6. Fuera de alcance

- No tocar las interfaces (`UIAdapter`, `EventPublisher`, `CatalogService`).
- No cambiar nombres de tabla/columna ni las claves JSON del wire (siguen snake).
- **No añadir** la directiva `// orm:typed_fields` (ya no existe) ni ningún opt-in.
- No parchear/forkear `ormc` — el fix está publicado aguas arriba (orm v0.9.24).

## 7. Criterio de aceptación

- `gotest ./...` verde con `go.mod` en `orm v0.9.24` / `fmt v0.25.1`.
- `model_orm.go` regenerado compila, incluyendo `var CatalogItem_ = struct{...}{...}` con casing puro
  (`Id`, `Sku`, `TenantId`, …) y valores = columnas snake.
- `mcp.go` (y demás consumidores) usan el nuevo casing: `CatalogItem_.{Id,Sku,TenantId,Type,IsActive}`
  y `item.Id`/`item.Sku`/`item.TenantId`.
- `Limit`/`Offset` son `int64` en todo el código consumidor.
- No queda struct plano con tags `db:` ni directiva `ormc:`/`orm:` en `model.go`.

## 8. Etapas

| # | Etapa | Salida | Criterio |
|---|---|---|---|
| 1 | `go get` orm v0.9.24 + reescribir `model.go` | Definitions de §4, sin directiva | compila |
| 2 | Regenerar `model_orm.go` | struct + plomería + `CatalogItem_` (automático) | helper presente con casing puro |
| 3 | Actualizar casing + `int64` en callers | `mcp.go`/`view.go`/tests | `gotest ./...` verde |
