---
PLAN: "feat: migrate model.go to Kind API + agreements (ex-fonasa) child model"
TAG: v0.1.0
---

> This plan is dispatched via the CodeJob workflow. See skill: **agents-workflow**.

# PLAN — item_catalog: API `Kind` de `model` + convenios (agreements)

Eres un agente **sin contexto previo** y **solo tienes este repositorio** (`item_catalog`). Este
plan es autocontenido: todo contrato, regla y ejemplo está inline. No inventes diseño: cada decisión
ya está tomada aquí.

`item_catalog` va a **reemplazar a `service_catalog`** en la app consumidora (esa migración es OTRO
plan, en el repo de la app — no la hagas aquí). Aquí solo dejamos `item_catalog` listo: (1) migrado a
la API nueva de `github.com/tinywasm/model`, (2) con el campo `type` (servicio/producto — **ya existe**)
expuesto en formulario, y (3) con **convenios** (`agreement`) como definición hija con su tabla y su
CRUD MCP. El código ex-`fonasa_code` de `service_catalog` **no** vuelve como columna del item: ahora un
item puede tener **varios convenios**, y cada convenio lleva su aseguradora + código + precio.

---

## 1. Qué cambia y por qué

Dos cosas, independientes pero en el mismo plan:

**(A) Migración de API `model`.** Hoy `model.go` escribe `Type: model.FieldText` — un **literal del
enum** `FieldType`. La versión nueva (`model@v0.0.14`) cambió `Field.Type` de un enum a la **interfaz
`Kind`**: se rellena llamando a un **constructor** (`model.Text()`, `model.Int()`, …) o a un **widget**
(`input.Text()`, `input.Decimal()`, …). Es mecánico: mismos nombres de columna/tabla, mismo JSON.
Además el campo que el app renderiza como formulario **debe** llevar `input.X()` (un `Kind` con UI);
si se deja en `model.Text()` (Kind base, sin widget) el formulario sale **vacío en silencio** — el
mismo bug que ya se detectó y corrigió en `service_catalog`.

**(B) Convenios (`agreement`).** Nueva `model.Definition` hija con **FK escalar** al item
(`catalog_item_id`), su tabla, sus métodos de servicio y sus tools MCP (list / upsert / delete). Un
item tiene N convenios; cada convenio lleva `insurer` (aseguradora: FONASA, Isapre X), `code` (el
ex-`fonasa_code`), `price` (tarifa propia del convenio) e `is_active`.

También se migra `mcp.go`: `Tool.Action` pasó de `byte` (`'r'`) a `model.Action` (`model.Read`), y se
exportan **constantes de nombre de op** para que la app las importe (no las repita).

**Pilares tinywasm (innegociables):** cero `stdlib` en código que compila a WASM (este módulo es
**backend** y sus *tests* sí pueden usar `encoding/json` — **no** lo "corrijas"); cero strings
repetidos en lógica (nombres de op = constantes exportadas); cerrado por defecto (el cero de
`Tool.Access` ya es `AccessGuarded`).

## 2. Contrato de `github.com/tinywasm/model@v0.0.14` (inline)

`Field.Type` es la interfaz `Kind`. Se rellena con un constructor, **nunca** asignando un literal
`model.FieldText`:

```go
package model

type FieldType int
const (
    FieldText FieldType = iota // string
    FieldInt                   // int64
    FieldFloat                 // float64
    FieldBool                  // bool
    FieldBlob                  // []byte
    FieldStruct                // struct anidado — Kind = model.Struct(ref)
    FieldIntSlice
    FieldStructSlice           // []T anidado — Kind = model.StructSlice(ref)
    FieldRaw
)

// Kind reemplaza el par (enum Field.Type + Field.Widget). Implementaciones sin estado.
type Kind interface {
    Storage() FieldType          // mapeo determinista a Go/DDL
    Name() string                // "text", "int", "email", ...
    Validate(value string) error // SIEMPRE presente — fail-closed
}

// Constructores base — devuelven Kind, NO un literal FieldType:
func Text() Kind; func Int() Kind; func Float() Kind; func Bool() Kind; func Blob() Kind

type FieldDB struct {
    PK, Unique, AutoInc bool
    RefColumn string // columna PK referenciada en la tabla de Ref (vacío = auto-detecta el PK)
    OnDelete  string // vacío = default del generador (CASCADE)
}

type Field struct {
    Name      string
    Type      Kind        // model.Text(), input.Decimal(), ... — NUNCA un literal FieldType
    NotNull   bool
    OmitEmpty bool
    DB        *FieldDB    // nil = campo sin metadata DB (igual se persiste y viaja)
    Ref       *Definition // SOLO FK escalar; dispara la FK en DDL. No cambia el tipo Go (sigue escalar)
    Exclude   bool
    Permitted             // reglas de validación embebidas (chars/min/max)
}

type Fields = []Field
type Definition struct { Name string; Fields Fields }

// RBAC tipado — Tool.Action ahora es model.Action, no byte:
type Action uint8
const ( Create Action = 1 << iota; Read; Update; Delete )
```

**Mapeo fijo de tipos Go:** `Text()`→`string`, `Int()`→`int64`, `Float()`→`float64`, `Bool()`→`bool`.

**Convención de nombre:** la variable debe llamarse `<Struct>Model` (`AgreementModel` → genera
`type Agreement struct`).

**Widget = un `Kind` con UI** de `github.com/tinywasm/form/input` (`input.Text()`, `input.Decimal()`,
`input.Checkbox()`, `input.Number()`, `input.Textarea()`). También implementa `Storage()/Name()/
Validate()`. **Ya no existe `Field.Widget`.**

### FK escalar (patrón probado en `tinywasm/user`)

Un hijo con FK al padre se declara con `Ref` + `FieldDB.RefColumn` en la MISMA `Field` escalar:

```go
// en SessionModel (tinywasm/user), FK a UserModel:
{Name: "user_id", Type: model.Text(), DB: &model.FieldDB{RefColumn: "id"}, Ref: &UserModel},
```

`ormc` genera un campo escalar `UserId string` **y** la FK en DDL. El tipo Go sigue siendo `string`.

> **Fallback (regla de mecánica riesgosa):** si al regenerar `ormc` **rechaza** `Ref` o no emite la
> FK, deja el campo como columna escalar simple **sin** `Ref` (`{Name: "catalog_item_id", Type:
> model.Text(), NotNull: true}`) — la integridad se cuida en la lógica del módulo — y **repórtalo**.
> No inventes otra mecánica.

## 3. Estado actual (a portar)

`model.go` usa literales de enum (API vieja) — **no** compila contra `model@v0.0.14`:

```go
{Name: "id", Type: model.FieldText, DB: &model.FieldDB{PK: true}},   // ← model.FieldText ya no es un Kind
{Name: "price", Type: model.FieldFloat, NotNull: true},
{Name: "is_active", Type: model.FieldBool, NotNull: true},
{Name: "updated_at", Type: model.FieldInt, NotNull: true},
```

`mcp.go` usa `Action: 'r'` (byte) — **no** compila contra `mcp@v0.1.22` (espera `model.Action`):

```go
{ Name: "list_catalog_items", ..., Action: 'r', Execute: m.mcpListItems },
```

## 4. Estado objetivo

### 4.1 `model.go` reescrito

Preserva `package itemcatalog`. Importa `github.com/tinywasm/form/input` y `github.com/tinywasm/model`.

**Regla de widgets:** los modelos que el app renderiza como **formulario** (`CatalogItemModel`,
`AgreementModel`) llevan `input.X()` en cada campo editable. La **FK** `catalog_item_id` lleva
`model.Text()` + `Ref` (la fija el app desde el item padre, no se teclea). Los **DTO de argumentos**
(`ItemFilterModel`, `ListItemsArgsModel`, `GetItemArgsModel`, `FindBySKUArgsModel`,
`DeactivateItemArgsModel`, `DeleteItemArgsModel`, `ListAgreementsArgsModel`, `DeleteAgreementArgsModel`)
no se renderizan: llevan **Kinds base** `model.Text()`/`model.Bool()`/`model.Int()`.

```go
package itemcatalog

import (
	"github.com/tinywasm/form/input"
	"github.com/tinywasm/model"
)

// CatalogItem: producto o servicio. `type` == "S" (servicio) / "P" (producto).
// ⚠️ NO cambies los valores "S"/"P": appointment_booking.ServiceExists depende de type == "S".
var CatalogItemModel = model.Definition{
	Name: "catalog_item",
	Fields: model.Fields{
		{Name: "id", Type: input.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "tenant_id", Type: input.Text(), NotNull: true},
		{Name: "sku", Type: input.Text(), NotNull: true},
		{Name: "name", Type: input.Text(), NotNull: true},
		{Name: "description", Type: input.Textarea()},
		{Name: "category", Type: input.Text()},
		{Name: "type", Type: input.Text(), NotNull: true}, // "S" servicio / "P" producto
		{Name: "price", Type: input.Decimal(), NotNull: true},
		{Name: "currency", Type: input.Text(), NotNull: true},
		{Name: "is_active", Type: input.Checkbox(), NotNull: true},
		{Name: "updated_at", Type: input.Number(), NotNull: true},
	},
}

// Agreement (convenio): un item tiene N convenios. Cada uno con su aseguradora, su código
// (ex-fonasa_code) y su tarifa propia. FK a catalog_item.
var AgreementModel = model.Definition{
	Name: "catalog_agreement",
	Fields: model.Fields{
		{Name: "id", Type: input.Text(), DB: &model.FieldDB{PK: true}},
		{Name: "tenant_id", Type: input.Text(), NotNull: true},
		{Name: "catalog_item_id", Type: model.Text(), NotNull: true, DB: &model.FieldDB{RefColumn: "id"}, Ref: &CatalogItemModel}, // FK — la fija el app
		{Name: "insurer", Type: input.Text(), NotNull: true}, // aseguradora: "FONASA", "Isapre X"
		{Name: "code", Type: input.Text()},                   // ex fonasa_code: código de facturación del convenio
		{Name: "price", Type: input.Decimal()},               // tarifa propia del convenio (opcional)
		{Name: "is_active", Type: input.Checkbox(), NotNull: true},
		{Name: "updated_at", Type: input.Number(), NotNull: true},
	},
}

var ItemFilterModel = model.Definition{
	Name: "item_filter",
	Fields: model.Fields{
		{Name: "type", Type: model.Text()},
		{Name: "active_only", Type: model.Bool()},
		{Name: "limit", Type: model.Int()},
		{Name: "offset", Type: model.Int()},
	},
}

var ListItemsArgsModel = model.Definition{
	Name: "list_items_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "type", Type: model.Text()},
		{Name: "active_only", Type: model.Bool()},
		{Name: "limit", Type: model.Int()},
		{Name: "offset", Type: model.Int()},
	},
}

var GetItemArgsModel = model.Definition{
	Name: "get_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}

var FindBySKUArgsModel = model.Definition{
	Name: "find_by_sku_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "sku", Type: model.Text()},
	},
}

var DeactivateItemArgsModel = model.Definition{
	Name: "deactivate_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}

var DeleteItemArgsModel = model.Definition{
	Name: "delete_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}

var ListAgreementsArgsModel = model.Definition{
	Name: "list_agreements_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "catalog_item_id", Type: model.Text()},
	},
}

var DeleteAgreementArgsModel = model.Definition{
	Name: "delete_agreement_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}
```

**Conserva** en `model.go` las interfaces existentes (`UIAdapter`, `EventPublisher`, `CatalogService`)
tal cual. Añade a `CatalogService` los tres métodos de convenio (§4.3).

### 4.2 `mcp.go` — Action tipado + constantes de op + upsert de item

1. Importa `github.com/tinywasm/model`.
2. Cambia cada `Action: '<letra>'` por la `model.Action` tipada, según esta tabla **exacta**:

| Tool | Antes | Ahora |
|---|---|---|
| `list_catalog_items` | `'r'` | `model.Read` |
| `get_catalog_item` | `'r'` | `model.Read` |
| `find_item_by_sku` | `'r'` | `model.Read` |
| `create_catalog_item` | `'c'` | `model.Create` |
| `update_catalog_item` | `'u'` | `model.Update` |
| `deactivate_catalog_item` | `'u'` | `model.Update` |
| `delete_catalog_item` | `'d'` | `model.Delete` |

3. Añade el bloque de **constantes de nombre de op exportadas** (cero strings repetidos; la app las
   importará). Reemplaza los literales `"list_catalog_items"` etc. en `Tools()` por estas constantes:

```go
const (
	OpListItems      = "list_catalog_items"
	OpGetItem        = "get_catalog_item"
	OpFindItemBySKU  = "find_item_by_sku"
	OpCreateItem     = "create_catalog_item"
	OpUpdateItem     = "update_catalog_item"
	OpUpsertItem     = "upsert_catalog_item"
	OpDeactivateItem = "deactivate_catalog_item"
	OpDeleteItem     = "delete_catalog_item"

	OpListAgreements  = "list_agreements"
	OpUpsertAgreement = "upsert_agreement"
	OpDeleteAgreement = "delete_agreement"
)
```

4. Añade el tool **`upsert_catalog_item`** (la app usa UN solo "save": crear-o-actualizar según `Id`).
   Espeja el patrón de `service_catalog`:

```go
// en Tools(), un item más en el slice:
{
	Name:        OpUpsertItem,
	Description: "Create or update a catalog item (create if id is empty)",
	Args:        &CatalogItem{},
	Resource:    "catalog_item",
	Action:      model.Create,
	Execute:     m.mcpUpsertItem,
},

func (m *Module) mcpUpsertItem(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var item CatalogItem
	if err := json.Decode(req.Params.Arguments, &item); err != nil {
		return nil, err
	}
	var out CatalogItem
	var err error
	if item.Id == "" {
		out, err = m.CreateItem(item)
	} else {
		out, err = m.UpdateItem(item)
	}
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	var res string
	if err := json.Encode(&out, &res); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text(res), nil
}
```

5. Rellena `Args:` en los tools existentes con su DTO (schema completo para `tools/list`):
   `list_catalog_items`→`&ListItemsArgs{}`, `get_catalog_item`→`&GetItemArgs{}`,
   `find_item_by_sku`→`&FindBySKUArgs{}`, `create_catalog_item`/`update_catalog_item`→`&CatalogItem{}`,
   `deactivate_catalog_item`→`&DeactivateItemArgs{}`, `delete_catalog_item`→`&DeleteItemArgs{}`.

### 4.3 `mcp.go` — convenios: tabla, servicio y tools

En `New()`, crea también la tabla del convenio (después de la del item):

```go
if err := db.CreateTable(&Agreement{}); err != nil {
	return nil, err
}
```

Métodos de servicio (añádelos al `*Module` y a la interfaz `CatalogService`):

```go
func (m *Module) ListAgreements(tenantId, catalogItemId string) ([]Agreement, error) {
	var a Agreement
	qb := m.db.Query(&a).Where(Agreement_.TenantId).Eq(tenantId)
	if catalogItemId != "" {
		qb = qb.Where(Agreement_.CatalogItemId).Eq(catalogItemId)
	}
	results, err := ReadAllAgreement(qb)
	if err != nil {
		return nil, err
	}
	items := make([]Agreement, len(results))
	for i, r := range results {
		items[i] = *r
	}
	return items, nil
}

func (m *Module) UpsertAgreement(a Agreement) (Agreement, error) {
	a.UpdatedAt = time.Now()
	if a.Id == "" {
		a.Id = m.uid.GetNewID()
		if err := m.db.Create(&a); err != nil {
			return Agreement{}, err
		}
		m.publish("catalog.agreement.created", a)
		return a, nil
	}
	if err := m.db.Update(&a, orm.Eq(Agreement_.Id, a.Id)); err != nil {
		return Agreement{}, err
	}
	m.publish("catalog.agreement.updated", a)
	return a, nil
}

func (m *Module) DeleteAgreement(tenantId, id string) error {
	a := Agreement{Id: id, TenantId: tenantId}
	if err := m.db.Delete(&a, orm.Eq(Agreement_.Id, id)); err != nil {
		return err
	}
	m.publish("catalog.agreement.deleted", map[string]string{"tenant_id": tenantId, "id": id})
	return nil
}
```

Tools de convenio (añádelos al slice de `Tools()`):

```go
{
	Name:        OpListAgreements,
	Description: "List agreements (convenios) of a catalog item",
	Args:        &ListAgreementsArgs{},
	Resource:    "catalog_agreement",
	Action:      model.Read,
	Execute:     m.mcpListAgreements,
},
{
	Name:        OpUpsertAgreement,
	Description: "Create or update an agreement (create if id is empty)",
	Args:        &Agreement{},
	Resource:    "catalog_agreement",
	Action:      model.Create,
	Execute:     m.mcpUpsertAgreement,
},
{
	Name:        OpDeleteAgreement,
	Description: "Delete an agreement",
	Args:        &DeleteAgreementArgs{},
	Resource:    "catalog_agreement",
	Action:      model.Delete,
	Execute:     m.mcpDeleteAgreement,
},
```

Handlers MCP (espejan los del item; usa `AgreementList` generado por `ormc` para el listado):

```go
func (m *Module) mcpListAgreements(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args ListAgreementsArgs
	if err := json.Decode(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	items, err := m.ListAgreements(args.TenantId, args.CatalogItemId)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	list := make(AgreementList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	var res string
	if err := json.Encode(&list, &res); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text(res), nil
}

func (m *Module) mcpUpsertAgreement(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var a Agreement
	if err := json.Decode(req.Params.Arguments, &a); err != nil {
		return nil, err
	}
	out, err := m.UpsertAgreement(a)
	if err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	var res string
	if err := json.Encode(&out, &res); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text(res), nil
}

func (m *Module) mcpDeleteAgreement(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
	var args DeleteAgreementArgs
	if err := json.Decode(req.Params.Arguments, &args); err != nil {
		return nil, err
	}
	if err := m.DeleteAgreement(args.TenantId, args.Id); err != nil {
		return &mcp.Result{IsError: true, Content: err.Error()}, nil
	}
	return mcp.Text("agreement deleted"), nil
}
```

> El `ItemFilter` (struct generado) sigue teniendo el campo `Type` — no lo toques. `ServiceExists`
> **no cambia**: sigue devolviendo `item.Type == "S" && item.IsActive`.

## 5. Pasos

> **Dependencias (set conocido-bueno, el mismo de `service_catalog@v0.0.4`):**
> `go get github.com/tinywasm/model@v0.0.14 github.com/tinywasm/orm@v0.9.28 github.com/tinywasm/mcp@v0.1.22 github.com/tinywasm/form@v0.2.15 github.com/tinywasm/json@v0.5.11`
> luego `go mod tidy` (resuelve `fmt`/`context`/`time`/`unixid`). `form` pasa a dependencia **directa**
> (antes no se importaba; ahora `model.go` usa `form/input`).

1. Reescribe `model.go` con §4.1 (Kinds/widgets + `AgreementModel` + DTOs de convenio). No dejes
   ningún `model.Field<Tipo>` (literal de enum) en el archivo.
2. Instala y corre el generador: `go install github.com/tinywasm/orm/cmd/ormc@latest` y ejecuta `ormc`
   en la raíz del módulo. Regenera `model_orm.go` con el struct `Agreement` (`CatalogItemId string`,
   `Insurer`, `Code`, `Price float64`, …), su `AgreementList`, `Agreement_` (columnas) y
   `ReadAllAgreement`/`ReadOneAgreement`. Verifica que la FK a `catalog_item` aparezca en el DDL (si no,
   aplica el **fallback** de §2 y repórtalo).
3. Edita `mcp.go`: importa `model`; aplica la tabla de Action (§4.2·2); añade el bloque de constantes
   `Op*` y reemplaza los literales en `Tools()`; añade `upsert_catalog_item` (§4.2·4); rellena `Args:`
   (§4.2·5); crea la tabla `Agreement` en `New()` y añade servicio + tools + handlers de convenio (§4.3).
4. Ajusta consumidores/tests: `catalog_test.go` (raíz, `package itemcatalog`) sigue válido salvo
   compilación; corre `gotest ./...` y corrige lo mínimo. **Añade** un test de CRUD de convenio
   (`UpsertAgreement` con `Id==""` crea; con `Id` actualiza; `ListAgreements(tenant, itemID)` filtra por
   item; `DeleteAgreement` borra) usando `sqlite.Open(":memory:")` como el test actual.
5. Sube versiones del submódulo `tests/` (`tests/go.mod`) al mismo set y verifica que compile.
6. Docs: actualiza `docs/ARCHITECTURE.md` y `docs/diagrams/database.md` para incluir la tabla
   `catalog_agreement` (FK a `catalog_item`, N convenios por item) y el nuevo grupo de tools.

## 6. Fuera de alcance

- **No** hagas la migración de la app (repo consumidor) — es otro plan.
- **No** renombres tipos/columnas existentes del item ni cambies su comportamiento (`ServiceExists`
  intacto; valores `type` "S"/"P" intactos).
- **No** conviertas `type` en `input.Select()` con opciones: fuera de alcance (queda `input.Text()`).
- **No** "corrijas" el `encoding/json` de los *tests*: este módulo es backend y sus tests legítimamente
  usan stdlib.
- **No** borres los tools existentes del item (create/update/get/find/deactivate/delete): `upsert` es
  **aditivo**.

## 7. Criterios de aceptación

- `gotest ./...` verde con `go.mod` en `model v0.0.14` / `orm v0.9.28` / `mcp v0.1.22` / `form v0.2.15`.
- `grep -rn "model.Field[A-Z]" .` (excluyendo `model_orm.go` generado) **vacío**: no quedan literales de
  enum; todo es `model.X()` o `input.X()`.
- `grep -rn "Action: '" .` **vacío**: no quedan Action byte; todos son `model.Read/Create/Update/Delete`.
- `CatalogItemModel` y `AgreementModel` tienen `input.X()` en cada campo editable; `catalog_item_id`
  usa `model.Text()` + `Ref: &CatalogItemModel`.
- `model_orm.go` regenerado incluye `Agreement`, `AgreementList`, `Agreement_`, `ReadAllAgreement`.
- Existen y se exportan las constantes `Op*` (§4.2·3); `Tools()` no usa literales de nombre de op.
- `New()` crea las tablas `catalog_item` **y** `catalog_agreement`.
- Un test verifica el CRUD de convenio (upsert crea/actualiza, list filtra por item, delete borra).

## 8. Etapas

| # | Etapa | Archivos | Criterio |
|---|---|---|---|
| 1 | Bump deps | `go.mod`, `tests/go.mod` | resuelven; `form` directa |
| 2 | Reescribir `model.go` | `model.go` | sin literales de enum; `AgreementModel` + FK; widgets |
| 3 | Regenerar | `model_orm.go` | struct `Agreement` + plomería + FK en DDL |
| 4 | Migrar `mcp.go` | `mcp.go` | Action tipado, `Op*`, `upsert_catalog_item`, convenios (servicio+tools+handlers) |
| 5 | Tests | `catalog_test.go`, `tests/` | `gotest ./...` verde + test CRUD convenio |
| 6 | Docs | `docs/ARCHITECTURE.md`, `docs/diagrams/database.md` | reflejan `catalog_agreement` |
