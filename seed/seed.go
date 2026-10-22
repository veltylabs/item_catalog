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

	targetPrefixes := []string{"md", "tr", "ec"}

	for i, prefix := range targetPrefixes {
		var cs servicecatalog.CanonicalSpecialty
		found := false
		for _, canonical := range servicecatalog.CanonicalSpecialties {
			if canonical.Prefix == prefix {
				cs = canonical
				found = true
				break
			}
		}
		if !found {
			return data, fmt.Err("seed: canonical specialty not found", prefix)
		}

		spec, err := m.UpsertSpecialty(servicecatalog.Specialty{
			TenantId:    tenantID,
			Prefix:      cs.Prefix,
			Name:        cs.Name,
			Position:    int64(i + 1),
			IsPublished: true,
		})
		if err != nil {
			return data, fmt.Err("seed: UpsertSpecialty", cs.Prefix, err)
		}
		data.Specialties = append(data.Specialties, spec)
	}

	type itemSpec struct {
		prefix string
		sku   string
		name  string
		price float64
	}

	itemsToCreate := []itemSpec{
		{prefix: "md", sku: "MD001", name: "Consulta Medicina General", price: 25000},
		{prefix: "tr", sku: "TR001", name: "Consulta Traumatología", price: 35000},
		{prefix: "ec", sku: "EC001", name: "Ecografía Abdominal", price: 40000},
	}

	for _, specDef := range itemsToCreate {
		var specID string
		found := false
		for _, spec := range data.Specialties {
			if spec.Prefix == specDef.prefix {
				specID = spec.Id
				found = true
				break
			}
		}
		if !found {
			return data, fmt.Err("seed: specialty id not found", specDef.prefix)
		}

		createdItem, err := m.CreateService(servicecatalog.Service{
			TenantId:    tenantID,
			SpecialtyId: specID,
			Sku:         specDef.sku,
			Name:        specDef.name,
			Price:       specDef.price,
			Currency:    currencyCLP,
			IsActive:    true,
			TaxTreatment: servicecatalog.TaxExempt,
		})
		if err != nil {
			return data, fmt.Err("seed: CreateService", specDef.sku, err)
		}
		data.Items = append(data.Items, createdItem)
	}

	return data, nil
}
