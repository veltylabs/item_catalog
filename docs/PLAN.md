---
PLAN: "feat(ui): the catalog explains itself — description on the group, Servicios and Especialidades"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 3510063836519186808
---

> Read AGENTS.md at the repo root before any change.

# PLAN — the catalog screens explain themselves

## Why

Rule of the ecosystem (skill **ui-consistency**): a module explains itself and guides step by step.
The shell (`webtyp.com/layout/platformd`) already draws a «?» button next to the title and a
welcome card for any screen that implements `platformd.Described`. `service_catalog` does not
implement it, so a first-time user opens «Catálogo» with no idea what it is for.
`pharmacy_manager` already does it this way (`ui/browser.go`, `platformd.Describe`).

## Tools

```
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
```

`go.mod` already requires `webtyp.com/layout v0.3.64`, which has `platformd.Describe`. Do not change
any dependency version.

## APIs used

| Symbol | Exact signature | Tag / file |
|---|---|---|
| `platformd.Describe` | `func Describe(m UIModule, text lang.Text, steps ...Step) UIModule` — wraps a group into a module that still implements `GroupModule` | `webtyp.com/layout v0.3.64`, `platformd/capabilities.go` |
| `platformd.Described` | `interface { Description() lang.Text; Steps() []Step }` | same |
| `platformd.GroupModule` | `interface { UIModule; Children() []UIModule }` | same |
| `platformd.NewUIModule` | `func NewUIModule(id, label string, icon svg.Icon, view Component, onActivate ...func()) UIModule` | `platformd/factory.go` |
| `platformd.NewGroupUIModule` | `func NewGroupUIModule(id, label string, icon svg.Icon, children []UIModule) UIModule` | same |
| `lang.Text` | `type Text string` | `webtyp.com/lang v0.1.9` |

If a signature in the published tag differs from this table, stop and name the symbol in the PR;
never adapt it by guessing.

## Stage 1 — `ui/browser.go`

1. Add the import `"webtyp.com/lang"`.
2. Below `errListSpecialties`, add exactly:

```go
// Textos de ayuda que el chasis muestra en el botón «?» y en la bienvenida de
// cada pantalla. Las claves están en inglés; la traducción vive en lang.json.
const (
	descCatalog     lang.Text = "The services the clinic offers and the specialties that group them. Start with the specialties, then add each service with its price."
	descServices    lang.Text = "Each service the clinic offers, with its specialty, price and tax treatment. Pick one in the list to edit it, or press + to add a new one."
	descSpecialties lang.Text = "Specialties group the services (for example, dentistry or kinesiology). Create them first: every service must belong to one."
)
```

3. Replace the two `platformd.NewUIModule(...)` lines and the final `return` with:

```go
	servicesMod := platformd.Describe(
		platformd.NewUIModule(ID+".services", "Servicios", IconServices, servicesView), descServices)
	specialtiesMod := platformd.Describe(
		platformd.NewUIModule(ID+".specialties", "Especialidades", IconSpecialties, specialtiesView), descSpecialties)
```

```go
	return platformd.Describe(platformd.NewGroupUIModule(ID, Label, IconCatalog, []platformd.UIModule{
		servicesMod,
		specialtiesMod,
	}), descCatalog), nil
```

Nothing else in the file changes (the `SetOptions` calls keep using `servicesView`).

## Stage 2 — `lang.json`

Run `langc sync .`, then fill the three new keys with exactly:

```json
    "Each service the clinic offers, with its specialty, price and tax treatment. Pick one in the list to edit it, or press + to add a new one.": ["Cada servicio que ofrece la clínica, con su especialidad, precio y tratamiento tributario. Elige uno en la lista para editarlo, o presiona + para agregar uno nuevo."],
    "Specialties group the services (for example, dentistry or kinesiology). Create them first: every service must belong to one.": ["Las especialidades agrupan los servicios (por ejemplo, odontología o kinesiología). Créalas primero: cada servicio debe pertenecer a una."],
    "The services the clinic offers and the specialties that group them. Start with the specialties, then add each service with its price.": ["Los servicios que ofrece la clínica y las especialidades que los agrupan. Empieza por las especialidades y luego agrega cada servicio con su precio."],
```

If `langc sync .` reports other empty values, fill them too in Spanish and list them in the PR.

## Stage 3 — test `tests/ui_described_test.go` (new, `//go:build browser`, package `tests`)

Uses the existing `mockCaller` (`tests/service_catalog_view_mock_test.go`) and `testIDGen`
(`tests/idgen_test.go`).

```go
//go:build browser

package tests

import (
	"testing"

	"github.com/veltylabs/service_catalog/ui"
	"webtyp.com/layout/platformd"
)

// Cada pantalla del catálogo se explica sola: el grupo y sus dos hijos
// implementan platformd.Described con un texto no vacío.
func TestBrowser_EveryScreenIsDescribed(t *testing.T) {
	screen, err := ui.Browser(&mockCaller{}, &testIDGen{}, "")
	if err != nil {
		t.Fatalf("ui.Browser: %v", err)
	}
	group, ok := screen.(platformd.GroupModule)
	if !ok {
		t.Fatalf("expected a platformd.GroupModule, got %T", screen)
	}
	all := append([]platformd.UIModule{screen}, group.Children()...)
	if len(all) != 3 {
		t.Fatalf("expected the group and 2 children, got %d screens", len(all))
	}
	for _, m := range all {
		d, ok := m.(platformd.Described)
		if !ok {
			t.Errorf("%s must implement platformd.Described", m.ModelName())
			continue
		}
		if d.Description() == "" {
			t.Errorf("%s has an empty description", m.ModelName())
		}
	}
}
```

The existing tests that type-assert `platformd.GroupModule` and read `Children()[0].View()` must
keep passing unchanged: `Describe` keeps both capabilities.

## Stage 4 — language

Translate to Spanish every README, `docs/*` file, diagram and code comment of this repo that is
still in English (only those; never rewrite what is already Spanish). Add a short `// Ejemplo: …`
line to every exported symbol's comment that lacks one. In `README.md`, the line about `ui/` must
also say that every screen carries a description (`platformd.Describe`).

## Delivery check

```
GOPRIVATE=webtyp.com/*,github.com/veltylabs/* GOFLAGS=-mod=mod gotest
```

- A PR that does not compile, or whose tests could not run, is a failed delivery.
- Never pause to ask permission or report progress: when the check passes, commit, push and open
  the PR. Stuck for 30 minutes: open the PR anyway listing what fails.
- Never edit a generated file (`model_orm.go`).
- Commit after every stage (`git add -A` then `git commit -m "stage N: …"`); never `git reset`,
  `git restore`, `git checkout -- <file>` or `git stash` on your own work.
- `git diff --stat main` must list every file of the stages and nothing the plan did not order.

## Stages

| # | Files | Done when |
|---|---|---|
| 1 | `ui/browser.go` | group and both children wrapped with `platformd.Describe` |
| 2 | `lang.json` | no empty value |
| 3 | `tests/ui_described_test.go` | test passes in the browser suite |
| 4 | `README.md`, `docs/*`, comments | no English comment or doc left |
