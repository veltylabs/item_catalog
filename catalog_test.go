package itemcatalog

import (
	"encoding/json"
	"testing"

	"github.com/tinywasm/context"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/sqlite"
)

func TestCatalog(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.Close(db)

	ui := &MockUI{}
	pub := &MockPublisher{}
	module, err := New(db, Deps{
		UI:        ui,
		Publisher: pub,
	})
	if err != nil {
		t.Fatal(err)
	}

	tenantID := "tenant-1"

	// Test CreateItem
	item := CatalogItem{
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
	items, err := module.ListItems(tenantID, ItemFilter{})
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

	// Test UI methods
	// Reactivate item first to test RenderList (activeOnly=true)
	deactivated.IsActive = true
	module.UpdateItem(deactivated)

	res := module.RenderList(tenantID, "")
	if res != "List: 1 items" {
		t.Errorf("unexpected RenderList result: %s", res)
	}
	if !ui.RenderItemListCalled {
		t.Error("expected UI.RenderItemList to be called")
	}

	// Test MCP tools
	ctx := context.Background()
	tools := module.Tools()
	if len(tools) == 0 {
		t.Fatal("expected tools to be defined")
	}

	// Find get_catalog_item tool
	var getTool mcp.Tool
	for _, tool := range tools {
		if tool.Name == OpGetItem {
			getTool = tool
			break
		}
	}

	args, _ := json.Marshal(map[string]string{
		"tenant_id": tenantID,
		"id":        created.Id,
	})

	// Prepare MCP Request using JSON
	var mcpReq mcp.Request
	// In callToolParams, arguments is a string containing JSON
	argStr, _ := json.Marshal(string(args))
	err = json.Unmarshal([]byte(`{"params":{"name":"`+OpGetItem+`","arguments":`+string(argStr)+`}}`), &mcpReq)
	if err != nil {
		t.Fatalf("failed to unmarshal mock request: %v", err)
	}

	mcpRes, err := getTool.Execute(ctx, mcpReq)
	if err != nil {
		t.Fatalf("failed to execute MCP tool: %v", err)
	}
	if mcpRes.IsError {
		t.Fatalf("MCP tool returned error: %s", mcpRes.Content)
	}

	// The Content is a JSON string of textContent if it was created via mcp.Text
	var tcs []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(mcpRes.Content), &tcs); err != nil {
		t.Fatalf("failed to unmarshal MCP result content: %v. Content was: %s", err, mcpRes.Content)
	}
	if len(tcs) == 0 {
		t.Fatalf("MCP result content is empty. Content was: %s", mcpRes.Content)
	}
	tc := tcs[0]

	var mcpItem CatalogItem
	if err := json.Unmarshal([]byte(tc.Text), &mcpItem); err != nil {
		t.Fatalf("failed to unmarshal item from MCP text: %v. Text was: %s", err, tc.Text)
	}
	if mcpItem.Id != created.Id {
		t.Errorf("expected ID %s, got %s", created.Id, mcpItem.Id)
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
	// We expect 5 events now: create, update, deactivate, update (reactivate), delete
	if len(pub.Events) != 5 {
		t.Errorf("expected 5 events, got %d", len(pub.Events))
	}
}

func TestAgreements(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.Close(db)

	pub := &MockPublisher{}
	module, err := New(db, Deps{
		Publisher: pub,
	})
	if err != nil {
		t.Fatal(err)
	}

	tenantID := "tenant-1"
	itemID := "item-abc"

	// 1. Test UpsertAgreement - Create (Id == "")
	ag := Agreement{
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
	if len(pub.Events) != 1 || pub.Events[0] != "catalog.agreement.created" {
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
	if len(pub.Events) != 2 || pub.Events[1] != "catalog.agreement.updated" {
		t.Errorf("expected catalog.agreement.updated event, got %v", pub.Events)
	}

	// 4. Test MCP Tools for Agreements
	ctx := context.Background()
	tools := module.Tools()

	// Find the upsert_agreement tool
	var upsertTool mcp.Tool
	for _, tool := range tools {
		if tool.Name == OpUpsertAgreement {
			upsertTool = tool
			break
		}
	}

	// Call upsert_agreement to create a second agreement via MCP
	ag2 := map[string]any{
		"tenant_id":       tenantID,
		"catalog_item_id": itemID,
		"insurer":         "Isapre Colmena",
		"code":            "C-555",
		"price":           12000.0,
		"is_active":       true,
	}
	argsBytes, _ := json.Marshal(ag2)
	var mcpReq mcp.Request
	argsStr, _ := json.Marshal(string(argsBytes))
	err = json.Unmarshal([]byte(`{"params":{"name":"`+OpUpsertAgreement+`","arguments":`+string(argsStr)+`}}`), &mcpReq)
	if err != nil {
		t.Fatalf("failed to unmarshal request for upsert_agreement: %v", err)
	}

	mcpRes, err := upsertTool.Execute(ctx, mcpReq)
	if err != nil {
		t.Fatalf("failed to execute upsert_agreement MCP tool: %v", err)
	}
	if mcpRes.IsError {
		t.Fatalf("upsert_agreement MCP tool returned error: %s", mcpRes.Content)
	}

	var tcs []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(mcpRes.Content), &tcs); err != nil {
		t.Fatalf("failed to parse MCP content: %v", err)
	}
	var createdAg2 Agreement
	if err := json.Unmarshal([]byte(tcs[0].Text), &createdAg2); err != nil {
		t.Fatalf("failed to unmarshal created agreement from MCP: %v", err)
	}
	if createdAg2.Id == "" {
		t.Error("expected second agreement created via MCP to have an ID")
	}

	// List agreements via MCP tool list_agreements
	var listTool mcp.Tool
	for _, tool := range tools {
		if tool.Name == OpListAgreements {
			listTool = tool
			break
		}
	}

	listArgs := map[string]any{
		"tenant_id":       tenantID,
		"catalog_item_id": itemID,
	}
	listArgsBytes, _ := json.Marshal(listArgs)
	listArgsStr, _ := json.Marshal(string(listArgsBytes))
	err = json.Unmarshal([]byte(`{"params":{"name":"`+OpListAgreements+`","arguments":`+string(listArgsStr)+`}}`), &mcpReq)
	if err != nil {
		t.Fatalf("failed to unmarshal request for list_agreements: %v", err)
	}

	mcpRes, err = listTool.Execute(ctx, mcpReq)
	if err != nil {
		t.Fatalf("failed to execute list_agreements MCP tool: %v", err)
	}
	if err := json.Unmarshal([]byte(mcpRes.Content), &tcs); err != nil {
		t.Fatalf("failed to parse list_agreements MCP content: %v", err)
	}
	var mcpList AgreementList
	if err := json.Unmarshal([]byte(tcs[0].Text), &mcpList); err != nil {
		t.Fatalf("failed to unmarshal AgreementList from MCP: %v", err)
	}
	if len(mcpList) != 2 {
		t.Errorf("expected 2 agreements from list_agreements MCP, got %d", len(mcpList))
	}

	// 5. Test DeleteAgreement
	err = module.DeleteAgreement(tenantID, updated.Id)
	if err != nil {
		t.Fatalf("failed to delete agreement: %v", err)
	}
	listAfterDelete, _ := module.ListAgreements(tenantID, itemID)
	if len(listAfterDelete) != 1 {
		t.Errorf("expected 1 agreement after deletion, got %d", len(listAfterDelete))
	}
	if len(pub.Events) != 4 || pub.Events[3] != "catalog.agreement.deleted" {
		t.Errorf("expected catalog.agreement.deleted event, got %v", pub.Events)
	}
}
