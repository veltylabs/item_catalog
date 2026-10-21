# Plan Maestro: Sub-navegación Acordeón y Eliminación de DeckTabs

> **Estado:** EN PROGRESO · 2026-10-09 · Migración de sub-navegación a acordeón en platformd y rediseño de item_catalog

## 1. Motivación y Visión

El componente `decktabs` (`webtyp/components/decktabs`) consume un espacio vertical excesivo en la cabecera del contenido principal (especialmente en pantallas de escritorio), restando visibilidad a las tablas y formularios.
Adicionalmente, fragmenta la navegación de la aplicación: el usuario debe mirar la barra de navegación lateral y luego una barra horizontal secundaria en el contenido.

La solución arquitectónica correcta es trasladar la jerarquía de navegación al contenedor principal de la aplicación (`webtyp/layout/platformd`), transformando los módulos que contienen sub-pantallas en **menús desplegables tipo acordeón hacia abajo**:
- **Navegación en un solo clic:** Al presionar el botón padre por primera vez, el acordeón se despliega y selecciona inmediatamente el primer elemento hijo más cercano (arriba hacia abajo).
- **Colapso natural:** Un segundo clic sobre el botón padre abierto colapsa el acordeón.
- **Soporte móvil idéntico:** En el cajón (*drawer*) móvil, los elementos hijos se despliegan indentados. El botón flotante de cabecera refleja el ícono de la pantalla activa.
- **Fase de prueba local con `replace`:** Antes de publicar versiones con `gopush` y eliminar permanentemente `decktabs`, se prueba en vivo en `item_catalog` mediante `replace webtyp.com/layout` en `go.mod`.

---

## 2. Grafo de Dependencias y Fases

```mermaid
flowchart TD
    subgraph Fase 1: webtyp/layout [webtyp/layout]
        W1[Definir GroupModule en platformd] --> W2[Implementar acordeón y auto-selección en platformd.go]
        W2 --> W3[Añadir reglas CSS en css.go]
        W3 --> W4[Tests unitarios en platformd_test.go]
    end

    subgraph Fase 2: item_catalog [veltylabs/modules/item_catalog]
        C1[Configurar go.mod replace con layout local] --> C2[Incorporar íconos SVG en ui/svg.go]
        C2 --> C3[Refactorizar ui/browser.go a NewGroupUIModule]
        C3 --> C4[Actualizar tests en tests/]
        C4 --> C5[Verificación visual desktop y mobile]
    end

    subgraph Fase 3: Publicación y Limpieza [En cascada]
        P1[Aprobación visual de usuario] --> P2[Desacoplar decktabs en chatview]
        P2 --> P3[Eliminar webtyp/components/decktabs]
        P3 --> P4[gopush webtyp/layout y webtyp/components]
        P4 --> P5[Remover replace en item_catalog y gopush]
    end

    Fase 1 --> Fase 2 --> Fase 3
```

---

## 3. Matriz de Librerías y Módulos Afectados

| Repositorio / Librería | Archivo del Plan | Estado | Descripción del Cambio |
|------------------------|------------------|--------|------------------------|
| `webtyp/layout` | `docs/LAST_PLAN_EXECUTED.md` | COMPLETADO | Soporte de `GroupModule`, acordeón en `platformd`, chevron con rotación de 180°, submenú anidado con fondo Secondary, sin doble selección en el padre, y tests 100% verdes. |
| `veltylabs/modules/item_catalog` | `docs/LAST_PLAN_EXECUTED.md` | VALIDACIÓN VISUAL | Reemplazo de `decktabs` por `NewGroupUIModule`, íconos para Servicios y Especialidades, módulos ficticios Inicio/Ajustes agregados en `web/browser.go` para pruebas. |
| `webtyp/layout/chatview` | (Fase 3) | PENDIENTE | Migración de tabs a acordeón tras aprobación visual final. |
| `webtyp/components` | (Fase 3) | PENDIENTE | Eliminación física del paquete `decktabs/` y publicación en cascada con `gopush`. |

---

## 4. Criterios de Aceptación Globales

1. `webtyp/layout`: Pasa `gotest ./...` con pruebas para `GroupModule`, apertura en 1 clic y colapso en 2do clic, chevron y clases CSS verificadas.
2. `item_catalog`: Pasa `gotest ./...` tanto en el carril stdlib como en el carril WASM (race ✅, tests ✅, wasm ✅).
3. El área principal de trabajo de "Servicios" y "Especialidades" en `item_catalog` no renderiza ninguna barra superior de pestañas (`decktabs` eliminado de la vista).
4. Jerarquía visual clara:
   - El padre "Catálogo" no se marca en ámbar cuando un hijo está activo (solo estado `Open`).
   - El padre cuenta con flecha chevron que rota 180° hacia arriba al expandirse.
   - El contenedor de hijos (`submenu`) tiene fondo sutil diferenciado (`Secondary`), tipografía reducida (`TextSm`), e indentación interna.
   - Módulos hermanos ("Inicio", "Ajustes") interactúan fluidamente con el acordeón.

