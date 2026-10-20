package tests

import (
	"testing"

	servicecatalog "github.com/veltylabs/service_catalog"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/router/mock"
	"webtyp.com/storage/mem"
)

func TestCatalog(t *testing.T) {
	db := orm.New(mem.New())

	pub := &MockPublisher{}
	module, err := servicecatalog.New(db, servicecatalog.Deps{
		IDs:       &MockIDGen{},
		Publisher: pub,
	})
	if err != nil {
		t.Fatal(err)
	}

	tenantID := "tenant-1"

	// Create specialty first
	spec, err := module.UpsertSpecialty(servicecatalog.Specialty{
		TenantId: tenantID,
		Prefix:   "md",
		Slug:     "medicina-general",
		Name:     "Medicina General",
	})
	if err != nil {
		t.Fatalf("failed to create specialty: %v", err)
	}

	// Test CreateService
	item := servicecatalog.Service{
		TenantId:    tenantID,
		SpecialtyId: spec.Id,
		Sku:         "md1234",
		Name:        "Test Service",
		Price:       10.5,
		Currency:    "USD",
		IsActive:    true,
	}

	created, err := module.CreateService(item)
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}
	if created.Id == "" {
		t.Error("expected non-empty ID")
	}

	// Test duplicate SKU
	_, err = module.CreateService(item)
	if err == nil {
		t.Error("expected error for duplicate SKU")
	}

	// Test FindBySKU
	found, err := module.FindBySKU(tenantID, "md1234")
	if err != nil {
		t.Errorf("failed to find item by SKU: %v", err)
	}
	if found.Id != created.Id {
		t.Errorf("expected ID %s, got %s", created.Id, found.Id)
	}

	// Test GetService
	found, err = module.GetService(tenantID, created.Id)
	if err != nil {
		t.Errorf("failed to get item: %v", err)
	}
	if found.Sku != "md1234" {
		t.Errorf("expected SKU md1234, got %s", found.Sku)
	}

	// Test ServiceExists
	exists, err := module.ServiceExists(tenantID, created.Id)
	if err != nil || !exists {
		t.Errorf("expected ServiceExists to be true, got %v, err: %v", exists, err)
	}

	// Test UpdateService
	created.Name = "Updated Name"
	updated, err := module.UpdateService(created)
	if err != nil {
		t.Errorf("failed to update item: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("expected Name Updated Name, got %s", updated.Name)
	}

	// Test ListServices
	items, err := module.ListServices(tenantID, servicecatalog.ServiceFilter{})
	if err != nil {
		t.Errorf("failed to list items: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	// Create second specialty and item to test filter isolation between two real specialties
	spec2, err := module.UpsertSpecialty(servicecatalog.Specialty{
		TenantId: tenantID,
		Prefix:   "of",
		Slug:     "oftalmologia",
		Name:     "Oftalmología",
	})
	if err != nil {
		t.Fatalf("failed to create second specialty: %v", err)
	}

	item2 := servicecatalog.Service{
		TenantId:    tenantID,
		SpecialtyId: spec2.Id,
		Sku:         "of1234",
		Name:        "Ophthalmology Exam",
		Price:       20.0,
		Currency:    "USD",
		IsActive:    true,
	}
	created2, err := module.CreateService(item2)
	if err != nil {
		t.Fatalf("failed to create item 2: %v", err)
	}

	// Test ListServices with filter matching spec1
	filteredItems, err := module.ListServices(tenantID, servicecatalog.ServiceFilter{
		SpecialtyId: spec.Id,
		ActiveOnly:  true,
		Limit:       5,
		Offset:      0,
	})
	if err != nil {
		t.Errorf("failed to list items with filters: %v", err)
	}
	if len(filteredItems) != 1 || filteredItems[0].Id != created.Id {
		t.Errorf("expected 1 item with specialty spec1 ID %s, got %d items", created.Id, len(filteredItems))
	}

	// Test ListServices with filter matching spec2
	filteredItems2, err := module.ListServices(tenantID, servicecatalog.ServiceFilter{
		SpecialtyId: spec2.Id,
	})
	if err != nil {
		t.Errorf("failed to list items with spec2 filter: %v", err)
	}
	if len(filteredItems2) != 1 || filteredItems2[0].Id != created2.Id {
		t.Errorf("expected 1 item with specialty spec2 ID %s, got %d items", created2.Id, len(filteredItems2))
	}

	// Clean up second item and specialty for rest of test flow
	if err := module.DeleteService(tenantID, created2.Id); err != nil {
		t.Fatalf("failed to delete item 2: %v", err)
	}
	if err := module.DeleteSpecialty(tenantID, spec2.Id); err != nil {
		t.Fatalf("failed to delete specialty 2: %v", err)
	}

	// Test DeleteSpecialty while in use -> should fail with ErrSpecialtyInUse
	err = module.DeleteSpecialty(tenantID, spec.Id)
	if err == nil || err.Error() != servicecatalog.ErrSpecialtyInUse.Error() {
		t.Errorf("expected ErrSpecialtyInUse, got %v", err)
	}

	// Test DeactivateService
	err = module.DeactivateService(tenantID, created.Id)
	if err != nil {
		t.Errorf("failed to deactivate item: %v", err)
	}
	deactivated, _ := module.GetService(tenantID, created.Id)
	if deactivated.IsActive {
		t.Error("expected item to be inactive")
	}

	// Test ServiceExists for inactive
	exists, err = module.ServiceExists(tenantID, created.Id)
	if err != nil || exists {
		t.Errorf("expected ServiceExists to be false for inactive, got %v", exists)
	}

	// Test DeleteService
	err = module.DeleteService(tenantID, created.Id)
	if err != nil {
		t.Errorf("failed to delete item: %v", err)
	}
	_, err = module.GetService(tenantID, created.Id)
	if err == nil {
		t.Error("expected error getting deleted item")
	}

	// Test DeleteSpecialty now that item is deleted -> should succeed
	err = module.DeleteSpecialty(tenantID, spec.Id)
	if err != nil {
		t.Errorf("failed to delete specialty after item deletion: %v", err)
	}
}

func TestSpecialtyUniquenessAndDefaults(t *testing.T) {
	db := orm.New(mem.New())
	module, err := servicecatalog.New(db, servicecatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}

	tenant1 := "tenant-1"
	tenant2 := "tenant-2"

	// 1. Create specialty in tenant 1
	spec1, err := module.UpsertSpecialty(servicecatalog.Specialty{
		TenantId: tenant1,
		Prefix:   "of",
		Slug:     "oftalmologia",
		Name:     "Oftalmología",
	})
	if err != nil {
		t.Fatalf("failed to create specialty: %v", err)
	}

	// Verify is_published defaults to false
	if spec1.IsPublished {
		t.Error("expected is_published to default to false")
	}

	// 2. Duplicate prefix in tenant 1 -> rejected
	_, err = module.UpsertSpecialty(servicecatalog.Specialty{
		TenantId: tenant1,
		Prefix:   "of",
		Slug:     "oftalmologia-2",
		Name:     "Oftalmología Segunda",
	})
	if err == nil || err.Error() != servicecatalog.ErrSpecialtyPrefixExists.Error() {
		t.Errorf("expected ErrSpecialtyPrefixExists, got %v", err)
	}

	// 3. Duplicate slug in tenant 1 -> rejected
	_, err = module.UpsertSpecialty(servicecatalog.Specialty{
		TenantId: tenant1,
		Prefix:   "ox",
		Slug:     "oftalmologia",
		Name:     "Oftalmología Segunda",
	})
	if err == nil || err.Error() != servicecatalog.ErrSpecialtySlugExists.Error() {
		t.Errorf("expected ErrSpecialtySlugExists, got %v", err)
	}

	// 4. Same prefix in tenant 2 -> allowed
	spec2, err := module.UpsertSpecialty(servicecatalog.Specialty{
		TenantId: tenant2,
		Prefix:   "of",
		Slug:     "oftalmologia",
		Name:     "Oftalmología",
	})
	if err != nil {
		t.Fatalf("expected duplicate prefix across different tenants to be allowed, got: %v", err)
	}
	if spec2.Id == spec1.Id {
		t.Error("expected different IDs for different tenant specialties")
	}
}

func TestMigration(t *testing.T) {
	db := orm.New(mem.New())
	module, err := servicecatalog.New(db, servicecatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}

	tenantID := "tenant-1"

	// Create item with canonical prefix "mdcong" directly in DB (simulating legacy data before migration)
	item1 := servicecatalog.Service{
		Id:       "unmigrated-1",
		TenantId: tenantID,
		Sku:      "mdcong",
		Name:     "Congreso Medico",
		Price:    100.0,
		Currency: "USD",
		IsActive: true,
	}
	if err := db.Create(&item1); err != nil {
		t.Fatalf("failed to create unmigrated item 1: %v", err)
	}

	// Create item with unknown prefix "zz1234" directly in DB
	item2 := servicecatalog.Service{
		Id:       "unmigrated-2",
		TenantId: tenantID,
		Sku:      "zz1234",
		Name:     "Unknown Item",
		Price:    50.0,
		Currency: "USD",
		IsActive: true,
	}
	if err := db.Create(&item2); err != nil {
		t.Fatalf("failed to create unmigrated item 2: %v", err)
	}

	// Run migration
	report, err := module.MigrateSpecialtiesAndServices(tenantID)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	if report.MigratedCount != 1 {
		t.Errorf("expected 1 migrated item, got %d", report.MigratedCount)
	}
	if len(report.UnmatchedSKUs) != 1 || report.UnmatchedSKUs[0] != "zz1234" {
		t.Errorf("expected [zz1234] in unmatched SKUs, got %v", report.UnmatchedSKUs)
	}

	// Verify item 1 has specialty_id set to "md" specialty ID
	migrated1, err := module.GetService(tenantID, "unmigrated-1")
	if err != nil {
		t.Fatal(err)
	}
	if migrated1.SpecialtyId == "" {
		t.Error("expected item 1 specialty_id to be populated")
	}

	specMD, err := module.GetSpecialtyByPrefix(tenantID, "md")
	if err != nil {
		t.Fatal(err)
	}
	if migrated1.SpecialtyId != specMD.Id {
		t.Errorf("expected specialty_id %s, got %s", specMD.Id, migrated1.SpecialtyId)
	}

	// Verify item 2 specialty_id is still empty
	migrated2, err := module.GetService(tenantID, "unmigrated-2")
	if err != nil {
		t.Fatal(err)
	}
	if migrated2.SpecialtyId != "" {
		t.Errorf("expected item 2 specialty_id to be empty, got %s", migrated2.SpecialtyId)
	}
}

func TestServiceAgreements(t *testing.T) {
	db := orm.New(mem.New())

	pub := &MockPublisher{}
	module, err := servicecatalog.New(db, servicecatalog.Deps{
		IDs:       &MockIDGen{},
		Publisher: pub,
	})
	if err != nil {
		t.Fatal(err)
	}

	tenantID := "tenant-1"

	spec, err := module.UpsertSpecialty(servicecatalog.Specialty{
		TenantId: tenantID,
		Prefix:   "md",
		Slug:     "medicina-general",
		Name:     "Medicina General",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create Service first so foreign key validation is satisfied
	goodItem := servicecatalog.Service{
		TenantId:    tenantID,
		SpecialtyId: spec.Id,
		Sku:         "md-AG1",
		Name:        "ServiceAgreement Service",
		Price:       100.0,
		Currency:    "USD",
		IsActive:    true,
	}
	createdItem, err := module.CreateService(goodItem)
	if err != nil {
		t.Fatalf("failed to create parent item: %v", err)
	}
	itemID := createdItem.Id

	// Clear events before testing agreements
	pub.Events = nil

	// 1. Test UpsertServiceAgreement - Create (Id == "")
	ag := servicecatalog.ServiceAgreement{
		TenantId:  tenantID,
		ServiceId: itemID,
		Insurer:   "FONASA",
		Code:      "F-12345",
		Price:     8500.0,
		IsActive:  true,
	}

	created, err := module.UpsertServiceAgreement(ag)
	if err != nil {
		t.Fatalf("failed to create agreement: %v", err)
	}
	if created.Id == "" {
		t.Error("expected non-empty ID for created agreement")
	}
	if created.UpdatedAt == 0 {
		t.Error("expected UpdatedAt to be set")
	}
	if len(pub.Events) != 1 || pub.Events[0].Topic != "catalog.service_agreement.created" {
		t.Errorf("expected catalog.service_agreement.created event, got %v", pub.Events)
	}

	// 2. Test ListServiceAgreements - Filter by Tenant and Item ID
	list, err := module.ListServiceAgreements(tenantID, itemID)
	if err != nil {
		t.Fatalf("failed to list agreements: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 agreement, got %d", len(list))
	}
	if list[0].Id != created.Id {
		t.Errorf("expected ID %s, got %s", created.Id, list[0].Id)
	}

	// Filter by non-existent item id
	emptyList, err := module.ListServiceAgreements(tenantID, "non-existent")
	if err != nil {
		t.Fatalf("failed to list agreements: %v", err)
	}
	if len(emptyList) != 0 {
		t.Errorf("expected 0 agreements, got %d", len(emptyList))
	}

	// 3. Test UpsertServiceAgreement - Update (Id != "")
	created.Price = 9000.0
	updated, err := module.UpsertServiceAgreement(created)
	if err != nil {
		t.Fatalf("failed to update agreement: %v", err)
	}
	if updated.Price != 9000.0 {
		t.Errorf("expected updated price to be 9000.0, got %f", updated.Price)
	}
	if len(pub.Events) != 2 || pub.Events[1].Topic != "catalog.service_agreement.updated" {
		t.Errorf("expected catalog.service_agreement.updated event, got %v", pub.Events)
	}

	// 4. Test DeleteServiceAgreement
	err = module.DeleteServiceAgreement(tenantID, updated.Id)
	if err != nil {
		t.Fatalf("failed to delete agreement: %v", err)
	}
	listAfterDelete, _ := module.ListServiceAgreements(tenantID, itemID)
	if len(listAfterDelete) != 0 {
		t.Errorf("expected 0 agreements after deletion, got %d", len(listAfterDelete))
	}
	if len(pub.Events) != 3 || pub.Events[2].Topic != "catalog.service_agreement.deleted" {
		t.Errorf("expected catalog.service_agreement.deleted event, got %v", pub.Events)
	}
}

func TestModule_MountOperationsAndView(t *testing.T) {
	db := orm.New(mem.New())
	pub := &MockPublisher{}
	module, err := servicecatalog.New(db, servicecatalog.Deps{IDs: &MockIDGen{}, Publisher: pub})
	if err != nil {
		t.Fatal(err)
	}

	r := &mock.Router{}
	module.MountOperations(r)

	infos := r.Routes()
	var found bool
	for _, i := range infos {
		if i.Path == servicecatalog.OpUpsertService || i.Path == "/"+servicecatalog.OpUpsertService {
			found = true
			if i.Resource != "service" || i.Action != (model.Create|model.Update) {
				t.Errorf("RBAC mismatch for %s: %+v", servicecatalog.OpUpsertService, i)
			}
		}
	}
	if !found {
		t.Fatalf("MountOperations did not register %s", servicecatalog.OpUpsertService)
	}

	caller := &mock.Caller{}
	pres := servicecatalog.NewView(caller)
	if pres.Title() == "" {
		t.Error("expected a non-empty view title")
	}

	specPres := servicecatalog.NewSpecialtyView(caller)
	if specPres.Title() == "" {
		t.Error("expected a non-empty specialty view title")
	}
}

func TestTaxTreatmentValidation(t *testing.T) {
	db := orm.New(mem.New())
	module, err := servicecatalog.New(db, servicecatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}

	spec, err := module.UpsertSpecialty(servicecatalog.Specialty{
		TenantId: "t1", Prefix: "aa", Slug: "a", Name: "A",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Unknown tax treatment should fail
	badService := servicecatalog.Service{
		TenantId:     "t1",
		SpecialtyId:  spec.Id,
		Sku:          "sku-1",
		Name:         "S1",
		Price:        100,
		Currency:     "USD",
		IsActive:     true,
		TaxTreatment: "UNKNOWN",
	}
	err = badService.Validate(model.ActionCreate)
	if err == nil {
		t.Error("expected error for unknown tax treatment")
	}

	// Valid tax treatments should succeed
	s2 := servicecatalog.Service{
		TenantId:     "t1",
		SpecialtyId:  spec.Id,
		Sku:          "sku-2",
		Name:         "S2",
		Price:        100,
		Currency:     "USD",
		IsActive:     true,
		TaxTreatment: servicecatalog.TaxTaxed,
	}
	if _, err := module.CreateService(s2); err != nil {
		t.Errorf("expected success for TAXED, got %v", err)
	}
}
