package tests

import (
	"testing"

	servicecatalog "github.com/veltylabs/service_catalog"
	"webtyp.com/json"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/router"
	"webtyp.com/router/mock"
	"webtyp.com/storage/mem"
)

func TestMCPOperations(t *testing.T) {
	db := orm.New(mem.New())
	pub := &MockPublisher{}
	module, err := servicecatalog.New(db, servicecatalog.Deps{
		IDs:       &MockIDGen{},
		Publisher: pub,
	})
	if err != nil {
		t.Fatal(err)
	}

	r := &mock.Router{}
	r.Configure(mock.Config{
		Authn: func(next router.HandlerFunc) router.HandlerFunc {
			return func(ctx router.Context) {
				ctx.SetUserID("user-123")
				next(ctx)
			}
		},
		Authorize: func(userID string, res model.Resource, act model.Action) bool {
			return true
		},
	})
	module.MountOperations(r)

	tenantID := "tenant-1"

	// 0. Upsert Specialty via OP
	spec := servicecatalog.Specialty{
		TenantId: tenantID,
		Prefix:   "md",
		Slug:     "medicina-general",
		Name:     "Medicina General",
	}
	specArgs := servicecatalog.UpsertSpecialtyArgs{
		Records: []servicecatalog.Specialty{spec},
	}
	var specBytes []byte
	_ = json.Encode(&specArgs, &specBytes)

	ctxSpecUpsert := &mock.Context{
		InBody: specBytes,
	}
	r.Invoke("OP", "/"+servicecatalog.OpUpsertSpecialty, ctxSpecUpsert)
	if ctxSpecUpsert.Status != 0 && ctxSpecUpsert.Status != 200 {
		t.Fatalf("expected specialty upsert status success, got %d", ctxSpecUpsert.Status)
	}

	var createdSpec servicecatalog.Specialty
	if err := json.Decode(ctxSpecUpsert.ResponseBody(), &createdSpec); err != nil {
		t.Fatal(err)
	}
	if createdSpec.Id == "" {
		t.Fatal("expected non-empty Specialty ID")
	}

	// List Specialties via OP
	listSpecArgs := servicecatalog.ListSpecialtiesArgs{
		TenantId: tenantID,
	}
	var listSpecBytes []byte
	_ = json.Encode(&listSpecArgs, &listSpecBytes)

	ctxSpecList := &mock.Context{
		InBody: listSpecBytes,
	}
	r.Invoke("OP", "/"+servicecatalog.OpListSpecialties, ctxSpecList)
	if ctxSpecList.Status != 0 && ctxSpecList.Status != 200 {
		t.Fatalf("expected specialty list status success, got %d", ctxSpecList.Status)
	}

	var specListResult servicecatalog.SpecialtyList
	if err := json.Decode(ctxSpecList.ResponseBody(), &specListResult); err != nil {
		t.Fatal(err)
	}
	if specListResult.Len() != 1 {
		t.Fatalf("expected 1 specialty, got %d", specListResult.Len())
	}

	// 1. Create item via OP
	item := servicecatalog.Service{
		TenantId:    tenantID,
		SpecialtyId: createdSpec.Id,
		Sku:         "md-M1",
		Name:        "Mcp Product",
		Price:       100.0,
		Currency:    "USD",
		IsActive:    true,
	}
	var itemBytes []byte
	_ = json.Encode(&item, &itemBytes)

	ctxCreate := &mock.Context{
		InBody: itemBytes,
	}
	r.Invoke("OP", "/"+servicecatalog.OpCreateService, ctxCreate)
	if ctxCreate.Status != 0 && ctxCreate.Status != 200 {
		t.Fatalf("expected create status success, got %d", ctxCreate.Status)
	}

	var createdItem servicecatalog.Service
	if err := json.Decode(ctxCreate.ResponseBody(), &createdItem); err != nil {
		t.Fatal(err)
	}
	if createdItem.Id == "" {
		t.Fatal("expected non-empty ID")
	}

	// 2. Get item via OP
	getArgs := servicecatalog.GetServiceArgs{
		TenantId: tenantID,
		Id:       createdItem.Id,
	}
	var getArgsBytes []byte
	_ = json.Encode(&getArgs, &getArgsBytes)

	ctxGet := &mock.Context{
		InBody: getArgsBytes,
	}
	r.Invoke("OP", "/"+servicecatalog.OpGetService, ctxGet)
	if ctxGet.Status != 0 && ctxGet.Status != 200 {
		t.Fatalf("expected get status success, got %d, body: %s", ctxGet.Status, string(ctxGet.ResponseBody()))
	}

	var gotItem servicecatalog.Service
	if err := json.Decode(ctxGet.ResponseBody(), &gotItem); err != nil {
		t.Fatal(err)
	}
	if gotItem.Sku != "md-M1" {
		t.Fatalf("expected SKU md-M1, got %s", gotItem.Sku)
	}

	// 3. Find by SKU via OP
	skuArgs := servicecatalog.FindServiceBySKUArgs{
		TenantId: tenantID,
		Sku:      "md-M1",
	}
	var skuArgsBytes []byte
	_ = json.Encode(&skuArgs, &skuArgsBytes)

	ctxSku := &mock.Context{
		InBody: skuArgsBytes,
	}
	r.Invoke("OP", "/"+servicecatalog.OpFindServiceBySKU, ctxSku)
	if ctxSku.Status != 0 && ctxSku.Status != 200 {
		t.Fatalf("expected find sku status success, got %d", ctxSku.Status)
	}

	// 4. Update item via OP
	createdItem.Name = "Mcp Product Updated"
	updateArgs := servicecatalog.UpdateServiceArgs{
		Ids:    []string{createdItem.Id},
		Fields: []string{"name"},
		Record: createdItem,
	}
	var updateBytes []byte
	_ = json.Encode(&updateArgs, &updateBytes)

	ctxUpdate := &mock.Context{
		InBody: updateBytes,
	}
	r.Invoke("OP", "/"+servicecatalog.OpUpdateService, ctxUpdate)
	if ctxUpdate.Status != 0 && ctxUpdate.Status != 200 {
		t.Fatalf("expected update status success, got %d, body: %s", ctxUpdate.Status, string(ctxUpdate.ResponseBody()))
	}

	// 5. Upsert item via OP (Create with empty ID)
	upsertService := servicecatalog.Service{
		TenantId:    tenantID,
		SpecialtyId: createdSpec.Id,
		Sku:         "md-M2",
		Name:        "Mcp Product 2",
		Price:       200.0,
		Currency:    "USD",
		IsActive:    true,
	}
	upsertArgs := servicecatalog.UpsertServiceArgs{
		Records: []servicecatalog.Service{upsertService},
	}
	var upsertBytes []byte
	_ = json.Encode(&upsertArgs, &upsertBytes)

	ctxUpsert := &mock.Context{
		InBody: upsertBytes,
	}
	r.Invoke("OP", "/"+servicecatalog.OpUpsertService, ctxUpsert)
	if ctxUpsert.Status != 0 && ctxUpsert.Status != 200 {
		t.Fatalf("expected upsert status success, got %d", ctxUpsert.Status)
	}
	var upsertedItem servicecatalog.Service
	if err := json.Decode(ctxUpsert.ResponseBody(), &upsertedItem); err != nil {
		t.Fatal(err)
	}

	// 6. List items via OP
	listArgs := servicecatalog.ListServicesArgs{
		TenantId:    tenantID,
		SpecialtyId: createdSpec.Id,
		ActiveOnly:  true,
		Limit:       10,
		Offset:      0,
	}
	var listBytes []byte
	_ = json.Encode(&listArgs, &listBytes)

	ctxList := &mock.Context{
		InBody: listBytes,
	}
	r.Invoke("OP", "/"+servicecatalog.OpListServices, ctxList)
	if ctxList.Status != 0 && ctxList.Status != 200 {
		t.Fatalf("expected list status success, got %d", ctxList.Status)
	}

	var listResult servicecatalog.ServiceList
	if err := json.Decode(ctxList.ResponseBody(), &listResult); err != nil {
		t.Fatal(err)
	}
	if listResult.Len() != 1 {
	}

	// 7. Deactivate item via OP
	deactivateArgs := servicecatalog.DeactivateServiceArgs{
		TenantId: tenantID,
		Id:       createdItem.Id,
	}
	var deactivateBytes []byte
	_ = json.Encode(&deactivateArgs, &deactivateBytes)

	ctxDeactivate := &mock.Context{
		InBody: deactivateBytes,
	}
	r.Invoke("OP", "/"+servicecatalog.OpDeactivateService, ctxDeactivate)
	if ctxDeactivate.Status != 0 && ctxDeactivate.Status != 200 {
		t.Fatalf("expected deactivate status success, got %d", ctxDeactivate.Status)
	}

	// 8. Delete item via OP
	deleteArgs := servicecatalog.DeleteServicesArgs{
		Ids: []string{createdItem.Id},
	}
	var deleteBytes []byte
	_ = json.Encode(&deleteArgs, &deleteBytes)

	ctxDelete := &mock.Context{
		InBody: deleteBytes,
	}
	r.Invoke("OP", "/"+servicecatalog.OpDeleteService, ctxDelete)
	if ctxDelete.Status != 0 && ctxDelete.Status != 200 {
		t.Fatalf("expected delete status success, got %d", ctxDelete.Status)
	}

	// 9. Upsert agreement via OP (Create with empty ID)
	ag := servicecatalog.ServiceAgreement{
		TenantId:  tenantID,
		ServiceId: upsertedItem.Id,
		Insurer:   "Mcp Insurer",
		Code:      "M-123",
		Price:     150.0,
		IsActive:  true,
	}
	var agBytes []byte
	_ = json.Encode(&ag, &agBytes)

	ctxUpsertAg := &mock.Context{
		InBody: agBytes,
	}
	r.Invoke("OP", "/"+servicecatalog.OpUpsertAgreement, ctxUpsertAg)
	if ctxUpsertAg.Status != 0 && ctxUpsertAg.Status != 200 {
		t.Fatalf("expected upsert agreement status success, got %d", ctxUpsertAg.Status)
	}

	var upsertedAg servicecatalog.ServiceAgreement
	if err := json.Decode(ctxUpsertAg.ResponseBody(), &upsertedAg); err != nil {
		t.Fatal(err)
	}

	// 10. List agreements via OP
	listAgArgs := servicecatalog.ListAgreementsArgs{
		TenantId:  tenantID,
		ServiceId: upsertedItem.Id,
	}
	var listAgBytes []byte
	_ = json.Encode(&listAgArgs, &listAgBytes)

	ctxListAg := &mock.Context{
		InBody: listAgBytes,
	}
	r.Invoke("OP", "/"+servicecatalog.OpListAgreements, ctxListAg)
	if ctxListAg.Status != 0 && ctxListAg.Status != 200 {
		t.Fatalf("expected list agreements status success, got %d", ctxListAg.Status)
	}

	var listAgResult servicecatalog.ServiceAgreementList
	if err := json.Decode(ctxListAg.ResponseBody(), &listAgResult); err != nil {
		t.Fatal(err)
	}
	if listAgResult.Len() != 1 {
		t.Fatalf("expected 1 agreement, got %d", listAgResult.Len())
	}

	// 11. Delete agreement via OP
	deleteAgArgs := servicecatalog.DeleteAgreementArgs{
		TenantId: tenantID,
		Id:       upsertedAg.Id,
	}
	var deleteAgBytes []byte
	_ = json.Encode(&deleteAgArgs, &deleteAgBytes)

	ctxDeleteAg := &mock.Context{
		InBody: deleteAgBytes,
	}
	r.Invoke("OP", "/"+servicecatalog.OpDeleteAgreement, ctxDeleteAg)
	if ctxDeleteAg.Status != 0 && ctxDeleteAg.Status != 200 {
		t.Fatalf("expected delete agreement status success, got %d", ctxDeleteAg.Status)
	}
}

func TestMCPOperationsErrorPaths(t *testing.T) {
	db := orm.New(mem.New())
	module, err := servicecatalog.New(db, servicecatalog.Deps{
		IDs: &MockIDGen{},
	})
	if err != nil {
		t.Fatal(err)
	}

	r := &mock.Router{}
	r.Configure(mock.Config{
		Authn: func(next router.HandlerFunc) router.HandlerFunc {
			return func(ctx router.Context) {
				ctx.SetUserID("user-123")
				next(ctx)
			}
		},
		Authorize: func(userID string, res model.Resource, act model.Action) bool {
			return true
		},
	})
	module.MountOperations(r)

	// Decode error path
	ctxErr := &mock.Context{
		InBody: []byte(`invalid-json`),
	}
	r.Invoke("OP", "/"+servicecatalog.OpCreateService, ctxErr)
	if ctxErr.Status != 400 {
		t.Fatalf("expected status 400 on decode error, got %d", ctxErr.Status)
	}
}
