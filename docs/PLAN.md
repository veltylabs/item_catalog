---
PLAN: "feat!: one reference code per specialty, currency from the module's list, help texts, exempt by default"
TAG: v0.2.0
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.

# Plan — clearer specialty and service forms

## Why

The user (the clinic's administrator) found the forms confusing:

- A specialty has two codes: `prefix` ("md", the two letters every service SKU of that specialty starts
  with, e.g. `MD001`; the seed migration maps services to specialties by `sku[:2]`) and `slug`
  ("medicina-general", only checked for uniqueness, used by nobody). Only one stays: the prefix, shown as
  **"Reference code"**. The slug is deleted.
- The service `currency` is typed by hand. It becomes a select whose options are the currencies the app
  declares when it wires the module (`Deps.Currencies`, default CLP), always selected, never typed.
- `tax_treatment` starts empty; services of a clinic are almost always exempt from IVA. It becomes
  required and new services start **Exempt**.
- `description`, `tax_treatment` and `is_active` have no help: a person does not know what to write or what
  they mean.

Language: identifiers, keys and error strings in English; **comments and docs in Spanish**. Every visible text
is an English key with its Spanish value in `lang.json` (run `langc sync .`, then fill the values given here).

## Delivery rules

- Never pause to ask permission or to report progress. When the Delivery check passes: commit, push, open the
  PR. Stuck for 30 minutes on one point: open the PR anyway, listing what fails.
- A PR that does not compile, or whose tests could not run, is a failed delivery.
- Never edit `model_orm.go` by hand: regenerate it with `ormc`.
- Environment: `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`; `go mod tidy` before `ormc`.
- The working tree you receive includes the owner's in-progress changes (`web/browser/main.go` replacing
  `web/browser.go`, `ui/browser.go` without build tag, regenerated `model_orm.go`, bumped `go.mod`): keep them.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/ormc/cmd/ormc@latest
go install webtyp.com/lang/cmd/langc@latest
```

## Ecosystem search

Currency lists: `webtyp/money` has `money.Currency` (a 3-letter code type) but no list of accepted
currencies per module; `payment_manager` keeps `CurrencyCLP`/`CurrencyUSD` constants for its own rules. A
module-level list given by the app is new here and specific to this catalog (**extend** this repo). Select
options loaded from an operation: same pattern this repo already uses for specialties (`ui/browser.go`,
`OpListSpecialties` → `SetOptions("specialty_id", …)`).

## Design gate

1. **Prior art.** Shopify (store currency set in settings, product price uses it, no per-product typing),
   Odoo (company currency + list of active currencies, a select on documents), Stripe Prices (currency from an
   allowed set). All: currencies are configuration, documents pick from them.
2. **Novice names.** `Deps.Currencies` ("the currencies this catalog accepts"), `OpListCurrencies`,
   `CurrencyOption`, `ErrUnsupportedCurrency`; label "Reference code".
3. **Ledger.** Concepts −1 (slug) +1 (accepted currencies) · Fields in the specialty form 5 → 4 · Ways to set a
   service currency: typing → choosing from a closed list (1 path).
4. **Where.** All in this repo: the catalog owns its specialties, services and the currencies it accepts.
5. **Deletes.** `Specialty.Slug` (field, column, every use), `GetSpecialtyBySlug`, `ErrSpecialtySlugExists`, the
   `Slug` of `CanonicalSpecialties` in `migration.go`, the free-text currency input.

## Stage 1 — Specialty: one reference code (`model.go`, `module.go`, `errors.go`, `migration.go`, `seed/seed.go`)

`SpecialtyModel` fields become (delete the `slug` field):

```go
{Name: "prefix", Type: input.Text(), Label: "Reference code", Help: "Two letters every service code (SKU) of this specialty starts with, e.g. MD for MD001.", NotNull: true,
	Permitted: model.Permitted{Letters: true, Minimum: 2, Maximum: 2}},
```

- `module.go`: delete `GetSpecialtyBySlug` and the "Validate slug uniqueness" block in `UpsertSpecialty`.
- `errors.go`: delete `ErrSpecialtySlugExists` and its case in `statusOf`.
- `migration.go`: delete the `Slug` field of the canonical specialty struct and every `Slug:` value; the
  `Specialty` literal it builds no longer sets `Slug`.
- `seed/seed.go`, `README.md` example and every test: remove `Slug`. Tests that asserted
  `ErrSpecialtySlugExists` are deleted; the prefix-uniqueness tests stay.

Existing databases: no deployed database holds this table yet (`mjosefa-cms` runs only locally); a local
dev database is recreated. Do not write a column-drop migration.

## Stage 2 — Accepted currencies (`model.go`, `module.go`, `errors.go`, `ops.go`)

```go
// CurrencyCLP es la moneda por defecto del catálogo.
const CurrencyCLP = "CLP"
```

`Deps` gains:

```go
Currencies []string // monedas que acepta el catálogo (ISO 4217, tres letras mayúsculas), la primera es la de los servicios nuevos; vacío = solo CurrencyCLP
```

`New`: empty → `[]string{CurrencyCLP}`; any code that is not exactly three upper-case ASCII letters →
`fmt.Err("service_catalog: Deps.Currencies has an invalid code")`. `Module` keeps `currencies []string`.

`errors.go`: `ErrUnsupportedCurrency domainError = "service_catalog: currency not accepted by this catalog"`
→ 400 in `statusOf`.

`CreateService` and `UpdateService` (and therefore `UpsertService`, which calls them — verify; if it does not,
add the same check there), right after `Validate`: currency not in `m.currencies` → `ErrUnsupportedCurrency`.

`ServiceModel` field `currency` becomes `{Name: "currency", Type: input.Select(), Label: "Currency", NotNull: true, Permitted: model.Permitted{Letters: true, Minimum: 3, Maximum: 3}},`.

New model and operation:

```go
var CurrencyOptionModel = model.Definition{
	Name: "currency_option",
	Fields: model.Fields{
		{Name: "code", Type: model.Text()},
	},
}

var ListCurrenciesArgsModel = model.Definition{
	Name: "list_currencies_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
	},
}
```

`OpListCurrencies = "list_currencies"` → `reg.Operation(OpListCurrencies, m.opListCurrencies).Requires("service", model.Read).Accepts(&ListCurrenciesArgs{})`;
answers a `CurrencyOptionList` with one `CurrencyOption{Code}` per accepted currency, in `Deps` order.

| Case | Input | Result |
|---|---|---|
| C1 | `Deps.Currencies` nil, create service in CLP | created |
| C2 | `Deps.Currencies` nil, create service in USD | `ErrUnsupportedCurrency`, 400 |
| C3 | `Deps.Currencies{"CLP","USD"}`, update service to USD | updated |
| C4 | `Deps.Currencies{"clp"}` | `New` error `service_catalog: Deps.Currencies has an invalid code` |
| C5 | `list_currencies` with `{"CLP","USD"}` | `[{code: CLP}, {code: USD}]` |

## Stage 3 — Tax treatment and help texts (`model.go`)

```go
{Name: "description", Type: input.Textarea(), Label: "Description", Help: "Public information: patients read it when booking. Say what the service includes, how long it takes or how to prepare.", OmitEmpty: true},
{Name: "tax_treatment", Type: input.Select(), Label: "Tax treatment", Help: "Exempt: a health service without IVA (consultations and most exams). Taxed: it carries 19% IVA.", NotNull: true},
{Name: "is_active", Type: input.Checkbox(), Label: "Active", Help: "Only active services can be booked and charged. Turn it off to stop offering it without losing its history.", NotNull: true},
```

`CreateService`/`UpdateService`: `TaxTreatment` not `TaxTaxed`/`TaxExempt` → `ValidationError` (empty included:
the field is now required). Test: create with empty tax treatment → 400.

## Stage 4 — Screen (`ui/browser.go`)

- New services start exempt and with the catalog's first currency:

```go
defaultCurrency := servicecatalog.CurrencyCLP
// … NewRecord:
return &servicecatalog.Service{TenantId: tenantID, Currency: defaultCurrency, TaxTreatment: servicecatalog.TaxExempt, IsActive: true}
```

- After building the views, load the currencies like the specialties:

```go
cur := &servicecatalog.CurrencyOptionList{}
caller.Call(servicecatalog.ModelName+"."+servicecatalog.OpListCurrencies, nil, cur, func(err error) {
	if err != nil {
		dom.Log(errListCurrencies, err)
		return
	}
	opts := make([]fmt.KeyValue, 0, len(*cur))
	for _, c := range *cur {
		opts = append(opts, fmt.KeyValue{Key: c.Code, Value: c.Code})
	}
	if len(opts) > 0 {
		defaultCurrency = opts[0].Key
	}
	servicesView.SetOptions("currency", opts...)
})
```

  with `const errListCurrencies = "service_catalog: could not load currencies for the service form:"`.
- The tax options keep their keys `Taxed`/`Exempt`.
- `lang.json` holds a stale Spanish key `"Seleccione una especialidad antes de guardar"` that no Go file uses
  any more: it is deleted in Stage 5 (no code change).

## Stage 5 — Dictionary (`lang.json`)

Run `langc sync .` then set exactly these Spanish values (keys that are no longer used are removed by hand:
`"Slug"`, `"slug"`, `"Seleccione una especialidad antes de guardar"`, and the old slug help key):

```json
"Reference code": ["Código de referencia"],
"Two letters every service code (SKU) of this specialty starts with, e.g. MD for MD001.": ["Dos letras con que empieza el código (SKU) de cada servicio de esta especialidad; por ejemplo MD para MD001."],
"Public information: patients read it when booking. Say what the service includes, how long it takes or how to prepare.": ["Información pública: la verá el paciente al reservar. Cuenta qué incluye la atención, cuánto dura o cómo prepararse."],
"Exempt: a health service without IVA (consultations and most exams). Taxed: it carries 19% IVA.": ["Exento: prestación de salud sin IVA (consultas y la mayoría de los exámenes). Afecto: lleva IVA 19 %."],
"Only active services can be booked and charged. Turn it off to stop offering it without losing its history.": ["Solo los servicios activos se pueden reservar y cobrar. Desactívalo para dejar de ofrecerlo sin perder su historial."],
"service_catalog: currency not accepted by this catalog": ["service_catalog: el catálogo no acepta esa moneda"],
"service_catalog: Deps.Currencies has an invalid code": ["service_catalog: Deps.Currencies tiene un código inválido"],
"code": ["código"],
```

After filling, `langc sync .` must report no untranslated key.

## Stage 6 — Docs (Spanish)

- `README.md`: wiring example with `Currencies: []string{servicecatalog.CurrencyCLP}` and the sentence "Si no se
  indica, el catálogo acepta solo CLP"; operations table adds `list_currencies`; specialty example without `Slug`.
- `docs/ARCHITECTURE.md`: Specialty = "taxonomía con un código de referencia de dos letras (prefijo de los SKU)";
  remove "slug" from the uniqueness line; new line on accepted currencies.

## Deletions — each must grep empty

```bash
grep -rn "Slug\|slug" --include=*.go --include=*.md --include=*.json --exclude=PLAN.md .
grep -rn "Seleccione una especialidad" --exclude=PLAN.md .
```

## Delivery check

```bash
go mod tidy && ormc && langc sync . && gotest
```

All green, the greps empty, `git diff --stat main` lists every file named in the stages.

## Stages

| # | What | Files |
|---|---|---|
| 1 | One reference code | `model.go`, `model_orm.go` (generated), `module.go`, `errors.go`, `migration.go`, `seed/seed.go`, `tests/*` |
| 2 | Accepted currencies | `model.go`, `module.go`, `errors.go`, `ops.go`, `tests/currency_test.go` |
| 3 | Tax treatment required, help texts | `model.go`, `module.go`, `tests/catalog_test.go` |
| 4 | Screen defaults and currency select | `ui/browser.go` |
| 5 | Dictionary | `lang.json` |
| 6 | Docs | `README.md`, `docs/ARCHITECTURE.md` |
