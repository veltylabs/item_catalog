package seed

import (
	servicecatalog "github.com/veltylabs/service_catalog"
	"webtyp.com/fmt"
)

const currencyCLP = "CLP"

type Data struct {
	Specialties []servicecatalog.Specialty
	Items       []servicecatalog.Service
}

// Load creates three canonical specialties and one service in each.
func Load(m *servicecatalog.Module, tenantID string) (Data, error) {
	data := Data{}

	targetSlugs := []string{"medicina-general", "traumatologia", "ecografia"}

	for i, slug := range targetSlugs {
		var cs servicecatalog.CanonicalSpecialty
		found := false
		for _, canonical := range servicecatalog.CanonicalSpecialties {
			if canonical.Slug == slug {
				cs = canonical
				found = true
				break
			}
		}
		if !found {
			return data, fmt.Err("seed: canonical specialty not found", slug)
		}

		spec, err := m.UpsertSpecialty(servicecatalog.Specialty{
			TenantId:    tenantID,
			Prefix:      cs.Prefix,
			Slug:        cs.Slug,
			Name:        cs.Name,
			Position:    int64(i + 1),
			IsPublished: true,
		})
		if err != nil {
			return data, fmt.Err("seed: UpsertSpecialty", cs.Slug, err)
		}
		data.Specialties = append(data.Specialties, spec)
	}

	type itemSpec struct {
		slug  string
		sku   string
		name  string
		price float64
	}

	itemsToCreate := []itemSpec{
		{slug: "medicina-general", sku: "MD001", name: "Consulta Medicina General", price: 25000},
		{slug: "traumatologia", sku: "TR001", name: "Consulta Traumatología", price: 35000},
		{slug: "ecografia", sku: "EC001", name: "Ecografía Abdominal", price: 40000},
	}

	for _, specDef := range itemsToCreate {
		var specID string
		found := false
		for _, spec := range data.Specialties {
			if spec.Slug == specDef.slug {
				specID = spec.Id
				found = true
				break
			}
		}
		if !found {
			return data, fmt.Err("seed: specialty id not found", specDef.slug)
		}

		createdItem, err := m.CreateItem(servicecatalog.Service{
			TenantId:    tenantID,
			SpecialtyId: specID,
			Sku:         specDef.sku,
			Name:        specDef.name,
			Price:       specDef.price,
			Currency:    currencyCLP,
			IsActive:    true,
		})
		if err != nil {
			return data, fmt.Err("seed: CreateItem", specDef.sku, err)
		}
		data.Items = append(data.Items, createdItem)
	}

	return data, nil
}
