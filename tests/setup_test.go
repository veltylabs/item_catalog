package tests

import (
	"fmt"

	itemcatalog "github.com/veltylabs/item-catalog"
)

type MockUI struct {
	RenderItemListCalled bool
	RenderItemFormCalled bool
}

func (m *MockUI) RenderItemList(items []itemcatalog.CatalogItem, activeFilter string) string {
	m.RenderItemListCalled = true
	return fmt.Sprintf("List: %d items", len(items))
}

func (m *MockUI) RenderItemForm(item *itemcatalog.CatalogItem) string {
	m.RenderItemFormCalled = true
	if item == nil {
		return "Empty Form"
	}
	return "Form: " + item.Name
}

func (m *MockUI) RenderFilterSelector(current string) string {
	return "Filter: " + current
}

type MockPublisher struct {
	Events []string
}

func (m *MockPublisher) Publish(event string, payload any) error {
	m.Events = append(m.Events, event)
	return nil
}
