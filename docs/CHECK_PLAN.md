---
PLAN: "feat: item_catalog joins the reusable-module harness (OpModule, IDGenerator, events.Publisher, view)"
TAG: v0.2.0
---

> Este plan se despacha vía el flujo CodeJob. Ver skill: **agents-workflow**.
> Orquestado por `tinywasm/app-releases/docs/REUSABLE_MODULES_MASTER_PLAN.md` — **Fase C, el piloto**.

# PLAN — `item_catalog`: arnés de módulo reutilizable

Autocontenido, en español. Eres un agente **sin contexto previo** y **solo tienes este repo**
(`veltylabs/item_catalog`). Todo el contrato y el código exacto van inline.

> **Prerequisito:** la API `Kind` de `model` + los 11 `Op*`/tools + el modelo `Agreement` (convenios)
> **ya están implementados** en este repo (`mcp.go`, `model.go`, `model_orm.go`) — verificado, no lo
> reconstruyas. Esta fase **no** toca la lógica de negocio (`ListItems`/`CreateItem`/`UpdateItem`/…,
> `GetAgreement`/`UpsertAgreement`/…): esas quedan intactas. Esta fase reemplaza **solo** cómo el
> módulo se conecta con el resto del sistema (transporte, id, eventos, UI).

## 1. Qué cambia y por qué

`item_catalog` es el **primer módulo** que se prueba end-to-end contra el patrón "acoplado solo a
contratos": `router` (transporte, vía `Op`), `model` (codec + identidad, vía `IDGenerator`), `view`
(vista), `events` (pub/sub). Hoy:

- `mcp.go` importa `tinywasm/mcp` directamente e implementa `Tools() []mcp.Tool`.
- `New()` construye su propio `unixid.NewUnixID()` internamente — el módulo decide su propio
  generador de IDs, en vez de recibirlo.
- `Deps.Publisher EventPublisher` es una interfaz **local** (`model.go:65`) con
  `Publish(event string, payload any) error` — `payload any` es exactamente el hueco que el arnés
  prohíbe, y esta forma **no coincide** con la de otros módulos del ecosistema (`clinical_encounter`,
  `appointment_booking`), que redeclaran la suya propia con otra firma.
- `Deps.UI UIAdapter` (`model.go:57`) es una abstracción de presentación basada en strings
  (`RenderItemList(items, filter) string`), anterior a `tinywasm/view`.

Los cuatro son exactamente lo que esta fase elimina, reemplazándolos por contratos ya publicados en
`tinywasm/model`, `tinywasm/router`, `tinywasm/events`, `tinywasm/view`.

> **Nota de diseño importante (evita el error de una versión anterior de este plan).** El contrato de
> transporte que este módulo implementa es `router.OpModule { ModelName(); MountOps(reg
> router.OpRegistry) }` — **NO** `router.APIModule { MountAPI(r router.Router) }`. `router.Router` es
> la interfaz HTTP (`Get/Post/Put/Delete/…`); este módulo nunca la ve ni la importa. `OpRegistry` es
> una interfaz de **un solo método** (`Op(name, h) Route`), el espejo de `router.Caller` del lado de
> montaje. Si en algún punto escribes `MountAPI(r router.Router)` o `var _ router.APIModule = …`,
> te equivocaste de contrato — vuelve a `MountOps(reg router.OpRegistry)`.

## 2. Estado actual exacto (verificado, no supuesto)

Todo lo relevante vive en **`mcp.go`** (`//go:build !wasm`) y **`model.go`**.

- `model.go:57-67`:
  ```go
  type UIAdapter interface {
      RenderItemList(items []CatalogItem, activeFilter string) string
      RenderItemForm(item *CatalogItem) string
      RenderFilterSelector(current string) string
  }
  type EventPublisher interface {
      Publish(event string, payload any) error
  }
  ```
- `mcp.go:34-58`:
  ```go
  type Deps struct {
      UI        UIAdapter
      Publisher EventPublisher
  }
  type Module struct {
      db  *orm.DB
      uid *unixid.UnixID
      ui  UIAdapter
      pub EventPublisher
  }
  func New(db *orm.DB, deps Deps) (*Module, error) {
      // CreateTable(&CatalogItem{}), CreateTable(&Agreement{})
      u, err := unixid.NewUnixID() // el módulo CONSTRUYE su propio generador — esto se elimina
      // …
      return &Module{db: db, uid: u, ui: deps.UI, pub: deps.Publisher}, nil
  }
  ```
- `mcp.go:210-228` — `RenderList`/`RenderForm`/`RenderFilter` delegan en `m.ui`; `publish(event,
  payload any)` delega en `m.pub`, con **7 call sites** (`mcp.go:126,141,155,167,206,212,221`):
  `catalog.item.created/updated/deactivated/deleted`, `catalog.agreement.created/updated/deleted`.
  Los payloads hoy son: `item CatalogItem` (create/update), `a *Agreement` (agreement create/update),
  y `map[string]string{"tenant_id":…, "id":…}` (deactivate/delete de ambos) — este último **no**
  compila contra `events.Event.Payload model.Encodable` (`map` no lo implementa) y se reemplaza.
- `mcp.go:261-351` — `func (m *Module) Tools() []mcp.Tool` registra **11** tools (constantes en
  `mcp.go:19-31`): `OpListItems`, `OpGetItem`, `OpFindItemBySKU`, `OpCreateItem`, `OpUpdateItem`,
  `OpUpsertItem`, `OpDeactivateItem`, `OpDeleteItem`, `OpListAgreements`, `OpUpsertAgreement`,
  `OpDeleteAgreement`. Cada uno ya tiene `Args model.Fielder` poblado (p.ej. `&ListItemsArgs{}`,
  `&CatalogItem{}`) — mismo tipo que `router.Route.Accepts(args model.Fielder)`, encaja sin fricción.
- `mcp.go:354-`(fin del archivo) — 11 funciones `mcpXxx(ctx *context.Context, req mcp.Request)
  (*mcp.Result, error)`, cada una: `json.Decode(req.Params.Arguments, &args)` (import
  `tinywasm/json`), llama al método de servicio (`m.ListItems`/`m.CreateItem`/…, sin cambios), y
  `json.Encode(&out, &res)` → `mcp.Text(res)`. Este plan **reescribe la envoltura**, no la lógica de
  negocio que llaman.
- `setup_test.go` (paquete `itemcatalog`, junto al código, no en `tests/`): define `MockUI` (implementa
  `UIAdapter`) y `MockPublisher` (implementa `EventPublisher`, `Publish(event string, payload any)
  error`). `catalog_test.go:19-21,211-212` construye `New(db, Deps{UI: ui, Publisher: pub})`. Ambos
  dobles quedan obsoletos por el cambio de contrato.
- `tests/go.mod` existe como módulo anidado (`module github.com/veltylabs/item_catalog/tests`, con
  `replace … => ../`) pero está **vacío** (cero archivos `.go`) — es un scaffold sin usar. Los tests
  reales de este repo viven en el paquete raíz (`catalog_test.go`, `setup_test.go`, package
  `itemcatalog`, sin build tag). El test de esta fase sigue esa misma convención.
- `go.mod` actual: `github.com/tinywasm/router v0.1.10 // indirect`, `github.com/tinywasm/model
  v0.0.15` (directo), `github.com/tinywasm/mcp v0.1.22` (directo), `github.com/tinywasm/unixid v0.2.23`
  (directo). Se necesitan: `router` sube a **directo** en `v0.1.14`+ (trae `OpRegistry`/`OpModule`/
  `Route.Accepts`/`Context.Decode`+`Encode`/`Caller.Call` tipado); `tinywasm/events@v0.0.2`+ (nuevo,
  directo); `tinywasm/view@v0.1.0`+ (nuevo, directo). `tinywasm/mcp` y `tinywasm/unixid` **se borran**
  de las dependencias directas (§3.2/§3.4) — quedan fuera del módulo por completo, no solo sin usar.

## 3. El cambio exacto

### 3.1 `go.mod`

```
go get github.com/tinywasm/router@v0.1.14
go get github.com/tinywasm/events@latest
go get github.com/tinywasm/view@latest
go mod tidy   # debe DROPEAR tinywasm/mcp y tinywasm/unixid por completo — ver §3.6
```

### 3.2 `Deps`/`Module`/`New` — inyecta lo que hoy se construye solo

Reemplaza en `mcp.go` (cerca del principio):

```go
type Deps struct {
	IDs       model.IDGenerator // requerido — el módulo NUNCA construye un generador
	Publisher events.Publisher  // opcional — nil desactiva la publicación de eventos
}

type Module struct {
	db  *orm.DB
	ids model.IDGenerator
	pub events.Publisher
}

func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err("item_catalog: Deps.IDs is required")
	}
	if err := db.CreateTable(&CatalogItem{}); err != nil {
		return nil, err
	}
	if err := db.CreateTable(&Agreement{}); err != nil {
		return nil, err
	}
	return &Module{db: db, ids: deps.IDs, pub: deps.Publisher}, nil
}
```

- **Borra** `uid *unixid.UnixID` del struct `Module` y el import `github.com/tinywasm/unixid`.
- Todo `m.uid.GetNewID()` (`CreateItem` en `mcp.go:113-125`, `UpsertAgreement`) → `m.ids.NewID()`.
- Añade `"github.com/tinywasm/events"` al import de `mcp.go`.

### 3.3 Eventos tipados — reemplaza `EventPublisher`/`publish`

**Borra por completo** de `model.go`: la interfaz `EventPublisher` (§2). **Borra** de `mcp.go`: `func
(m *Module) publish(event string, payload any)`.

Añade constantes de topic (junto a los `Op*` existentes en `mcp.go:19-31`):

```go
const (
	TopicItemCreated      = "catalog.item.created"
	TopicItemUpdated      = "catalog.item.updated"
	TopicItemDeactivated  = "catalog.item.deactivated"
	TopicItemDeleted      = "catalog.item.deleted"
	TopicAgreementCreated = "catalog.agreement.created"
	TopicAgreementUpdated = "catalog.agreement.updated"
	TopicAgreementDeleted = "catalog.agreement.deleted"
)
```

Cada uno de los 7 call sites de `m.publish(...)` pasa a publicar directo, con un `model.Encodable`
real — **cero `map`** (ya era regla del arnés; aquí además dejaría de compilar:
`events.Event.Payload` exige `model.Encodable`, un `map` no lo satisface):

```go
// mcp.go:126 (dentro de CreateItem) — antes: m.publish("catalog.item.created", item)
if m.pub != nil {
	m.pub.Publish(events.Event{Topic: TopicItemCreated, Payload: &item})
}
```

```go
// mcp.go:155 (dentro de DeactivateItem) — antes: m.publish("catalog.item.deactivated",
// map[string]string{"tenant_id": tenantId, "id": id})
if m.pub != nil {
	m.pub.Publish(events.Event{Topic: TopicItemDeactivated, Payload: &item}) // item ya está en scope (GetItem lo trajo)
}
```

Aplica el mismo patrón a los 5 call sites restantes (`updated`→`&item`, `deleted`→`&item` ya en scope
tras `GetItem`, `agreement.created/updated`→`a` ya es `*Agreement`, `agreement.deleted`→trae el
`*Agreement` con `GetAgreement` antes de borrar, igual que `DeleteItem` ya hace `GetItem` antes de
publicar). **No** inventes un tipo `map`/DTO nuevo solo para el evento de deactivate/delete — el
propio `CatalogItem`/`Agreement` ya tiene `Id`/`TenantId` poblados en ese punto del código.

### 3.4 `UIAdapter`/`RenderList`/`RenderForm`/`RenderFilter` — se borran, los reemplaza `view.go` (§3.7)

**Borra por completo**: la interfaz `UIAdapter` (`model.go:57-61`), el campo `Deps.UI`, el campo `ui
UIAdapter` en `Module`, y los métodos `RenderList`/`RenderForm`/`RenderFilter` (`mcp.go:210-228`). Es
una abstracción de UI basada en strings, previa a `view`, y con `view.New` como único camino de vista
("una forma de hacer cada cosa") ya no tiene función.

### 3.5 `Tools() []mcp.Tool` → `MountOps(reg router.OpRegistry)`

**Borra** `func (m *Module) Tools() []mcp.Tool` completo (`mcp.go:261-351`) y el import
`github.com/tinywasm/mcp`. Reemplázalo por `MountOps`, que registra las **11** ops vía `r.Op(...)`
sobre el `OpRegistry` recibido — **no** un `router.Router`:

```go
func (m *Module) ModelName() string { return "item_catalog" }

func (m *Module) MountOps(reg router.OpRegistry) {
	reg.Op(OpListItems, m.opListItems).Requires("catalog_item", model.Read).Accepts(&ListItemsArgs{})
	reg.Op(OpGetItem, m.opGetItem).Requires("catalog_item", model.Read).Accepts(&GetItemArgs{})
	reg.Op(OpFindItemBySKU, m.opFindItemBySKU).Requires("catalog_item", model.Read).Accepts(&FindBySKUArgs{})
	reg.Op(OpCreateItem, m.opCreateItem).Requires("catalog_item", model.Create).Accepts(&CatalogItem{})
	reg.Op(OpUpdateItem, m.opUpdateItem).Requires("catalog_item", model.Update).Accepts(&CatalogItem{})
	reg.Op(OpUpsertItem, m.opUpsertItem).Requires("catalog_item", model.Create).Accepts(&CatalogItem{})
	reg.Op(OpDeactivateItem, m.opDeactivateItem).Requires("catalog_item", model.Update).Accepts(&DeactivateItemArgs{})
	reg.Op(OpDeleteItem, m.opDeleteItem).Requires("catalog_item", model.Delete).Accepts(&DeleteItemArgs{})
	reg.Op(OpListAgreements, m.opListAgreements).Requires("catalog_agreement", model.Read).Accepts(&ListAgreementsArgs{})
	reg.Op(OpUpsertAgreement, m.opUpsertAgreement).Requires("catalog_agreement", model.Create).Accepts(&Agreement{})
	reg.Op(OpDeleteAgreement, m.opDeleteAgreement).Requires("catalog_agreement", model.Delete).Accepts(&DeleteAgreementArgs{})
}

var _ router.OpModule = (*Module)(nil)
```

Cada `mcpXxx(ctx *context.Context, req mcp.Request) (*mcp.Result, error)` se reescribe como
`opXxx(ctx router.Context)` — firma `router.HandlerFunc`, decode/encode tipados vía `router.Context`,
**cero `tinywasm/json`, cero `tinywasm/mcp`** en el handler. 3 ejemplos completos (list, upsert, delete
— el resto sigue la misma forma, tabla abajo):

```go
func (m *Module) opListItems(ctx router.Context) {
	var args ListItemsArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	filter := ItemFilter{Type: args.Type, ActiveOnly: args.ActiveOnly, Limit: args.Limit, Offset: args.Offset}
	items, err := m.ListItems(args.TenantId, filter)
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	list := make(CatalogItemList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	if err := ctx.Encode(&list); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opUpsertItem(ctx router.Context) {
	var item CatalogItem
	if err := ctx.Decode(&item); err != nil {
		ctx.WriteStatus(400)
		return
	}
	var out CatalogItem
	var err error
	if item.Id == "" {
		out, err = m.CreateItem(item)
	} else {
		out, err = m.UpdateItem(item)
	}
	if err != nil {
		ctx.WriteStatus(500)
		return
	}
	if err := ctx.Encode(&out); err != nil {
		ctx.WriteStatus(500)
	}
}

func (m *Module) opDeleteItem(ctx router.Context) {
	var args DeleteItemArgs
	if err := ctx.Decode(&args); err != nil {
		ctx.WriteStatus(400)
		return
	}
	if err := m.DeleteItem(args.TenantId, args.Id); err != nil {
		ctx.WriteStatus(500)
		return
	}
	ctx.WriteStatus(200)
}
```

**Tabla del resto** (mismo patrón: `Decode` el `*ArgsModel`/registro correspondiente, llama al método
de servicio ya existente sin cambios, `Encode` el resultado o `WriteStatus` según corresponda):

| Op | Reemplaza | Args a decodificar | Servicio que llama |
|---|---|---|---|
| `OpGetItem` | `mcpGetItem` | `GetItemArgs` | `GetItem` |
| `OpFindItemBySKU` | `mcpFindBySKU` | `FindBySKUArgs` | `FindBySKU` |
| `OpCreateItem` | `mcpCreateItem` | `CatalogItem` | `CreateItem` |
| `OpUpdateItem` | `mcpUpdateItem` | `CatalogItem` | `UpdateItem` |
| `OpDeactivateItem` | `mcpDeactivateItem` | `DeactivateItemArgs` | `DeactivateItem` |
| `OpListAgreements` | `mcpListAgreements` | `ListAgreementsArgs` | `ListAgreements` |
| `OpUpsertAgreement` | `mcpUpsertAgreement` | `Agreement` | `UpsertAgreement` |
| `OpDeleteAgreement` | `mcpDeleteAgreement` | `DeleteAgreementArgs` | `DeleteAgreement` |

**Borra** todos los métodos `mcpXxx` viejos una vez migrados — no dejes las dos formas coexistiendo.

### 3.6 `mcp.go` — imports finales

Tras §3.2-§3.5, el bloque de imports de `mcp.go` queda:

```go
import (
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/events"
	"github.com/tinywasm/model"
	"github.com/tinywasm/orm"
	"github.com/tinywasm/router"
	"github.com/tinywasm/time"
)
```

`"github.com/tinywasm/context"`, `"github.com/tinywasm/json"`, `"github.com/tinywasm/mcp"`,
`"github.com/tinywasm/unixid"` **desaparecen** — ninguno vuelve a usarse en este archivo. El `//go:build
!wasm` de la primera línea **se mantiene** (el módulo sigue siendo server-only por su dependencia de
`orm`/`sqlite`, no cambia con esta fase).

### 3.7 `view.go` — nuevo, la vista del ítem

```go
package itemcatalog

import (
	"github.com/tinywasm/model"
	"github.com/tinywasm/router"
	"github.com/tinywasm/view"
)

// NewView builds the catalog item Presenter — the tech-agnostic engine a renderer (crudview,
// or any other) wraps. It is THIS module's job to build it (importing only view+model+router);
// the app decides which renderer draws it.
func NewView(caller router.Caller) view.Presenter {
	byID := map[string]*CatalogItem{} // estado privado — única excepción "cero map" (firma pública, no esto)
	record := &CatalogItem{}

	return view.New(
		caller,
		record,
		OpListItems,
		func() model.FielderSlice { return &CatalogItemList{} },
		func(list model.FielderSlice) []view.Item {
			l := list.(*CatalogItemList)
			items := make([]view.Item, l.Len())
			for i := 0; i < l.Len(); i++ {
				it := l.At(i).(*CatalogItem)
				byID[it.Id] = it
				items[i] = view.Item{ID: it.Id, Label: it.Name, Description: it.Sku}
			}
			return items
		},
		view.WithTitle("Catálogo"),
		view.WithSaveOp(OpUpsertItem),
		view.WithDeleteOp(OpDeleteItem),
		view.WithFill(func(id string) model.Model {
			if id == "" {
				return nil
			}
			return byID[id]
		}),
	)
}
```

> **Convenios sin vista propia todavía.** `NewView` cubre solo el catálogo de ítems (paridad con lo
> que ya existía). Una vista de convenios (`NewAgreementsView`, o una vista anidada) queda
> **explícitamente fuera de esta fase**: "convenios sin UI" no es deuda, es una decisión — el CRUD de
> convenios ya es alcanzable por `Op` (§3.5) aunque no tenga renderer.

### 3.8 Tests existentes — `setup_test.go`/`catalog_test.go`

`setup_test.go`: **borra** `MockUI` (implementaba `UIAdapter`, que ya no existe) y `MockPublisher`
(firma vieja `Publish(event string, payload any) error`). Reemplázalo por un doble que satisfaga
`events.Publisher`:

```go
type MockPublisher struct {
	Events []events.Event
}

func (m *MockPublisher) Publish(e events.Event) {
	m.Events = append(m.Events, e)
}

var _ events.Publisher = (*MockPublisher)(nil)
```

`catalog_test.go:19-21,211-212`: `New(db, Deps{UI: ui, Publisher: pub})` → `New(db, Deps{IDs:
<generador falso o events/mock>, Publisher: pub})`. Un generador falso mínimo, junto al `MockPublisher`
en `setup_test.go`:

```go
type sequentialIDs struct{ n int }

func (s *sequentialIDs) NewID() string {
	s.n++
	return fmt.Sprintf("test-id-%d", s.n)
}

var _ model.IDGenerator = (*sequentialIDs)(nil)
```

Cualquier aserción en `catalog_test.go` sobre `ui.RenderItemListCalled`/`RenderItemFormCalled`
(`MockUI`) se borra junto con `MockUI` — no queda código muerto referenciándolo.

## 4. Fuera de alcance

- **No** toques `model.go`/`model_orm.go` más allá de borrar `UIAdapter`/`EventPublisher` (§3.3/§3.4)
  — el schema `Kind` y `Agreement` ya están correctos.
- **No** toques `ListItems`/`GetItem`/`FindBySKU`/`CreateItem`/`UpdateItem`/`DeactivateItem`/
  `DeleteItem`/`GetAgreement`/`UpsertAgreement`/`ListAgreements`/`DeleteAgreement` — la lógica de
  negocio no cambia, solo cómo se invoca desde el borde (`opXxx` en vez de `mcpXxx`).
- **No** le des a `MountOps` una forma distinta de `Op(...).Requires(...).Accepts(...)` — es el único
  contrato de transporte del arnés.
- **No** hagas que el módulo reciba `router.Router` ni implemente `router.APIModule`/`MountAPI` — ve
  solo `router.OpRegistry`/`router.OpModule` (§1, nota de diseño).
- **No** construyas un broker de eventos dentro del módulo — `events.Publisher` se inyecta, `nil`
  desactiva la publicación silenciosamente (no es un error).
- **No** añadas una vista de convenios (§3.7, nota).
- **No** pobles el módulo anidado `tests/` (vacío hoy) — los tests de esta fase siguen la convención
  existente: package `itemcatalog`, junto al código.

## 5. Test con forma de consumidor (obligatorio, arnés de construcción)

Añade a `catalog_test.go` (o un archivo nuevo `harness_test.go`, mismo paquete) un test que ejerza
`MountOps` contra `router/mock` — prueba que un consumidor real (un composition root) puede montar
este módulo sin conocer su interior:

```go
func TestModule_MountOpsAndView(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	pub := &MockPublisher{}
	module, err := New(db, Deps{IDs: &sequentialIDs{}, Publisher: pub})
	if err != nil {
		t.Fatal(err)
	}

	r := mock.NewRouter() // github.com/tinywasm/router/mock — ajusta al constructor real del paquete
	module.MountOps(r)

	infos := r.Routes()
	var found bool
	for _, i := range infos {
		if i.Path == OpUpsertItem { // Op se registra por NOMBRE; RouteInfo.Path lleva ese nombre
			found = true
			if i.Resource != "catalog_item" || i.Action != model.Create {
				t.Errorf("RBAC mismatch for %s: %+v", OpUpsertItem, i)
			}
		}
	}
	if !found {
		t.Fatalf("MountOps did not register %s", OpUpsertItem)
	}

	caller := mock.NewCaller() // doble de router.Caller — ajusta al constructor real del paquete
	pres := module.NewView(caller)
	if pres.Title() == "" {
		t.Error("expected a non-empty view title")
	}
}
```

> Ajusta los nombres de constructor exactos de `router/mock` (`mock.NewRouter`/`mock.NewCaller` son
> ilustrativos) al API real publicada en `github.com/tinywasm/router/mock` — lee su `README.md` si
> difiere.

## 6. Criterios de aceptación

- `grep -rn "tinywasm/mcp\|tinywasm/json\|tinywasm/unixid" .` (código no-test) **vacío**.
- `grep -rn "router.Router\b\|router.APIModule\|MountAPI" .` (código no-test) **vacío** — el módulo
  solo ve `router.OpRegistry`/`router.OpModule`.
- `*Module` implementa `router.OpModule` (`ModelName`+`MountOps`); **no** existe `Tools() []mcp.Tool`.
- `UIAdapter`, `EventPublisher`, `Deps.UI`, `RenderList`/`RenderForm`/`RenderFilter` no existen.
- `Deps{ IDs model.IDGenerator; Publisher events.Publisher }` — `New` falla si `IDs == nil`.
- `view.go` expone `NewView(caller router.Caller) view.Presenter`.
- `go.mod`: `tinywasm/router@v0.1.14`+ directo, `tinywasm/events@v0.0.2`+ directo, `tinywasm/view@v0.1.0`+
  directo; `tinywasm/mcp`, `tinywasm/unixid` **ausentes** por completo (`go mod tidy` los dropea).
- El test de §5 verde: `MountOps` registrado contra `router/mock` con RBAC correcto, `NewView(...)`
  produce un `Presenter` usable.
- `gotest ./...` (o `go test ./...`) verde.

## 7. Etapas

| # | Etapa | Archivo(s) | Criterio |
|---|---|---|---|
| 1 | Bump deps | `go.mod`, `go.sum` | `router@v0.1.14`+, `events@v0.0.2`+, `view@v0.1.0`+; `mcp`/`unixid` fuera |
| 2 | `Deps`/`Module`/`New` | `mcp.go` | `IDGenerator`+`events.Publisher` inyectados, sin `unixid` interno |
| 3 | Eventos tipados | `mcp.go`, `model.go` | borra `EventPublisher`/`publish`; 7 call sites a `events.Event` |
| 4 | Borrar `UIAdapter` | `model.go`, `mcp.go` | §3.4 |
| 5 | `MountOps` | `mcp.go` | reemplaza `Tools()`; 11 ops vía `Op` sobre `OpRegistry` |
| 6 | `view.go` | `view.go` (nuevo) | `NewView` |
| 7 | Migrar tests existentes | `setup_test.go`, `catalog_test.go` | §3.8, sin `MockUI` |
| 8 | Test consumidor | `catalog_test.go` o `harness_test.go` | §5, verde |
| 9 | Verificación | — | `grep` de §6 vacío; `gotest ./...` verde |
