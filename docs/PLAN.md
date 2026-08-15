---
PLAN: "feat!: especialidad como entidad — category deja de ser texto libre"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 2953710230884396415
---

> Este plan se despacha con el flujo CodeJob. Ver skill: agents-workflow.

# Plan — `category` no puede seguir siendo un string suelto

## El problema

`CatalogItemModel` declara hoy:

```go
{Name: "category", Type: input.Text(), OmitEmpty: true,
 Permitted: model.Permitted{Letters: true, Spaces: true, Minimum: 1, Maximum: 100}},
```

Texto libre. Consecuencias reales, no hipotéticas:

- **No hay lista canónica.** "Oftalmología" y "Oftalmologia" son dos categorías
  distintas para la base de datos. Un typo crea una especialidad fantasma que
  nadie ve hasta que un listado sale partido en dos.
- **No hay dónde colgar nada.** Una especialidad necesita orden de
  presentación, descripción, ícono y —ver abajo— saber si es pública. Con un
  string no hay sitio para ninguna de esas cosas salvo inventar tablas
  paralelas.
- **La convención del SKU no está representada.** El catálogo documenta
  (`docs/CATALOGO_SERVICIOS.md`) que el SKU son 6 caracteres:
  `[categoría 2 letras][servicio 4 letras]` — `mdcong` = **md** + cong. Ese
  prefijo ES la categoría, pero vive solo en una convención de documento y en
  la disciplina de quien carga los datos.

## Por qué ahora

El sitio público de la clínica pasa a ser una **proyección de este catálogo**:
una página por especialidad, listando sus ítems. Sin entidad no hay slug para
la URL, ni descripción para el `<meta name="description">`, ni forma de decir
"esta especialidad todavía no se publica".

Y hay una regla que no se negocia: **una sola forma de editar**. Si la
especialidad se re-autora en otro lado para el sitio, existen dos fuentes de
verdad y divergen. Se edita aquí, donde ya se edita el catálogo.

## El arreglo

### 1. `SpecialtyModel`

Entidad nueva en este módulo — es la taxonomía del catálogo, no un concepto
prestado:

```go
var SpecialtyModel = model.Definition{
	Name: "specialty",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		// prefix: las 2 letras del SKU. Unique — es lo que hace que la
		// convención documentada pase a ser una invariante de la base.
		{Name: "prefix", Type: input.Text(), NotNull: true,
			Permitted: model.Permitted{Letters: true, Minimum: 2, Maximum: 2}},
		// slug: el segmento de URL del sitio público ("oftalmologia").
		{Name: "slug", Type: input.Text(), NotNull: true,
			Permitted: model.Permitted{Letters: true, Numbers: true, Extra: []rune{'-'}, Minimum: 1, Maximum: 60}},
		{Name: "name", Type: input.Text(), NotNull: true,
			Permitted: model.Permitted{Minimum: 1, Maximum: 100}},
		{Name: "description", Type: input.Textarea(), OmitEmpty: true},
		{Name: "position", Type: BaseInt_FieldInt, OmitEmpty: true},
		{Name: "is_published", Type: Checkbox_FieldBool, NotNull: true},
		{Name: "updated_at", Type: BaseInt_FieldInt, OmitEmpty: true},
	},
}
```

**`is_published` arranca en `false` y eso es deliberado** (principio 8 del
`CONSTRUCTION_HARNESS.md`: cerrado por defecto). Una especialidad nueva no
aparece sola en un sitio público: publicarla cuesta una línea explícita y
grepeable. Lo contrario —aparecer porque nadie dijo que no— es un fallo
silencioso.

`prefix` y `slug` son **únicos por tenant**. Mira cómo `CatalogItem.sku`
resuelve hoy la unicidad por tenant (`FindBySKU` + la comprobación en
`CreateItem`) y sigue ese mismo camino; no inventes un segundo mecanismo.

### 2. `CatalogItem.category` → `CatalogItem.specialty_id`

Referencia real, con el patrón que `AgreementModel.catalog_item_id` ya usa en
este mismo fichero:

```go
{Name: "specialty_id", Type: model.Text(), Ref: &SpecialtyModel,
 DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
```

El campo `category` **se elimina**. Cambio con ruptura, sin alias ni columna
puente: dejar las dos es garantizar que diverjan.

### 3. Operaciones y filtro

- CRUD de especialidad siguiendo exactamente el patrón de los ops existentes
  (`OpListItems`/`OpGetItem`/`OpUpsertItem`/`OpDeleteItem` + `MountOps` con
  `.Requires("specialty", model.Read|Create|…)`).
- `ItemFilter` gana `specialty_id`, para que "dame los ítems de esta
  especialidad" sea una consulta, no un filtrado en memoria del consumidor.
  Es la consulta que el sitio público hará por cada página.
- Borrar una especialidad **con ítems asociados debe fallar** con un error
  propio (`ErrSpecialtyInUse` o similar), no dejar ítems huérfanos apuntando a
  una `specialty_id` inexistente.

### 4. Migración de los datos existentes

**Deriva del prefijo del SKU, no del `category` actual.** El campo de texto
libre ya tiene ruido (revisando el seed aparecen valores que son nombres de
servicio, no categorías: "2 proyecciones", "Antebrazo", "Calcáneo"), mientras
que el SKU sí respeta la convención documentada.

Pasos:

1. Sembrar las especialidades canónicas desde la tabla de prefijos de
   `docs/CATALOGO_SERVICIOS.md` (`md`, `do`, `tr`, `po`, `la`, `ec`, `gi`,
   `ca`, `ga`, `of`, `ne`, `ps`, `de`, `ra`).
2. Asignar `specialty_id` a cada ítem por `sku[:2]`.
3. **Los ítems cuyo prefijo no esté en la lista canónica no se adivinan**: se
   reportan para revisión humana. Un ítem mal clasificado en un catálogo
   clínico es peor que un ítem sin clasificar.

Entrega esto como una función de migración testeable con datos de ejemplo, no
como un `.sql` suelto — tiene que poder correr contra el fake de la suite.

## Restricciones

- Sin carpetas `internal/`.
- Todo literal repetido es una constante exportada y grepeable — la regla que
  este repo ya aplica a `ItemTypeService`/`ItemTypeProduct`.
- Cambio con ruptura permitido y esperado: no dejes `category` ni un alias.
- No toques el montaje en `veltylabs/mjosefa-cms` — ese repo tiene su propio
  plan y se actualiza después de que éste se publique.
- Los helpers `Checkbox_FieldBool` / `BaseInt_FieldInt` existen porque el
  parser AST de `ormc` los necesita; úsalos, no los reemplaces por llamadas
  inline.

## Verificación

- `ormc` regenera `model_orm.go` sin errores y `Specialty`/`SpecialtyList`
  existen con sus codecs.
- Suite del módulo verde, incluyendo casos nuevos:
  - crear dos especialidades con el mismo `prefix` en un tenant → rechazado;
    el mismo `prefix` en tenants distintos → permitido.
  - `is_published` por defecto es `false` en una especialidad recién creada.
  - `ListItems` con `specialty_id` devuelve solo los de esa especialidad.
  - borrar una especialidad con ítems → error, y los ítems siguen intactos.
  - migración: un ítem `mdcong` queda con la especialidad `md`; un ítem con
    prefijo desconocido se reporta y **no** se asigna a ninguna.
- `grep -rn '"category"' .` vacío.

## Etapas

| # | Alcance | Archivos | Aceptación |
|---|---|---|---|
| 1 | `SpecialtyModel` + regenerar codecs | `model.go`, `model_orm.go` | `ormc` limpio; tipos disponibles |
| 2 | Servicio CRUD + ops + unicidad por tenant | `mcp.go` | ops montadas; unicidad de `prefix`/`slug` probada |
| 3 | `category` → `specialty_id` + `ItemFilter.specialty_id` | `model.go`, `mcp.go` | `grep '"category"'` vacío; filtro por especialidad funciona |
| 4 | Migración derivada del SKU | fichero nuevo + tests | `mdcong`→`md`; prefijo desconocido reportado, no adivinado |
| 5 | Vista: presenter de especialidad | `view.go` | `NewSpecialtyView(caller)` devuelve un `view.Presenter`, mismo patrón que `NewView` |
