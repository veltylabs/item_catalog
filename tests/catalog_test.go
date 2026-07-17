package tests

import (
	"testing"

	"github.com/tinywasm/model"
	"github.com/tinywasm/orm"
	"github.com/tinywasm/router/mock"
	"github.com/tinywasm/storage/mem"
	"github.com/tinywasm/view"
	"github.com/tinywasm/view/conformance"
	itemcatalog "github.com/veltylabs/item_catalog"
)

func TestCatalog(t *testing.T) {
	db := orm.New(mem.New())

	pub := &MockPublisher{}
	module, err := itemcatalog.New(db, itemcatalog.Deps{
		IDs:       &MockIDGen{},
		Publisher: pub,
	})
	if err != nil {
		t.Fatal(err)
	}

	tenantID := "tenant-1"

	// Test CreateItem
	item := itemcatalog.CatalogItem{
		TenantId: tenantID,
		Sku:      "SKU123",
		Name:     "Test Service",
		Type:     "S",
		Price:    10.5,
		Currency: "USD",
		IsActive: true,
	}

	created, err := module.CreateItem(item)
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}
	if created.Id == "" {
		t.Error("expected non-empty ID")
	}

	// Test duplicate SKU
	_, err = module.CreateItem(item)
	if err == nil {
		t.Error("expected error for duplicate SKU")
	}

	// Test FindBySKU
	found, err := module.FindBySKU(tenantID, "SKU123")
	if err != nil {
		t.Errorf("failed to find item by SKU: %v", err)
	}
	if found.Id != created.Id {
		t.Errorf("expected ID %s, got %s", created.Id, found.Id)
	}

	// Test GetItem
	found, err = module.GetItem(tenantID, created.Id)
	if err != nil {
		t.Errorf("failed to get item: %v", err)
	}
	if found.Sku != "SKU123" {
		t.Errorf("expected SKU SKU123, got %s", found.Sku)
	}

	// Test ServiceExists
	exists, err := module.ServiceExists(tenantID, created.Id)
	if err != nil || !exists {
		t.Errorf("expected ServiceExists to be true, got %v, err: %v", exists, err)
	}

	// Test UpdateItem
	created.Name = "Updated Name"
	updated, err := module.UpdateItem(created)
	if err != nil {
		t.Errorf("failed to update item: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("expected Name Updated Name, got %s", updated.Name)
	}

	// Test ListItems
	items, err := module.ListItems(tenantID, itemcatalog.ItemFilter{})
	if err != nil {
		t.Errorf("failed to list items: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	// Test DeactivateItem
	err = module.DeactivateItem(tenantID, created.Id)
	if err != nil {
		t.Errorf("failed to deactivate item: %v", err)
	}
	deactivated, _ := module.GetItem(tenantID, created.Id)
	if deactivated.IsActive {
		t.Error("expected item to be inactive")
	}

	// Test ServiceExists for inactive
	exists, err = module.ServiceExists(tenantID, created.Id)
	if err != nil || exists {
		t.Errorf("expected ServiceExists to be false for inactive, got %v", exists)
	}

	// Test DeleteItem
	err = module.DeleteItem(tenantID, created.Id)
	if err != nil {
		t.Errorf("failed to delete item: %v", err)
	}
	_, err = module.GetItem(tenantID, created.Id)
	if err == nil {
		t.Error("expected error getting deleted item")
	}

	// Check publisher events
	// We expect 4 events now: create, update, deactivate, delete
	if len(pub.Events) != 4 {
		t.Errorf("expected 4 events, got %d", len(pub.Events))
	}
}

func TestAgreements(t *testing.T) {
	db := orm.New(mem.New())

	pub := &MockPublisher{}
	module, err := itemcatalog.New(db, itemcatalog.Deps{
		IDs:       &MockIDGen{},
		Publisher: pub,
	})
	if err != nil {
		t.Fatal(err)
	}

	tenantID := "tenant-1"
	itemID := "item-abc"

	// 1. Test UpsertAgreement - Create (Id == "")
	ag := itemcatalog.Agreement{
		TenantId:      tenantID,
		CatalogItemId: itemID,
		Insurer:       "FONASA",
		Code:          "F-12345",
		Price:         8500.0,
		IsActive:      true,
	}

	created, err := module.UpsertAgreement(ag)
	if err != nil {
		t.Fatalf("failed to create agreement: %v", err)
	}
	if created.Id == "" {
		t.Error("expected non-empty ID for created agreement")
	}
	if created.UpdatedAt == 0 {
		t.Error("expected UpdatedAt to be set")
	}
	if len(pub.Events) != 1 || pub.Events[0].Topic != "catalog.agreement.created" {
		t.Errorf("expected catalog.agreement.created event, got %v", pub.Events)
	}

	// 2. Test ListAgreements - Filter by Tenant and Item ID
	list, err := module.ListAgreements(tenantID, itemID)
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
	emptyList, err := module.ListAgreements(tenantID, "non-existent")
	if err != nil {
		t.Fatalf("failed to list agreements: %v", err)
	}
	if len(emptyList) != 0 {
		t.Errorf("expected 0 agreements, got %d", len(emptyList))
	}

	// 3. Test UpsertAgreement - Update (Id != "")
	created.Price = 9000.0
	updated, err := module.UpsertAgreement(created)
	if err != nil {
		t.Fatalf("failed to update agreement: %v", err)
	}
	if updated.Price != 9000.0 {
		t.Errorf("expected updated price to be 9000.0, got %f", updated.Price)
	}
	if len(pub.Events) != 2 || pub.Events[1].Topic != "catalog.agreement.updated" {
		t.Errorf("expected catalog.agreement.updated event, got %v", pub.Events)
	}

	// 4. Test DeleteAgreement
	err = module.DeleteAgreement(tenantID, updated.Id)
	if err != nil {
		t.Fatalf("failed to delete agreement: %v", err)
	}
	listAfterDelete, _ := module.ListAgreements(tenantID, itemID)
	if len(listAfterDelete) != 0 {
		t.Errorf("expected 0 agreements after deletion, got %d", len(listAfterDelete))
	}
	if len(pub.Events) != 3 || pub.Events[2].Topic != "catalog.agreement.deleted" {
		t.Errorf("expected catalog.agreement.deleted event, got %v", pub.Events)
	}
}

func TestModule_MountOpsAndView(t *testing.T) {
	db := orm.New(mem.New())
	pub := &MockPublisher{}
	module, err := itemcatalog.New(db, itemcatalog.Deps{IDs: &MockIDGen{}, Publisher: pub})
	if err != nil {
		t.Fatal(err)
	}

	r := &mock.Router{}
	module.MountOps(r)

	infos := r.Routes()
	var found bool
	for _, i := range infos {
		if i.Path == itemcatalog.OpUpsertItem || i.Path == "/"+itemcatalog.OpUpsertItem { // Op registers as Synthetic method "OP" and path "/"+name
			found = true
			if i.Resource != "catalog_item" || i.Action != model.Create {
				t.Errorf("RBAC mismatch for %s: %+v", itemcatalog.OpUpsertItem, i)
			}
		}
	}
	if !found {
		t.Fatalf("MountOps did not register %s", itemcatalog.OpUpsertItem)
	}

	caller := &mock.Caller{}
	pres := itemcatalog.NewView(caller)
	if pres.Title() == "" {
		t.Error("expected a non-empty view title")
	}
}

func copyFields(src, dst model.Model) {
	if src == nil || src.IsNil() || dst == nil || dst.IsNil() {
		return
	}
	srcFields := src.Schema()
	srcPointers := src.Pointers()
	dstFields := dst.Schema()
	dstPointers := dst.Pointers()

	for i, sf := range srcFields {
		for j, df := range dstFields {
			if sf.Name == df.Name {
				switch sVal := srcPointers[i].(type) {
				case *string:
					if dVal, ok := dstPointers[j].(*string); ok {
						*dVal = *sVal
					}
				case *float64:
					if dVal, ok := dstPointers[j].(*float64); ok {
						*dVal = *sVal
					}
				case *bool:
					if dVal, ok := dstPointers[j].(*bool); ok {
						*dVal = *sVal
					}
				case *int:
					if dVal, ok := dstPointers[j].(*int); ok {
						*dVal = *sVal
					}
				case *int64:
					if dVal, ok := dstPointers[j].(*int64); ok {
						*dVal = *sVal
					}
				}
			}
		}
	}
}

func TestConformance(t *testing.T) {
	conformance.Run(t, conformance.Factory{
		New: func(t *testing.T, p view.Presenter) conformance.Driver {
			return conformance.Driver{
				Mount: func() {
					_ = p.Reload()
				},
				Labels: func() []string {
					items := p.Items()
					labels := make([]string, len(items))
					for i, it := range rangeSlice(items) {
						labels[i] = it.Label
					}
					return labels
				},
				Select: func(id string) {
					m := p.Select(id)
					copyFields(m, p.Record())
				},
				SetField: func(name, value string) {
					rec := p.Record()
					if rec == nil || rec.IsNil() {
						return
					}
					fields := rec.Schema()
					pointers := rec.Pointers()
					for i, f := range fields {
						if f.Name == name {
							ptr := pointers[i]
							if sPtr, ok := ptr.(*string); ok {
								*sPtr = value
							}
						}
					}
				},
				Save: func() {
					_ = p.Save(p.Record())
				},
				Delete: func() {
					_ = p.Delete(p.Selected())
				},
			}
		},
	})
}

// helper to iterate safely without map or standard loops if we want to avoid any banned structure
func rangeSlice(items []view.Item) []view.Item {
	return items
}
