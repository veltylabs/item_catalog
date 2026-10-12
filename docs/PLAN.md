---
PLAN: "feat: ServiceCard — nombre, precio de lista, moneda y estado de un servicio"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 6641213984342180958
PR: https://github.com/veltylabs/item_catalog/pull/10
---

> Este plan se despacha con el flujo CodeJob. Ver skill: agents-workflow.

> **Pieza I** de
> [`mjosefa-cms/docs/RESERVA_Y_PAGOS_MASTER_PLAN.md`](https://github.com/veltylabs/mjosefa-cms/blob/main/docs/RESERVA_Y_PAGOS_MASTER_PLAN.md).
> Compuerta de la pieza B (`appointment_booking`).

# Plan — `item_catalog`: método `ServiceCard`

## 0. Contexto

`appointment_booking` congela en cada reserva el precio que se cobra. Hoy usa solo el
`PriceOverride` de la configuración profesional × servicio y fija la moneda en `CLP`: si la
configuración no sobrescribe el precio, la reserva queda en $0. Debe usar el precio de lista
del catálogo cuando no hay sobrescritura, y para eso su `CatalogReader` necesita un método con
**tipos primitivos** (no puede importar este módulo) que devuelva nombre, precio y moneda.

## Design gate

1. **Prior art:** `ServiceExists` (puerto estrecho satisfecho estructuralmente).
2. **Prueba del nombre:** `ServiceCard(tenantId, serviceId)` "la tarjeta del servicio".
3. **Libro de complejidad:** +1 método exportado.
4. **Dónde vive:** aquí; el dueño del catálogo.
5. **Qué borra:** nada.

## 1. Reglas

- Sin `strings`/`strconv`/`errors`/`fmt` de la stdlib; sin `map`; sin `reflect`. Comentarios
  en español.
- **Todas las pruebas en `tests/`**. `gotest` (prerrequisito:
  `go install webtyp.com/devflow/cmd/gotest@latest`).
- **TDD:** firma con `panic("sin implementar")` + pruebas de §3 → `gotest` → confirmar rojo →
  commit `test: ServiceCard en rojo` → implementar → verde → commit `feat: ServiceCard`.
- **Cobertura:** `ServiceCard` en 100.0%:
  `go test -coverpkg=github.com/veltylabs/item_catalog -coverprofile=/tmp/c.out ./tests/... && go tool cover -func=/tmp/c.out | grep ServiceCard`.
- Diagrama: [`docs/diagrams/service_card.md`](diagrams/service_card.md) (ya escrito).

## 2. Implementación (`mcp.go`, junto a `ServiceExists`)

```go
// ServiceCard devuelve nombre, precio de lista, moneda y estado de un servicio
// del catálogo. Satisface CatalogReader.ServiceCard de appointment_booking
// estructuralmente. Ausente, de otro tenant o que no es servicio ⇒ ErrNotFound.
func (m *Module) ServiceCard(tenantId, serviceId string) (name string, price float64, currency string, active bool, err error)
```

Diagrama: vacío ⇒ `ErrNotFound`; `GetItem` ⇒ `ErrNotFound` tal cual; otro error tal cual;
`item.Type != ItemTypeService` ⇒ `ErrNotFound`; si no ⇒
`item.Name, item.Price, item.Currency, item.IsActive, nil`.

## 3. Pruebas — trazabilidad (`tests/service_card_test.go`)

| Rama | Prueba |
|---|---|
| `V1-sí` | `TestServiceCard_EmptyArgsNotFound` |
| `V2-ErrNotFound` | `TestServiceCard_UnknownOrOtherTenantNotFound` |
| `V2-otro error` | `TestServiceCard_StorageErrorPropagates` (un `storage.Conn` falso declarado en `tests/` que envuelve `storage/mem` y falla en `Query`) |
| `V3-no` | `TestServiceCard_ProductIsNotFound` |
| `V3-sí` | `TestServiceCard_ReturnsNamePriceCurrencyActive` (activo e inactivo) |

## 4. Documentación

`README.md`: una línea en la sección de lectores para otros módulos.

## 5. Criterios de aceptación

- `gotest` en verde; `ServiceCard` en 100.0%; cada prueba de §3 con su nombre exacto.
- Historial: commit `test:` en rojo antes del `feat:`.
- `git diff --stat main` lista exactamente `mcp.go`, `tests/service_card_test.go`, `README.md`;
  nada borrado.

## Executor notes
- Implementing `ServiceCard` using `m.GetItem` as planned.
- Tests updated to correctly fail/mock storage errors through the `Compile` interceptor instead of `Query` interceptor.
- `README.md` updated with `ServiceCard` under `Readers (Cross-Module)`.
