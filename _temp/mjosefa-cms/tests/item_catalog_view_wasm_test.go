//go:build wasm

package tests

import (
	"testing"

	itemcatalog "github.com/veltylabs/item_catalog"
	"github.com/veltylabs/mjosefa-cms/modules/item_catalog"
	"webtyp.com/components/decktabs"
	"webtyp.com/dom"
	"webtyp.com/json"
	"webtyp.com/layout/crudview"
	"webtyp.com/model"
	"webtyp.com/view"
)

// itemsPanel reaches the "Servicios" tab's own *crudview.CrudView. Browser
// composes two tabs (Servicios + Especialidades — a catalog item cannot save
// without a specialty to pick, see requireSpecialty in browser.go), so the
// screen itself is a *decktabs.DeckTabs, not a bare crudview.
func itemsPanel(t *testing.T, comp dom.Component) *crudview.CrudView {
	t.Helper()
	tabs, ok := comp.(*decktabs.DeckTabs)
	if !ok {
		t.Fatalf("expected *decktabs.DeckTabs, got %T", comp)
	}
	cv, ok := tabs.Items[0].Panel.(*crudview.CrudView)
	if !ok {
		t.Fatalf("expected tab 0 (Servicios) to be a *crudview.CrudView, got %T", tabs.Items[0].Panel)
	}
	return cv
}

func TestWASM_ItemCatalog_View(t *testing.T) {
	var calledOps []string
	mock := &mockCaller{
		onCall: func(op string, args model.Encodable, into model.Decodable) error {
			calledOps = append(calledOps, op)
			if op == itemcatalog.ModelName+"."+itemcatalog.OpListItems {
				list := itemcatalog.CatalogItemList{
					{Id: "1", Name: "Service One", Sku: "SKU1"},
					{Id: "2", Name: "Service Two", Sku: "SKU2"},
				}
				var out []byte
				_ = json.Encode(&list, &out)
				return json.Decode(string(out), into)
			}
			return nil
		},
	}

	m, err := item_catalog.Browser(mock, &testIDGen{}, "")
	if err != nil {
		t.Fatalf("item_catalog.Browser: %v", err)
	}

	cv := itemsPanel(t, m.View())
	cv.Init(nil)

	// Verifica que list_catalog_items se llamó en Init
	found := false
	for _, op := range calledOps {
		if op == itemcatalog.ModelName+"."+itemcatalog.OpListItems {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected OpListItems to be called on init, ops=%v", calledOps)
	}
}

// TestWASM_ItemCatalog_SaveAndDelete es el test que habría atrapado el bug
// METHOD_NOT_FOUND del camino de escritura: prueba que seleccionar una
// tarjeta llena el formulario desde el registro cacheado, y que
// Save/Delete despachan las ops reales (upsert_catalog_item /
// delete_catalog_item) con los datos del registro, no el nombre de la tool
// como método RPC pelado.
func TestWASM_ItemCatalog_SaveAndDelete(t *testing.T) {
	seed := &itemcatalog.CatalogItem{
		Id: "svc1", TenantId: "tenant1", Sku: "SKU1", Name: "Service One",
		Description: "General consultation service",
		SpecialtyId: "spec-1", Type: "S",
		Price: 100, Currency: "CLP", IsActive: true, UpdatedAt: 1,
	}
	// Una especialidad, para que el picker de Browser tenga algo que elegir
	// y requireSpecialty.Save no rechace el guardado por falta de selección.
	specialty := &itemcatalog.Specialty{Id: "spec-1", Name: "General"}

	mock := &mockCaller{
		onCall: func(op string, args model.Encodable, into model.Decodable) error {
			switch op {
			case itemcatalog.ModelName + "." + itemcatalog.OpListItems:
				list := itemcatalog.CatalogItemList{seed}
				var out []byte
				_ = json.Encode(&list, &out)
				return json.Decode(string(out), into)
			case itemcatalog.ModelName + "." + itemcatalog.OpListSpecialties:
				list := itemcatalog.SpecialtyList{specialty}
				var out []byte
				_ = json.Encode(&list, &out)
				return json.Decode(string(out), into)
			}
			return nil
		},
	}

	m, err := item_catalog.Browser(mock, &testIDGen{}, "")
	if err != nil {
		t.Fatalf("item_catalog.Browser: %v", err)
	}

	cv := itemsPanel(t, m.View())
	cv.Init(nil)

	// Seleccionar la tarjeta debe llenar el formulario desde el registro
	// cacheado (byID). Los selectAction/saveAction/deleteAction propios de
	// CrudView son no exportados — manejar Presenter directamente ejercita
	// el mismo camino real de despacho de ops que este test protege
	// (upsert_catalog_item / delete_catalog_item, no el nombre de la tool
	// como método RPC pelado) sin necesitar un click de DOM.
	record := cv.Presenter.Select(seed.Id)
	if record == nil {
		t.Fatal("expected Select to return the cached record")
	}

	saver, ok := cv.Presenter.(view.Saver)
	if !ok {
		t.Fatal("expected Presenter to implement view.Saver")
	}
	var serr error
	saver.Save([]model.Model{record}, func(err error) { serr = err })
	if serr != nil {
		t.Fatalf("Save: %v", serr)
	}
	if mock.lastOp != itemcatalog.ModelName+"."+itemcatalog.OpUpsertItem {
		t.Errorf("expected last op %q after save, got %q", itemcatalog.ModelName+"."+itemcatalog.OpUpsertItem, mock.lastOp)
	}

	deleter, ok := cv.Presenter.(view.Deleter)
	if !ok {
		t.Fatal("expected Presenter to implement view.Deleter")
	}
	var derr error
	deleter.Delete([]string{seed.Id}, func(err error) { derr = err })
	if derr != nil {
		t.Fatalf("Delete: %v", derr)
	}
	if mock.lastOp != itemcatalog.ModelName+"."+itemcatalog.OpDeleteItem {
		t.Errorf("expected last op %q after delete, got %q", itemcatalog.ModelName+"."+itemcatalog.OpDeleteItem, mock.lastOp)
	}
}

// TestWASM_ItemCatalog_SaveWithoutSpecialtyIsRejected: sin especialidades
// registradas el picker no tiene nada que elegir, y el guardado debe
// rechazarse ANTES de llegar al servidor — nunca en silencio. Esto es
// exactamente lo que faltaba antes de esta pantalla: el "+" no producía
// fila ni error visible.
func TestWASM_ItemCatalog_SaveWithoutSpecialtyIsRejected(t *testing.T) {
	mock := &mockCaller{
		onCall: func(op string, args model.Encodable, into model.Decodable) error {
			return nil // sin especialidades: list_specialties responde vacío
		},
	}

	m, err := item_catalog.Browser(mock, &testIDGen{}, "")
	if err != nil {
		t.Fatalf("item_catalog.Browser: %v", err)
	}
	cv := itemsPanel(t, m.View())
	cv.Init(nil)

	saver, ok := cv.Presenter.(view.Saver)
	if !ok {
		t.Fatal("expected Presenter to implement view.Saver")
	}
	var serr error
	saver.Save([]model.Model{&itemcatalog.CatalogItem{Name: "X", Sku: "X"}}, func(err error) { serr = err })
	if serr == nil {
		t.Fatal("expected Save to fail loudly when no specialty is selected")
	}
	if mock.lastOp == itemcatalog.ModelName+"."+itemcatalog.OpUpsertItem {
		t.Error("expected the save to be rejected before reaching the server")
	}
}
