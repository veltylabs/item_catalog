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
		if tool.Name == "get_catalog_item" {
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
	err = json.Unmarshal([]byte(`{"params":{"name":"get_catalog_item","arguments":`+string(argStr)+`}}`), &mcpReq)
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
