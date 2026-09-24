package tests

import (
	"testing"

	itemcatalog "github.com/veltylabs/item_catalog"
	"github.com/veltylabs/item_catalog/ui"
	"webtyp.com/components/decktabs"
	"webtyp.com/json"
	"webtyp.com/layout/crudview"
	"webtyp.com/model"
)

func TestView_Creation(t *testing.T) {
	mock := &mockCaller{}
	m, err := ui.Browser(mock, &testIDGen{}, "")
	if err != nil {
		t.Fatalf("ui.Browser: %v", err)
	}

	if m.ModelName() != "catalog_item" {
		t.Errorf("expected ModelName %q, got %q", "catalog_item", m.ModelName())
	}
	if m.Label() != "Catálogo" {
		t.Errorf("expected Label %q, got %q", "Catálogo", m.Label())
	}
}

func TestView_LoadList(t *testing.T) {
	mock := &mockCaller{
		onCall: func(op string, args model.Encodable, into model.Decodable) error {
			if op == itemcatalog.ModelName+"."+itemcatalog.OpListItems {
				list := itemcatalog.CatalogItemList{
					{Id: "1", Name: "Service One", Sku: "SKU1"},
					{Id: "2", Name: "Service Two", Sku: "SKU2"},
				}
				var out []byte
				if err := json.Encode(&list, &out); err != nil {
					return err
				}
				return json.Decode(string(out), into)
			}
			return nil
		},
	}

	m, err := ui.Browser(mock, &testIDGen{}, "")
	if err != nil {
		t.Fatalf("ui.Browser: %v", err)
	}

	tabs, ok := m.View().(*decktabs.DeckTabs)
	if !ok {
		t.Fatalf("expected *decktabs.DeckTabs, got %T", m.View())
	}
	cv, ok := tabs.Items[0].Panel.(*crudview.CrudView)
	if !ok {
		t.Fatalf("expected tab 0 (Servicios) to be a *crudview.CrudView, got %T", tabs.Items[0].Panel)
	}
	cv.Init(nil)

	if mock.lastOp != itemcatalog.ModelName+"."+itemcatalog.OpListItems {
		t.Errorf("expected last op %q, got %q", itemcatalog.ModelName+"."+itemcatalog.OpListItems, mock.lastOp)
	}
}
