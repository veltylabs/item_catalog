package tests

import (
	"testing"

	itemcatalog "github.com/veltylabs/item_catalog"
	"github.com/veltylabs/item_catalog/ui"
	"webtyp.com/json"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/msgtype"
	"webtyp.com/view"
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

	group, ok := m.(platformd.GroupModule)
	if !ok {
		t.Fatalf("expected platformd.GroupModule, got %T", m)
	}
	children := group.Children()
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
	if children[0].Label() != "Servicios" {
		t.Errorf("expected first child to be 'Servicios', got %q", children[0].Label())
	}
	if children[1].Label() != "Especialidades" {
		t.Errorf("expected second child to be 'Especialidades', got %q", children[1].Label())
	}

	cv, ok := children[0].View().(*crudview.CrudView)
	if !ok {
		t.Fatalf("expected child 0 (Servicios) to be a *crudview.CrudView, got %T", children[0].View())
	}
	cv.Init(nil)
	m.Activate()

	if mock.lastOp != itemcatalog.ModelName+"."+itemcatalog.OpListItems {
		t.Errorf("expected last op %q, got %q", itemcatalog.ModelName+"."+itemcatalog.OpListItems, mock.lastOp)
	}
}

type mockNotifier struct {
	lastType msgtype.Type
	lastMsg  string
}

func (n *mockNotifier) Notify(t msgtype.Type, msg string, dur platformd.Duration) {
	n.lastType = t
	n.lastMsg = msg
}

func TestView_UsesNotifierAndCRUD(t *testing.T) {
	mock := &mockCaller{}
	m, err := ui.Browser(mock, &testIDGen{}, "")
	if err != nil {
		t.Fatalf("ui.Browser: %v", err)
	}

	notifierMod, ok := m.(platformd.UsesNotifier)
	if !ok {
		t.Fatalf("expected platformd.UsesNotifier, got %T", m)
	}

	notifier := &mockNotifier{}
	notifierMod.UseNotifier(notifier)

	group := m.(platformd.GroupModule)
	children := group.Children()
	servicesCv, ok := children[0].View().(*crudview.CrudView)
	if !ok {
		t.Fatalf("expected *crudview.CrudView, got %T", children[0].View())
	}
	specsCv, ok := children[1].View().(*crudview.CrudView)
	if !ok {
		t.Fatalf("expected *crudview.CrudView, got %T", children[1].View())
	}

	// Verify that the framework auto-wired the notifier to the views
	if _, ok := any(servicesCv).(platformd.UsesNotifier); !ok {
		t.Fatal("expected servicesCv to implement platformd.UsesNotifier")
	}
	if _, ok := any(specsCv).(platformd.UsesNotifier); !ok {
		t.Fatal("expected specsCv to implement platformd.UsesNotifier")
	}
	if servicesCv.NewRecord == nil {
		t.Error("expected servicesCv.NewRecord to be configured")
	}
	if specsCv.NewRecord == nil {
		t.Error("expected specsCv.NewRecord to be configured")
	}
}


func TestView_AddSaveItemWorkflow(t *testing.T) {
	var savedItems []*itemcatalog.CatalogItem
	mock := &mockCaller{
		onCall: func(op string, args model.Encodable, into model.Decodable) error {
			t.Logf("onCall: op=%s, args=%T (%+v)", op, args, args)
			switch op {
			case itemcatalog.ModelName + "." + itemcatalog.OpListItems:
				list := itemcatalog.CatalogItemList{}
				for _, it := range savedItems {
					list = append(list, it)
				}
				var out []byte
				if err := json.Encode(&list, &out); err != nil {
					return err
				}
				return json.Decode(string(out), into)
			case itemcatalog.ModelName + "." + itemcatalog.OpListSpecialties:
				list := itemcatalog.SpecialtyList{
					&itemcatalog.Specialty{Id: "spec-1", Name: "Medicina General"},
				}
				var out []byte
				if err := json.Encode(&list, &out); err != nil {
					return err
				}
				return json.Decode(string(out), into)
			case itemcatalog.ModelName + "." + itemcatalog.OpUpsertItem:
				var out []byte
				if err := json.Encode(args, &out); err != nil {
					return err
				}
				var upsertArgs itemcatalog.UpsertItemArgs
				if err := json.Decode(string(out), &upsertArgs); err != nil {
					return err
				}
				for _, it := range upsertArgs.Records {
					item := it
					savedItems = append(savedItems, &item)
				}
				return nil
			}
			return nil
		},
	}

	m, err := ui.Browser(mock, &testIDGen{}, "test-tenant")
	if err != nil {
		t.Fatalf("ui.Browser: %v", err)
	}

	notifier := &mockNotifier{}
	m.(platformd.UsesNotifier).UseNotifier(notifier)

	group := m.(platformd.GroupModule)
	servicesCv := group.Children()[0].View().(*crudview.CrudView)
	servicesCv.Init(nil)
	m.Activate()

	// Sku & Name entered
	rec := servicesCv.NewRecord().(*itemcatalog.CatalogItem)
	rec.Sku = "SKU-99"
	rec.Name = "Servicio Neurología"

	saver := servicesCv.Presenter.(view.Saver)
	saver.Save([]model.Model{rec}, func(err error) {
		if err != nil {
			t.Fatalf("save error: %v", err)
		}
	})

	if len(savedItems) != 1 {
		t.Fatalf("expected 1 saved item, got %d", len(savedItems))
	}
	if savedItems[0].Sku != "SKU-99" {
		t.Errorf("expected Sku 'SKU-99', got %q", savedItems[0].Sku)
	}
	if savedItems[0].Name != "Servicio Neurología" {
		t.Errorf("expected Name 'Servicio Neurología', got %q", savedItems[0].Name)
	}
}
