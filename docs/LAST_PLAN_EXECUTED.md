---
PLAN: "feat: replace decktabs with platformd accordion subnavigation in item_catalog"
TAG: v0.1.18
EXECUTOR: jules
REVIEWER: none
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

# Plan: Migración a Acordeón de platformd y Rediseño de item_catalog

Este plan elimina completamente el uso del componente `decktabs` en `item_catalog`. Sustituye las pestañas internas incrustadas por la nueva sub-navegación tipo acordeón en `platformd`. La pestaña "Servicios" se ubica en primer lugar (arriba) por su alta frecuencia de edición y consulta de precios, y "Especialidades" se ubica al final (abajo) por tratarse de una configuración base. Cada elemento cuenta con su propio ícono SVG representativo.

---

## Design gate

### 1. Prior art
- **Admin Dashboards modernos:** Paneles de administración donde la gestión de un catálogo de productos/servicios ubica el listado principal de items en la parte superior y las tablas maestras/categorías asociadas al final del menú desplegable.
- **Navegación jerárquica en apps WebTyp:** Eliminación de franjas de pestañas dentro del contenedor de vista (`RightPanel` / `crudview`), otorgando el 100% de la altura útil de la pantalla a los datos.

### 2. Novice-name test
Mantiene los identificadores `catalog_item.services` y `catalog_item.specialties`, con etiquetas en español visibles: "Servicios" y "Especialidades".

### 3. Complexity ledger
- Elimina la dependencia de `webtyp.com/components/decktabs`.
- Reemplaza la instancia de `decktabs.DeckTabs` por un `platformd.GroupModule` construido con `platformd.NewGroupUIModule`.
- Añade los trazados SVG para Servicios (`catalog_item_services`) y Especialidades (`catalog_item_specialties`).

### 4. Where it belongs
Pertenece a `github.com/veltylabs/item_catalog/ui`.

### 5. What it deletes
- Elimina el import `webtyp.com/components/decktabs` en `ui/browser.go`.
- Elimina el widget `decktabs.DeckTabs` y el contenedor de pestañas en la vista de este módulo.

---

## Etapas de Ejecución

### Etapa 1: Registro de Íconos SVG en `ui/svg.go`
- **Archivos a modificar:**
  - `ui/svg.go`:
    - Incorporar el ícono de Servicios extraído de `/home/cesar/Dev/Project/veltylabs/monjitaschillan.cl/docs/icons/service-icon.svg` bajo el ID `catalog_item_services`:
      - ViewBox: `0 0 16 16`
      - Paths optimizados para sprite.
    - Incorporar el ícono de Especialidades extraído de `/home/cesar/Dev/Resources/iconos/iconosSVG/153-aid-kit.svg` bajo el ID `catalog_item_specialties`:
      - ViewBox: `0 0 16 16`
      - Path: `M14 4h-3v-2c0-0.55-0.45-1-1-1h-4c-0.55 0-1 0.45-1 1v2h-3c-1.1 0-2 0.9-2 2v8c0 1.1 0.9 2 2 2h12c1.1 0 2-0.9 2-2v-8c0-1.1-0.9-2-2-2zM6 2h4v2h-4v-2zM12 11h-3v3h-2v-3h-3v-2h3v-3h2v3h3v2z`
    - Exportar constantes tipadas `IconServices = svg.Icon("catalog_item_services")` e `IconSpecialties = svg.Icon("catalog_item_specialties")`.

### Etapa 2: Configuración para Pruebas Locales en `go.mod`
- **Archivos a modificar:**
  - `go.mod`:
    - Añadir directiva `replace webtyp.com/layout => ../../../webtyp/layout` para consumir la implementación local de `GroupModule` antes de su publicación remota.

### Etapa 3: Refactorización de `ui/browser.go`
- **Archivos a modificar:**
  - `ui/browser.go`:
    - Eliminar `import "webtyp.com/components/decktabs"`.
    - Crear `itemsView` con ID `ID + ".services"` (o `ID + ".items"`).
    - Crear `specialtiesView` con ID `ID + ".specialties"`.
    - Construir los dos `UIModule` hijos:
      ```go
      servicesMod := platformd.NewUIModule(ID+".services", "Servicios", IconServices, itemsView)
      specialtiesMod := platformd.NewUIModule(ID+".specialties", "Especialidades", IconSpecialties, specialtiesView)
      ```
    - Retornar el módulo agrupado:
      ```go
      return platformd.NewGroupUIModule(ID, Label, svg.Icon(ID), []platformd.UIModule{
          servicesMod,
          specialtiesMod,
      }), nil
      ```
    - Asegurar que `picker.load(nil)` se siga invocando adecuadamente para cargar las especialidades del formulario.

### Etapa 4: Actualización de Tests
- **Archivos a modificar:**
  - `tests/item_catalog_view_test.go`:
    - Eliminar aserciones que buscaban clases de `decktabs`.
    - Verificar que `Browser()` retorna una implementación de `platformd.GroupModule`.
    - Verificar que tiene 2 hijos: "Servicios" primero y "Especialidades" segundo.
  - `tests/item_catalog_view_wasm_test.go`:
    - Verificar compilación y renderizado WASM.

---

## Tabla de Entregables de Etapas

| Etapa | Archivo | Acción | Descripción |
|-------|---------|--------|-------------|
| 1 | `ui/svg.go` | MODIFICAR | Definir sprites `IconServices` e `IconSpecialties` |
| 2 | `go.mod` | MODIFICAR | Añadir replace temporal de `webtyp.com/layout` |
| 3 | `ui/browser.go` | MODIFICAR | Eliminar `decktabs`, retornar `GroupModule` con Servicios y Especialidades |
| 4 | `tests/item_catalog_view_test.go` | MODIFICAR | Tests adaptados a `GroupModule` |
| 4 | `tests/item_catalog_view_wasm_test.go` | MODIFICAR | Conformance WASM |

---

## Criterios de Aceptación
1. `gotest ./...` ejecuta todos los tests en verde en `veltylabs/modules/item_catalog`.
2. La vista principal no contiene ninguna referencia visual ni en el DOM a `decktabs`.
3. Al despachar este plan, ningún archivo fuera de `item_catalog` es alterado (excepto el enlace local `replace`).

