package servicecatalog

import (
	"webtyp.com/fmt"
)

type CanonicalSpecialty struct {
	Prefix string
	Name   string
}

// CanonicalSpecialties defines the list of standard specialties derived from SKU prefixes.
var CanonicalSpecialties = []CanonicalSpecialty{
	{Prefix: "md", Name: "Medicina General"},
	{Prefix: "do", Name: "Dental (Odontología + Ortodoncia)"},
	{Prefix: "tr", Name: "Traumatología"},
	{Prefix: "po", Name: "Podología"},
	{Prefix: "la", Name: "Laboratorio"},
	{Prefix: "ec", Name: "Ecografía"},
	{Prefix: "gi", Name: "Ginecología y Obstetricia"},
	{Prefix: "ca", Name: "Cardiología"},
	{Prefix: "ga", Name: "Gastroenterología"},
	{Prefix: "of", Name: "Oftalmología"},
	{Prefix: "ne", Name: "Neurología"},
	{Prefix: "ps", Name: "Psicología"},
	{Prefix: "de", Name: "Dermatología"},
	{Prefix: "ra", Name: "Radiología"},
}

type MigrationReport struct {
	MigratedCount int
	UnmatchedSKUs []string
}

// MigrateSpecialtiesAndServices seeds canonical specialties for the tenant if missing,
// and maps existing Services to the corresponding specialty_id based on SKU prefix (sku[:2]).
// Unmatched items are left unassigned and returned in MigrationReport.UnmatchedSKUs.
func (m *Module) MigrateSpecialtiesAndServices(tenantId string) (MigrationReport, error) {
	report := MigrationReport{}

	prefixToID := make([]fmt.KeyValue, 0, len(CanonicalSpecialties))

	findID := func(prefix string) (string, bool) {
		for _, kv := range prefixToID {
			if kv.Key == prefix {
				return kv.Value, true
			}
		}
		return "", false
	}

	for _, cs := range CanonicalSpecialties {
		existing, err := m.GetSpecialtyByPrefix(tenantId, cs.Prefix)
		if err == nil {
			prefixToID = append(prefixToID, fmt.KeyValue{Key: cs.Prefix, Value: existing.Id})
		} else if e, ok := err.(domainError); ok && e == ErrSpecialtyNotFound {
			spec := Specialty{
				TenantId:    tenantId,
				Prefix:      cs.Prefix,
				Name:        cs.Name,
				IsPublished: false,
			}
			created, err := m.UpsertSpecialty(spec)
			if err != nil {
				return report, fmt.Err("failed to seed specialty %s: %w", cs.Prefix, err)
			}
			prefixToID = append(prefixToID, fmt.KeyValue{Key: cs.Prefix, Value: created.Id})
		} else {
			return report, err
		}
	}

	items, err := m.ListServices(tenantId, ServiceFilter{})
	if err != nil {
		return report, err
	}

	for _, item := range items {
		if len(item.Sku) < 2 {
			report.UnmatchedSKUs = append(report.UnmatchedSKUs, item.Sku)
			continue
		}
		prefix := item.Sku[:2]
		specID, found := findID(prefix)
		if !found {
			report.UnmatchedSKUs = append(report.UnmatchedSKUs, item.Sku)
			continue
		}

		item.SpecialtyId = specID
		_, err := m.UpdateService(item)
		if err != nil {
			return report, fmt.Err("failed to update item %s: %w", item.Id, err)
		}
		report.MigratedCount++
	}

	return report, nil
}
