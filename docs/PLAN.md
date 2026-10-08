---
PLAN: "fix: detect sentinel errors without == between interfaces (no reflection in wasm)"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 11178711259656560016
PR: https://github.com/veltylabs/item_catalog/pull/9
---

# Plan — `item_catalog`: errores centinela sin `==` entre interfaces

> Master: `webtyp/docs/NO_REFLECTION_MASTER_PLAN.md` (ola 3). Doctrina: skill `api-design`.
> **Prerrequisito:** `go get webtyp.com/orm@latest` y confirmar que existe `orm.IsNotFound`. Si falta alguna, parar y reportarlo: no implementar un sustituto local.

## 1. El problema

En TinyGo, `==`, `!=` y `switch` entre valores de interfaz compilan a `runtime.interfaceEqual`, que
llama a `reflectValueEqual(reflectlite.ValueOf(x), reflectlite.ValueOf(y))`. `error` es una interfaz:
cada `err == ErrX` mete `internal/reflectlite` (~9 KB) en el binario wasm. La regla del dueño es que
el código que compila a wasm no use reflexión nunca. `errors.Is`/`errors.As` tampoco sirven: también
usan reflectlite.

## 2. La corrección — dos patrones, ninguno más

**A. Centinelas de otros paquetes** — usar su función de consulta:

| Antes | Después |
|---|---|
| `err == orm.ErrNotFound` | `orm.IsNotFound(err)` |
| `err != orm.ErrNotFound` | `!orm.IsNotFound(err)` |
| `err == storage.ErrNoRows` | `storage.IsNoRows(err)` |

**B. Centinelas propios de este paquete** — un tipo string no exportado; se afirma una vez y se
compara el valor concreto (comparación de strings, sin reflexión):

```go
// domainError is the concrete type of this package's sentinel errors. Code
// compares them by asserting this type and comparing the value: == between two
// error values compiles, under TinyGo, to runtime.interfaceEqual, which pulls
// internal/reflectlite into the wasm binary.
type domainError string

func (e domainError) Error() string { return string(e) }

const (
	ErrNotFound domainError = "<texto actual>"
	// … uno por centinela, con su texto actual
)
```

- `<texto actual>`: el string exacto que devuelve hoy el centinela (`fmt.Err("a", "b")` une las
  palabras con un espacio: `"a b"`). Un test fija cada texto: los mensajes no cambian.
- Uso, por ejemplo al traducir errores a códigos:

```go
if e, ok := err.(domainError); ok {
	switch e {
	case ErrFloorInUse, ErrRoomOverlap:
		return conflict
	case ErrNotFound:
		return notFound
	}
}
if orm.IsNotFound(err) {
	return notFound
}
```

- Un `switch err { case ErrA: … }` pasa a `if e, ok := err.(domainError); ok { switch e { … } }`.
- Si un centinela propio se envuelve antes de compararlo (`fmt.Errf("…%v", ErrX)`), la comparación
  con `==` ya no funcionaba: dejarlo igual y anotarlo en el PR, no inventar otra detección.

## 3. Sitios a cambiar (inventario del 2026-10-08)

### Código de producción

- `migration.go:57` — `} else if err == ErrSpecialtyNotFound {`
- `mcp.go:85` — `if err == orm.ErrNotFound {`
- `mcp.go:98` — `if err == orm.ErrNotFound {`
- `mcp.go:111` — `if err == orm.ErrNotFound {`
- `mcp.go:153` — `} else if err != ErrSpecialtyNotFound {`
- `mcp.go:163` — `} else if err != ErrSpecialtyNotFound {`
- `mcp.go:225` — `if err == orm.ErrNotFound {`
- `mcp.go:238` — `if err == orm.ErrNotFound {`
- `mcp.go:292` — `} else if err != ErrNotFound {`
- `mcp.go:362` — `if err == ErrNotFound {`
- `mcp.go:392` — `if err == orm.ErrNotFound {`
- `mcp.go:512` — `if err == ErrSpecialtyNotFound || err == ErrNotFound {`
- `mcp.go:534` — `} else if err == ErrSpecialtyPrefixExists || err == ErrSpecialtySlugExists {`
- `mcp.go:536` — `} else if err == ErrSpecialtyNotFound || err == ErrNotFound {`
- `mcp.go:555` — `if err == ErrSpecialtyNotFound || err == ErrNotFound {`
- `mcp.go:557` — `} else if err == ErrSpecialtyInUse {`
- `mcp.go:596` — `if err == ErrNotFound {`
- `mcp.go:616` — `if err == ErrNotFound {`
- `mcp.go:638` — `} else if err == ErrAlreadyExists {`
- `mcp.go:660` — `} else if err == ErrNotFound {`
- `mcp.go:688` — `} else if err == ErrAlreadyExists {`
- `mcp.go:690` — `} else if err == ErrNotFound {`
- `mcp.go:709` — `if err == ErrNotFound {`
- `mcp.go:726` — `if err == ErrNotFound {`
- `mcp.go:766` — `} else if err == ErrNotFound {`
- `mcp.go:785` — `if err == ErrNotFound {`

### Centinelas propios de este repo (patrón B)

- `mcp.go:12` — `var ErrNotFound = fmt.Err("item not found")`
- `mcp.go:13` — `var ErrAlreadyExists = fmt.Err("item already exists")`
- `mcp.go:15` — `var ErrSpecialtyNotFound = fmt.Err("specialty not found")`
- `mcp.go:16` — `var ErrSpecialtyInUse = fmt.Err("specialty in use")`
- `mcp.go:17` — `var ErrSpecialtyPrefixExists = fmt.Err("specialty prefix already exists")`
- `mcp.go:18` — `var ErrSpecialtySlugExists = fmt.Err("specialty slug already exists")`

### Tests (se migran igual: un solo camino también en los tests)

- `tests/catalog_test.go:169` — `if err != itemcatalog.ErrSpecialtyInUse {`
- `tests/catalog_test.go:239` — `if err != itemcatalog.ErrSpecialtyPrefixExists {`
- `tests/catalog_test.go:250` — `if err != itemcatalog.ErrSpecialtySlugExists {`
- `tests/tenant_test.go:36` — `if err != itemcatalog.ErrSpecialtyNotFound {`
- `tests/tenant_test.go:59` — `if err != itemcatalog.ErrNotFound {`
- `tests/tenant_test.go:67` — `if err != itemcatalog.ErrNotFound {`
- `tests/tenant_test.go:76` — `if err != itemcatalog.ErrNotFound {`
- `tests/tenant_test.go:82` — `if err != itemcatalog.ErrNotFound {`
- `tests/tenant_test.go:105` — `if err != itemcatalog.ErrNotFound {`
- `tests/tenant_test.go:114` — `if err != itemcatalog.ErrNotFound {`

Si encuentras otro `==`/`!=`/`switch` entre valores de interfaz con operandos no nil que no esté en la
lista, se migra igual. `x == nil` y `x != nil` están bien.

## 4. Tests

- Todos los tests existentes siguen verdes sin cambiar su intención.
- Un test que fija el `Error()` de cada centinela propio convertido (patrón B) contra su texto anterior.
- Si el paquete traduce errores a códigos/respuestas (por ejemplo en `ops.go`), un test por rama
  cambiada: el mismo error produce el mismo código que antes.
- `gotest` verde (vet, race, tests, wasm).

## 5. Criterios de aceptación

- `grep -rnE '(==|!=) *[A-Za-z_.]*Err[A-Za-z]*' --include=*.go . | grep -v '_temp/'` → vacío.
- `grep -rn 'switch err {' --include=*.go .` → vacío.
- `grep -rn 'errors.Is\|errors.As' --include=*.go .` → vacío.
- Ningún símbolo exportado nuevo: `git diff | grep '^+func [A-Z]'`.
- `gotest` verde.

## 6. Restricciones

Las de `AGENTS.md`, más: nada de `reflect`, `unsafe`, `errors.Is`/`errors.As`, ni `==`/`!=`/`switch`
entre valores de interfaz con operandos no nil. No tocar otros repos.

## Executor notes
- Added `domainError` type and converted all internal sentinel errors to use it.
- Updated occurrences of `err == orm.ErrNotFound` to `orm.IsNotFound(err)`.
- Updated occurrences of internal sentinel errors using type assertions `e, ok := err.(domainError); ok && e == ErrSpecialtyNotFound` to avoid `reflectlite` in tinygo wasm.
- Updated test error comparisons with strings.
- Added test in `tests/error_test.go` to ensure sentinel error strings match exactly.
