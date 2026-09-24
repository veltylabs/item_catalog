package seed

import (
	itemcatalog "github.com/veltylabs/item_catalog"
	"webtyp.com/fmt"
)

const currencyCLP = "CLP"

type Data struct {
	Specialties []itemcatalog.Specialty
	Items       []itemcatalog.CatalogItem
}

// Load creates three canonical specialties and one service in each.
func Load(m *itemcatalog.Module, tenantID string) (Data, error) {
	data := Data{}

	targetSlugs := []string{"medicina-general", "traumatologia", "ecografia"}

	for i, slug := range targetSlugs {
		var cs itemcatalog.CanonicalSpecialty
		found := false
		for _, canonical := range itemcatalog.CanonicalSpecialties {
			if canonical.Slug == slug {
				cs = canonical
				found = true
				break
			}
		}
		if !found {
			return data, fmt.Err("seed: canonical specialty not found", slug)
		}

		spec, err := m.UpsertSpecialty(itemcatalog.Specialty{
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

		createdItem, err := m.CreateItem(itemcatalog.CatalogItem{
			TenantId:    tenantID,
			SpecialtyId: specID,
			Sku:         specDef.sku,
			Name:        specDef.name,
			Type:        itemcatalog.ItemTypeService,
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
