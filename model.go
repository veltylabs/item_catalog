package servicecatalog

import (
	"webtyp.com/input"
	"webtyp.com/model"
)

// Helpers to satisfy ormc v0.9.24 AST static type parser:
var (
	Checkbox_FieldBool = input.Checkbox()
	Decimal_FieldFloat = input.Decimal()

	BaseBool_FieldBool = model.Bool()
	BaseInt_FieldInt   = model.Int()
)

const (
	TaxTaxed  = "TAXED"
	TaxExempt = "EXEMPT"
)

var SpecialtyModel = model.Definition{
	Name: "specialty",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "prefix", Type: input.Text(), Label: "Prefix", NotNull: true,
			Permitted: model.Permitted{Letters: true, Minimum: 2, Maximum: 2}},
		{Name: "slug", Type: input.Text(), Label: "Slug", NotNull: true,
			Permitted: model.Permitted{Letters: true, Numbers: true, Extra: []rune{'-'}, Minimum: 1, Maximum: 60}},
		// input.Text()'s own baseline Extra ('.', ',', '(', ')') no incluye
		// '+' — "Dental (Odontología + Ortodoncia)" lo necesita, así que
		// este campo declara su propio whitelist para reemplazar el piso
		// del Kind (Field.Validate: un whitelist positivo propio reemplaza
		// el floor de Text, no lo suma). Tilde cubre las vocales acentuadas
		// sin listarlas una a una.
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true,
			Permitted: model.Permitted{Letters: true, Tilde: true, Spaces: true, Extra: []rune{'+', '(', ')'}, Minimum: 1, Maximum: 100}},
		{Name: "description", Type: input.Textarea(), Label: "Description", OmitEmpty: true},
		{Name: "position", Type: model.Int(), OmitEmpty: true},
		// NotNull deliberadamente ausente: para un bool, NotNull rechaza el
		// zero value (ValidateFields: field.NotNull && IsZeroPtr, y
		// IsZeroPtr para FieldBool es !*p) — con NotNull:true sería
		// imposible crear una especialidad SIN publicar, justo lo opuesto
		// de "cerrado por defecto". El false-por-defecto ya lo garantiza
		// el zero value de Go, no una restricción de esquema.
		{Name: "is_published", Type: input.Checkbox(), Label: "Published"},
		{Name: "updated_at", Type: model.Int(), OmitEmpty: true},
	},
}

var ServiceModel = model.Definition{
	Name: "service",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "specialty_id", Type: input.Select(), Label: "Specialty", Ref: &SpecialtyModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "sku", Type: input.Text(), Label: "SKU", NotNull: true, Permitted: model.Permitted{Letters: true, Numbers: true, Extra: []rune{'-'}, Minimum: 1, Maximum: 50}},
		{Name: "name", Type: input.Text(), Label: "Name", NotNull: true, Permitted: model.Permitted{Minimum: 1, Maximum: 255}},
		{Name: "description", Type: input.Textarea(), Label: "Description", OmitEmpty: true},
		{Name: "tax_treatment", Type: input.Select(), Label: "Tax treatment", OmitEmpty: true},
		{Name: "price", Type: input.Decimal(), Label: "Price", NotNull: true},
		{Name: "currency", Type: input.Text(), Label: "Currency", NotNull: true, Permitted: model.Permitted{Letters: true, Minimum: 3, Maximum: 3}},
		{Name: "is_active", Type: input.Checkbox(), Label: "Active", NotNull: true},
		{Name: "updated_at", Type: model.Int(), OmitEmpty: true},
	},
}

var ServiceAgreementModel = model.Definition{
	Name: "service_agreement",
	Fields: model.Fields{
		{Name: "id", Type: model.Text(), DB: &model.FieldDB{PK: true}, OmitEmpty: true},
		{Name: "tenant_id", Type: model.Text(), NotNull: true},
		{Name: "service_id", Type: model.Text(), Ref: &ServiceModel, DB: &model.FieldDB{RefColumn: "id"}, NotNull: true},
		{Name: "insurer", Type: input.Text(), Label: "Insurer", NotNull: true, Permitted: model.Permitted{Letters: true, Spaces: true, Minimum: 1, Maximum: 100}},
		{Name: "code", Type: input.Text(), Label: "Code", OmitEmpty: true, Permitted: model.Permitted{Letters: true, Numbers: true, Extra: []rune{'-'}, Minimum: 1, Maximum: 50}},
		{Name: "price", Type: input.Decimal(), Label: "Price", OmitEmpty: true},
		{Name: "is_active", Type: input.Checkbox(), Label: "Active", NotNull: true},
		{Name: "updated_at", Type: model.Int(), OmitEmpty: true},
	},
}

var ServiceFilterModel = model.Definition{
	Name: "item_filter",
	Fields: model.Fields{
		{Name: "specialty_id", Type: model.Text()},
		{Name: "active_only", Type: model.Bool()},
		{Name: "limit", Type: model.Int()},
		{Name: "offset", Type: model.Int()},
	},
}

var ListServicesArgsModel = model.Definition{
	Name: "list_services_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "specialty_id", Type: model.Text()},
		{Name: "active_only", Type: model.Bool()},
		{Name: "limit", Type: model.Int()},
		{Name: "offset", Type: model.Int()},
	},
}

var GetServiceArgsModel = model.Definition{
	Name: "get_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}

var FindServiceBySKUArgsModel = model.Definition{
	Name: "find_by_sku_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "sku", Type: model.Text()},
	},
}

var DeactivateServiceArgsModel = model.Definition{
	Name: "deactivate_item_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}

var DeleteServiceArgsModel = model.Definition{
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
		{Name: "service_id", Type: model.Text()},
	},
}

var DeleteAgreementArgsModel = model.Definition{
	Name: "delete_agreement_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}

var ListSpecialtiesArgsModel = model.Definition{
	Name: "list_specialties_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
	},
}

var GetSpecialtyArgsModel = model.Definition{
	Name: "get_specialty_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}

var DeleteSpecialtyArgsModel = model.Definition{
	Name: "delete_specialty_args",
	Fields: model.Fields{
		{Name: "tenant_id", Type: model.Text()},
		{Name: "id", Type: model.Text()},
	},
}
