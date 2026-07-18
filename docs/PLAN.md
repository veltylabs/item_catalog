---
PLAN: "fix: adopt tinywasm/ddl for schema migration, drop sqlite from tests"
TAG: v0.3.0
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 10767037318860624234
PR: https://github.com/veltylabs/item_catalog/pull/5
---

> This plan is dispatched via the CodeJob workflow. See skill: **agents-workflow**.

# PLAN — item_catalog: cerrar el drift de API (`ddl`) y sacar `sqlite` de los tests

Eres un agente **sin contexto previo** y **solo tienes este repositorio** (`item_catalog`). Plan
autocontenido: todo contrato, regla y ejemplo está inline. Lee también `AGENTS.md` (raíz de este
repo) antes de tocar nada — declara la whitelist/blacklist de imports que este plan aplica.

---

## 1. Qué está roto y por qué

`item_catalog` es la implementación de referencia del patrón de 5 contratos
(`model`+`router`+`view`+`events`+`orm`) para `veltylabs/modules` — ver
[`REUSABLE_MODULES_MASTER_PLAN.md`](https://github.com/tinywasm/app/blob/main/docs/REUSABLE_MODULES_MASTER_PLAN.md).
Ese trabajo ya está mergeado a `main` (commit `c6f90d0`). Desde entonces, una dependencia aguas
arriba cambió de API y este repo quedó roto:

```
./mcp.go:55:15: db.CreateTable undefined (type *orm.DB has no field or method CreateTable)
./mcp.go:58:15: db.CreateTable undefined (type *orm.DB has no field or method CreateTable)
```

`orm.DB.CreateTable` fue removido — la migración de schema se extrajo a `github.com/tinywasm/ddl`
(sibling de `orm`, ninguno depende del otro). Además hay tres cosas más por cerrar en el mismo
repo, detectadas en la misma revisión (§3, §4, §5).

## 1b. Segundo bug real, verificado: `mcp.go` no compila para `wasm`

`mcp.go` tiene `//go:build !wasm`, pero declara las constantes `OpListItems`/`OpUpsertItem`/
`OpDeleteItem`/etc. que `view.go` (sin build tag) referencia. Verificado con:

```
GOOS=js GOARCH=wasm go build ./...
# ./view.go:19:3: undefined: OpListItems
# ./view.go:32:19: undefined: OpUpsertItem
# ./view.go:33:21: undefined: OpDeleteItem
```

El tag es un resto de cuando `mcp.go` importaba `tinywasm/mcp`/`tinywasm/unixid` directamente (no
isomórficos). Hoy `mcp.go` solo importa `fmt`/`model`/`orm`/`router`/`time`/`events` — todos
isomórficos según la regla de build tags de `AGENTS.md`. **Elimina la línea `//go:build !wasm` de
`mcp.go`** (primera línea del archivo) como parte de este mismo plan — no hace falta ningún otro
cambio para que esta parte compile en `wasm` (el resto de errores de `wasm` son los mismos de `!wasm`,
cubiertos por el fix de §2).

## 2. Fix — `New()` adopta `tinywasm/ddl`

`ddl.New` toma **dos** argumentos: `ddl.New(conn storage.Conn, ddlCompiler ddl.Compiler) *DB`.
`ddl.Compiler` es una **capacidad opcional** — solo la implementan backends SQL (`sqlt`, `postgres`);
el backend en memoria `storage/mem` (el que este repo debe usar en tests, ver §4) **no** la
implementa — crea tablas de forma perezosa en el primer `Exec` y no necesita DDL. El módulo hace un
*type assertion* sobre `db.RawConn()` en vez de asumir la capacidad — el mismo idioma que
`storage.TxExecutor` ya usa para transacciones opcionales en el resto del ecosistema:

```go
if ddlCompiler, ok := db.RawConn().(ddl.Compiler); ok {
    if err := ddl.New(db.RawConn(), ddlCompiler).CreateTable(&CatalogItem{}); err != nil {
        return nil, err
    }
    if err := ddl.New(db.RawConn(), ddlCompiler).CreateTable(&Agreement{}); err != nil {
        return nil, err
    }
}
```

Contra `storage/mem` esto es un no-op (nada que crear). Contra un backend SQL real, migra el schema
exactamente como hacía el `orm.DB.CreateTable` removido.

**Edita `mcp.go`:**

1. Reemplaza el import `"github.com/tinywasm/orm"` (sigue usándose para las queries — no lo quites)
   agregando `"github.com/tinywasm/ddl"` al bloque de imports.
2. En `New(db *orm.DB, deps Deps) (*Module, error)`, sustituye las dos líneas actuales:
   ```go
   if err := db.CreateTable(&CatalogItem{}); err != nil {
       return nil, err
   }
   if err := db.CreateTable(&Agreement{}); err != nil {
       return nil, err
   }
   ```
   por el bloque de §2 arriba (una sola comprobación `ok`, dos llamadas a `CreateTable` dentro).
3. Añade `github.com/tinywasm/ddl` como dependencia **directa** en `go.mod` (hoy es `// indirect` —
   ver `go.sum`, ya está resuelta en el grafo de dependencias vía `ddlc`; solo falta declararla
   directa) en la versión que ya trae el `go.mod` actual (`v0.0.4` o la que `go mod tidy` fije).

## 3. Confirmar y comitear los cambios locales ya en curso

Este repo tiene cambios sin commitear que **no son parte de este plan** pero deben commitearse junto
con el fix de §2 para que el árbol quede limpio — son bumps de dependencias y un refactor de
organización ya correctos, solo pendientes de confirmar:

- `go.mod`/`go.sum`: bump a `model v0.0.16`, `orm v0.11.1`, `router v0.1.15`, `view v0.1.1`
  (y transitivos `ddl v0.0.4`, `storage v0.0.2`, `ddlc v0.0.6`, `dom v0.11.2`, `css v0.1.4`). **No
  revertir estos bumps** — son la razón por la que `db.CreateTable` desapareció (§1) y son
  prerequisito de este fix.
- `model.go`: la interfaz `CatalogService` fue movida fuera de este archivo. Verifica que
  `interfaces.go` (archivo nuevo, sin tracking de git — `git status` lo muestra como *untracked*) la
  contiene íntegra:
  ```go
  package itemcatalog

  // CatalogService — the core business interface. Implemented by *Module.
  type CatalogService interface {
      GetItem(tenantId, id string) (CatalogItem, error)
      FindBySKU(tenantId, sku string) (CatalogItem, error)
      ListItems(tenantId string, filter ItemFilter) ([]CatalogItem, error)
      CreateItem(item CatalogItem) (CatalogItem, error)
      UpdateItem(item CatalogItem) (CatalogItem, error)
      DeactivateItem(tenantId, id string) error
      DeleteItem(tenantId, id string) error
      ServiceExists(tenantId, serviceId string) (bool, error)

      ListAgreements(tenantId, catalogItemId string) ([]Agreement, error)
      UpsertAgreement(Agreement) (Agreement, error)
      DeleteAgreement(tenantId, id string) error
  }
  ```
  Si falta algo, complétalo — no reescribas la interfaz, solo verifica que el contenido movido es
  exactamente este.

## 4. Sacar `tinywasm/sqlite` de los tests — usar `storage/mem`

Por regla del ecosistema (`AGENTS.md`, sección "blacklist"): **ningún backend de storage concreto,
ni siquiera en tests.** Hoy `catalog_test.go` (paquete `itemcatalog`, en la raíz del repo, no en
`tests/`) abre `tinywasm/sqlite`:

```go
// catalog_test.go — ACTUAL, a reemplazar
import (
    "testing"

    "github.com/tinywasm/model"
    "github.com/tinywasm/router/mock"
    "github.com/tinywasm/sqlite"
)

func TestCatalog(t *testing.T) {
    db, err := sqlite.Open(":memory:")
    if err != nil {
        t.Fatal(err)
    }
    defer sqlite.Close(db)
    // ...
}
```

**Reescribe la construcción de `db`** usando `storage/mem` + `orm.New`:

```go
package tests // el archivo se mueve a tests/catalog_test.go — ver más abajo

import (
    "testing"

    itemcatalog "github.com/veltylabs/item_catalog"
    "github.com/tinywasm/orm"
    "github.com/tinywasm/router/mock"
    "github.com/tinywasm/storage/mem"
)

func TestCatalog(t *testing.T) {
    db := orm.New(mem.New())

    pub := &itemcatalog.MockPublisher{}
    module, err := itemcatalog.New(db, itemcatalog.Deps{
        IDs:       &mockIDGen{},
        Publisher: pub,
    })
    if err != nil {
        t.Fatal(err)
    }
    // ... resto del test sin cambios de lógica, solo el paquete externo
    // (usos internos como module.CreateItem(...) siguen iguales; los que
    // referenciaban símbolos no exportados del paquete itemcatalog deben
    // pasar a la API pública o quedarse en un archivo interno — ver Etapa 3).
}
```

**`tests/` ya es un submódulo Go propio** (no un simple paquete): `tests/go.mod` existe hoy, vacío
salvo identidad + `replace`:

```
module github.com/veltylabs/item_catalog/tests

go 1.25.2

replace github.com/veltylabs/item_catalog => ../
```

Esto es intencional (mismo patrón que replican los otros módulos del batch, ver
`veltylabs/modules/agent_switch/docs/PLAN.md` §8 si quieres un segundo ejemplo): aísla las
dependencias *solo de test* (`storage/mem`, `router/mock`) del `go.mod` raíz del módulo, que así
queda limpio de todo lo que un consumidor de producción no necesita. **Los `require` de `orm`,
`storage/mem`, `router/mock` van en `tests/go.mod`, no en el `go.mod` raíz.**

**Pasos concretos:**

1. Mueve `catalog_test.go` a `tests/catalog_test.go`, cambia `package itemcatalog` → `package tests`,
   y antepone `itemcatalog.` a todo símbolo exportado del paquete (`CatalogItem`, `New`, `Deps`, …).
2. `setup_test.go` define `MockPublisher` y `mockIDGen` — son tipos de ayuda para tests, no símbolos
   de producción. Muévelos también a `tests/setup_test.go` con `package tests`, exportando
   `MockPublisher`/`MockIDGen` (con `M` mayúscula si no lo estaban) para que `tests/catalog_test.go`
   pueda usarlos como paquete externo.
3. En el `setup_test.go` movido, reemplaza el import `"strconv"` (stdlib, prohibido por `AGENTS.md`
   — este ecosistema usa `github.com/tinywasm/fmt` en su lugar) en `mockIDGen.NewID`:
   ```go
   // ANTES
   import "strconv"
   func (m *mockIDGen) NewID() string {
       m.counter++
       return "test-id-" + strconv.Itoa(m.counter)
   }

   // DESPUÉS
   import "github.com/tinywasm/fmt"
   func (m *MockIDGen) NewID() string {
       m.counter++
       return "test-id-" + fmt.Convert(m.counter).String()
   }
   ```
4. Corre `go mod tidy` **dentro de `tests/`** (no en la raíz) tras mover los archivos — resuelve
   `github.com/tinywasm/orm`, `github.com/tinywasm/storage`, `github.com/tinywasm/router/mock` como
   requires directos de `tests/go.mod`, vía el `replace` local que ya apunta a la raíz del repo.
5. Borra `github.com/tinywasm/sqlite` del `go.mod` **raíz** (`go mod tidy` ahí también) — no debe
   quedar ninguna referencia en todo el repo, ni siquiera indirecta.

**6. Elimina la línea `//go:build !wasm` de `mcp.go`** (§1b) — sin este paso `GOOS=js GOARCH=wasm go
build ./...` sigue roto aunque §2/§3/§4 estén hechos.

## 5. Fuera de alcance

- No tocar `view.go`, `interfaces.go` (más allá de verificar §3), ni la lógica de negocio de
  `mcp.go` fuera de las líneas de `CreateTable` (§2).
- No renombrar `mcp.go` — ese archivo ya no importa `tinywasm/mcp` (el nombre es historia, no una
  violación de la whitelist); renombrarlo es puro churn, fuera de este plan.
- No añadir un test nuevo de `MountOps`/`view.Presenter` contra `router/mock` — ya existe cobertura
  para eso (verifica en `tests/` tras el movimiento); si falta, es un plan separado.

## 6. Criterio de aceptación

- `grep -rn "tinywasm/sqlite\|tinywasm/mcp\|tinywasm/json\|tinywasm/unixid" .` (repo completo, tests
  incluidos) → vacío.
- `grep -rn "\"strconv\"" .` → vacío.
- `go build ./...` y `GOOS=js GOARCH=wasm go build ./...` limpios.
- `gotest ./...` verde.
- `go.mod`: `tinywasm/ddl` es dependencia directa; `tinywasm/sqlite` no aparece en absoluto (ni
  directa ni indirecta).
- `git status` limpio tras el commit — sin cambios pendientes de `go.mod`/`go.sum`/`model.go` ni
  `interfaces.go` sin trackear.

## 7. Etapas

| # | Etapa | Salida | Criterio |
|---|---|---|---|
| 1 | `mcp.go`: adoptar `ddl.CreateTable` con type assertion (§2) | `New()` migra vía `ddl` cuando el backend lo soporta | compila, `db.CreateTable` ya no aparece |
| 2 | Quitar `//go:build !wasm` de `mcp.go` (§1b) | archivo sin build tag | `GOOS=js GOARCH=wasm go build ./...` limpio |
| 3 | Confirmar `interfaces.go`/`model.go` (§3) | árbol de git limpio | `git status` sin pendientes |
| 4 | Mover tests a `tests/` (submódulo propio), cambiar a `storage/mem`, quitar `strconv` (§4) | `tests/catalog_test.go`, `tests/setup_test.go`, `tests/go.mod` con requires resueltos | `gotest ./...` verde, sin `sqlite`/`strconv` |
| 5 | `go mod tidy` en raíz y en `tests/`, verificar deps (§6) | ambos `go.mod` limpios | `tinywasm/ddl` directa en la raíz; `tinywasm/sqlite` ausente en todo el repo |
