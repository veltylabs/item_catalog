package tests

import (
	"encoding/json"
	"testing"

	"github.com/tinywasm/model"
	"github.com/tinywasm/orm"
	"github.com/tinywasm/router"
	"github.com/tinywasm/router/mock"
	"github.com/tinywasm/storage/mem"
	itemcatalog "github.com/veltylabs/item_catalog"
)

func TestMCPOperations(t *testing.T) {
	db := orm.New(mem.New())
	pub := &MockPublisher{}
	module, err := itemcatalog.New(db, itemcatalog.Deps{
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
	module.MountOps(r)

	tenantID := "tenant-1"

	// 1. Create item via OP
	item := struct {
		TenantId string  `json:"tenant_id"`
		Sku      string  `json:"sku"`
		Name     string  `json:"name"`
		Type     string  `json:"type"`
		Price    float64 `json:"price"`
		Currency string  `json:"currency"`
		IsActive bool    `json:"is_active"`
	}{
		TenantId: tenantID,
		Sku:      "SKU-M1",
		Name:     "Mcp Product",
		Type:     "P",
		Price:    100.0,
		Currency: "USD",
		IsActive: true,
	}
	itemBytes, _ := json.Marshal(item)

	ctxCreate := &mock.Context{
		InBody: itemBytes,
	}
	r.Invoke("OP", "/"+itemcatalog.OpCreateItem, ctxCreate)
	if ctxCreate.Status != 0 && ctxCreate.Status != 200 {
		t.Fatalf("expected create status success, got %d", ctxCreate.Status)
	}

	var createdItem itemcatalog.CatalogItem
	if err := json.Unmarshal(ctxCreate.ResponseBody(), &createdItem); err != nil {
		t.Fatal(err)
	}
	if createdItem.Id == "" {
		t.Fatal("expected non-empty ID")
	}

	// 2. Get item via OP
	getArgs := struct {
		TenantId string `json:"tenant_id"`
		Id       string `json:"id"`
	}{
		TenantId: tenantID,
		Id:       createdItem.Id,
	}
	getArgsBytes, _ := json.Marshal(getArgs)

	ctxGet := &mock.Context{
		InBody: getArgsBytes,
	}
	r.Invoke("OP", "/"+itemcatalog.OpGetItem, ctxGet)
	if ctxGet.Status != 0 && ctxGet.Status != 200 {
		t.Fatalf("expected get status success, got %d, body: %s", ctxGet.Status, string(ctxGet.ResponseBody()))
	}

	var gotItem itemcatalog.CatalogItem
	if err := json.Unmarshal(ctxGet.ResponseBody(), &gotItem); err != nil {
		t.Fatal(err)
	}
	if gotItem.Sku != "SKU-M1" {
		t.Fatalf("expected SKU SKU-M1, got %s", gotItem.Sku)
	}

	// 3. Find by SKU via OP
	skuArgs := struct {
		TenantId string `json:"tenant_id"`
		Sku      string `json:"sku"`
	}{
		TenantId: tenantID,
		Sku:      "SKU-M1",
	}
	skuArgsBytes, _ := json.Marshal(skuArgs)

	ctxSku := &mock.Context{
		InBody: skuArgsBytes,
	}
	r.Invoke("OP", "/"+itemcatalog.OpFindItemBySKU, ctxSku)
	if ctxSku.Status != 0 && ctxSku.Status != 200 {
		t.Fatalf("expected find sku status success, got %d", ctxSku.Status)
	}

	// 4. Update item via OP
	updateItem := struct {
		Id       string  `json:"id"`
		TenantId string  `json:"tenant_id"`
		Sku      string  `json:"sku"`
		Name     string  `json:"name"`
		Type     string  `json:"type"`
		Price    float64 `json:"price"`
		Currency string  `json:"currency"`
		IsActive bool    `json:"is_active"`
	}{
		Id:       createdItem.Id,
		TenantId: tenantID,
		Sku:      "SKU-M1",
		Name:     "Mcp Product Updated",
		Type:     "P",
		Price:    100.0,
		Currency: "USD",
		IsActive: true,
	}
	updateBytes, _ := json.Marshal(updateItem)

	ctxUpdate := &mock.Context{
		InBody: updateBytes,
	}
	r.Invoke("OP", "/"+itemcatalog.OpUpdateItem, ctxUpdate)
	if ctxUpdate.Status != 0 && ctxUpdate.Status != 200 {
		t.Fatalf("expected update status success, got %d", ctxUpdate.Status)
	}

	// 5. Upsert item via OP (Create with empty ID)
	upsertItem := struct {
		TenantId string  `json:"tenant_id"`
		Sku      string  `json:"sku"`
		Name     string  `json:"name"`
		Type     string  `json:"type"`
		Price    float64 `json:"price"`
		Currency string  `json:"currency"`
		IsActive bool    `json:"is_active"`
	}{
		TenantId: tenantID,
		Sku:      "SKU-M2",
		Name:     "Mcp Product 2",
		Type:     "S",
		Price:    200.0,
		Currency: "USD",
		IsActive: true,
	}
	upsertBytes, _ := json.Marshal(upsertItem)

	ctxUpsert := &mock.Context{
		InBody: upsertBytes,
	}
	r.Invoke("OP", "/"+itemcatalog.OpUpsertItem, ctxUpsert)
	if ctxUpsert.Status != 0 && ctxUpsert.Status != 200 {
		t.Fatalf("expected upsert status success, got %d", ctxUpsert.Status)
	}

	var upsertedItem itemcatalog.CatalogItem
	if err := json.Unmarshal(ctxUpsert.ResponseBody(), &upsertedItem); err != nil {
		t.Fatal(err)
	}

	// 6. List items via OP
	listArgs := struct {
		TenantId   string `json:"tenant_id"`
		Type       string `json:"type"`
		ActiveOnly bool   `json:"active_only"`
		Limit      int64  `json:"limit"`
		Offset     int64  `json:"offset"`
	}{
		TenantId:   tenantID,
		Type:       "S",
		ActiveOnly: true,
		Limit:      10,
		Offset:     0,
	}
	listBytes, _ := json.Marshal(listArgs)

	ctxList := &mock.Context{
		InBody: listBytes,
	}
	r.Invoke("OP", "/"+itemcatalog.OpListItems, ctxList)
	if ctxList.Status != 0 && ctxList.Status != 200 {
		t.Fatalf("expected list status success, got %d", ctxList.Status)
	}

	var listResult []*itemcatalog.CatalogItem
	if err := json.Unmarshal(ctxList.ResponseBody(), &listResult); err != nil {
		t.Fatal(err)
	}
	if len(listResult) != 1 {
		t.Fatalf("expected 1 item of type S, got %d", len(listResult))
	}

	// 7. Deactivate item via OP
	deactivateArgs := struct {
		TenantId string `json:"tenant_id"`
		Id       string `json:"id"`
	}{
		TenantId: tenantID,
		Id:       createdItem.Id,
	}
	deactivateBytes, _ := json.Marshal(deactivateArgs)

	ctxDeactivate := &mock.Context{
		InBody: deactivateBytes,
	}
	r.Invoke("OP", "/"+itemcatalog.OpDeactivateItem, ctxDeactivate)
	if ctxDeactivate.Status != 0 && ctxDeactivate.Status != 200 {
		t.Fatalf("expected deactivate status success, got %d", ctxDeactivate.Status)
	}

	// 8. Delete item via OP
	deleteArgs := struct {
		TenantId string `json:"tenant_id"`
		Id       string `json:"id"`
	}{
		TenantId: tenantID,
		Id:       createdItem.Id,
	}
	deleteBytes, _ := json.Marshal(deleteArgs)

	ctxDelete := &mock.Context{
		InBody: deleteBytes,
	}
	r.Invoke("OP", "/"+itemcatalog.OpDeleteItem, ctxDelete)
	if ctxDelete.Status != 0 && ctxDelete.Status != 200 {
		t.Fatalf("expected delete status success, got %d", ctxDelete.Status)
	}

	// 9. Upsert agreement via OP (Create with empty ID)
	ag := struct {
		TenantId      string  `json:"tenant_id"`
		CatalogItemId string  `json:"catalog_item_id"`
		Insurer       string  `json:"insurer"`
		Code          string  `json:"code"`
		Price         float64 `json:"price"`
		IsActive      bool    `json:"is_active"`
	}{
		TenantId:      tenantID,
		CatalogItemId: upsertedItem.Id,
		Insurer:       "Mcp Insurer",
		Code:          "M-123",
		Price:         150.0,
		IsActive:      true,
	}
	agBytes, _ := json.Marshal(ag)

	ctxUpsertAg := &mock.Context{
		InBody: agBytes,
	}
	r.Invoke("OP", "/"+itemcatalog.OpUpsertAgreement, ctxUpsertAg)
	if ctxUpsertAg.Status != 0 && ctxUpsertAg.Status != 200 {
		t.Fatalf("expected upsert agreement status success, got %d", ctxUpsertAg.Status)
	}

	var upsertedAg itemcatalog.Agreement
	if err := json.Unmarshal(ctxUpsertAg.ResponseBody(), &upsertedAg); err != nil {
		t.Fatal(err)
	}

	// 10. List agreements via OP
	listAgArgs := struct {
		TenantId      string `json:"tenant_id"`
		CatalogItemId string `json:"catalog_item_id"`
	}{
		TenantId:      tenantID,
		CatalogItemId: upsertedItem.Id,
	}
	listAgBytes, _ := json.Marshal(listAgArgs)

	ctxListAg := &mock.Context{
		InBody: listAgBytes,
	}
	r.Invoke("OP", "/"+itemcatalog.OpListAgreements, ctxListAg)
	if ctxListAg.Status != 0 && ctxListAg.Status != 200 {
		t.Fatalf("expected list agreements status success, got %d", ctxListAg.Status)
	}

	var listAgResult []*itemcatalog.Agreement
	if err := json.Unmarshal(ctxListAg.ResponseBody(), &listAgResult); err != nil {
		t.Fatal(err)
	}
	if len(listAgResult) != 1 {
		t.Fatalf("expected 1 agreement, got %d", len(listAgResult))
	}

	// 11. Delete agreement via OP
	deleteAgArgs := struct {
		TenantId string `json:"tenant_id"`
		Id       string `json:"id"`
	}{
		TenantId: tenantID,
		Id:       upsertedAg.Id,
	}
	deleteAgBytes, _ := json.Marshal(deleteAgArgs)

	ctxDeleteAg := &mock.Context{
		InBody: deleteAgBytes,
	}
	r.Invoke("OP", "/"+itemcatalog.OpDeleteAgreement, ctxDeleteAg)
	if ctxDeleteAg.Status != 0 && ctxDeleteAg.Status != 200 {
		t.Fatalf("expected delete agreement status success, got %d", ctxDeleteAg.Status)
	}
}

func TestMCPOperationsErrorPaths(t *testing.T) {
	db := orm.New(mem.New())
	module, err := itemcatalog.New(db, itemcatalog.Deps{
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
	module.MountOps(r)

	// Decode error path
	ctxErr := &mock.Context{
		InBody: []byte(`invalid-json`),
	}
	r.Invoke("OP", "/"+itemcatalog.OpCreateItem, ctxErr)
	if ctxErr.Status != 400 {
		t.Fatalf("expected status 400 on decode error, got %d", ctxErr.Status)
	}
}
