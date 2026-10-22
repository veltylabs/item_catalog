package tests

import (
	"testing"

	servicecatalog "github.com/veltylabs/service_catalog"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
)

func TestCurrencyValidation(t *testing.T) {
	// C4: Invalid deps
	db := orm.New(mem.New())
	_, err := servicecatalog.New(db, servicecatalog.Deps{IDs: &MockIDGen{}, Currencies: []string{"clp"}})
	if err == nil || err.Error() != "service_catalog: Deps.Currencies has an invalid code" {
		t.Errorf("expected invalid code error for 'clp', got %v", err)
	}

	// Setup with nil currencies
	module, err := servicecatalog.New(db, servicecatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}

	spec, err := module.UpsertSpecialty(servicecatalog.Specialty{
		TenantId: "t1", Prefix: "aa", Name: "A",
	})
	if err != nil {
		t.Fatal(err)
	}

	// C1: Nil currencies, create in CLP -> success
	srv1, err := module.CreateService(servicecatalog.Service{
		TenantId:    "t1",
		SpecialtyId: spec.Id,
		Sku:         "AA01",
		Name:        "Service 1",
		Price:       100,
		Currency:    "CLP",
		IsActive:    true,
		TaxTreatment: "EXEMPT",
	})
	if err != nil {
		t.Errorf("expected success creating with CLP, got %v", err)
	}

	// C2: Nil currencies, create in USD -> fail
	_, err = module.CreateService(servicecatalog.Service{
		TenantId:    "t1",
		SpecialtyId: spec.Id,
		Sku:         "AA02",
		Name:        "Service 2",
		Price:       100,
		Currency:    "USD",
		IsActive:    true,
		TaxTreatment: "EXEMPT",
	})
	if err != servicecatalog.ErrUnsupportedCurrency {
		t.Errorf("expected ErrUnsupportedCurrency creating with USD, got %v", err)
	}

	// C3: Setup with CLP, USD
	module2, err := servicecatalog.New(db, servicecatalog.Deps{IDs: &MockIDGen{}, Currencies: []string{"CLP", "USD"}})
	if err != nil {
		t.Fatal(err)
	}

	// Update service to USD -> success
	srv1.Currency = "USD"
	_, err = module2.UpdateService(srv1)
	if err != nil {
		t.Errorf("expected success updating to USD, got %v", err)
	}
}
