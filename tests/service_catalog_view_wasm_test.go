//go:build wasm

package tests

import (
	"testing"

	servicecatalog "github.com/veltylabs/service_catalog"
	"github.com/veltylabs/service_catalog/ui"
	"webtyp.com/json"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/view"
)

// itemsPanel reaches the "Servicios" child's own *crudview.CrudView. Browser
// returns a platformd.GroupModule containing Servicios and Especialidades as submodules.
func itemsPanel(t *testing.T, mod platformd.UIModule) *crudview.CrudView {
	t.Helper()
	group, ok := mod.(platformd.GroupModule)
	if !ok {
		t.Fatalf("expected platformd.GroupModule, got %T", mod)
	}
	children := group.Children()
	if len(children) == 0 {
		t.Fatalf("expected GroupModule to have children")
	}
	cv, ok := children[0].View().(*crudview.CrudView)
	if !ok {
		t.Fatalf("expected child 0 (Servicios) view to be a *crudview.CrudView, got %T", children[0].View())
	}
	return cv
}

func TestWASM_ItemCatalog_View(t *testing.T) {
	var calledOps []string
	mock := &mockCaller{
		onCall: func(op string, args model.Encodable, into model.Decodable) error {
			calledOps = append(calledOps, op)
			if op == servicecatalog.ModelName+"."+servicecatalog.OpListServices {
				list := servicecatalog.ServiceList{
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

	m, err := ui.Browser(mock, &testIDGen{}, "")
	if err != nil {
		t.Fatalf("ui.Browser: %v", err)
	}

	cv := itemsPanel(t, m)
	cv.Init(nil)
	m.Activate()

	// Verifica que list_services se llamó en Init
	found := false
	for _, op := range calledOps {
		if op == servicecatalog.ModelName+"."+servicecatalog.OpListServices {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected OpListServices to be called on init, ops=%v", calledOps)
	}
}

// TestWASM_ItemCatalog_SaveAndDelete es el test que habría atrapado el bug
// METHOD_NOT_FOUND del camino de escritura: prueba que seleccionar una
// tarjeta llena el formulario desde el registro cacheado, y que
// Save/Delete despachan las ops reales (upsert_service /
// delete_service) con los datos del registro, no el nombre de la tool
// como método RPC pelado.
func TestWASM_ItemCatalog_SaveAndDelete(t *testing.T) {
	seed := &servicecatalog.Service{
		Id: "svc1", TenantId: "tenant1", Sku: "SKU1", Name: "Service One",
		Description: "General consultation service",
		Price:       100, Currency: "CLP", IsActive: true, UpdatedAt: 1,
	}
	// Una especialidad, para que el picker de Browser tenga algo que elegir
	// y el campo specialty_id del formulario tenga opciones.
	specialty := &servicecatalog.Specialty{Id: "spec-1", Name: "General"}

	mock := &mockCaller{
		onCall: func(op string, args model.Encodable, into model.Decodable) error {
			switch op {
			case servicecatalog.ModelName + "." + servicecatalog.OpListServices:
				list := servicecatalog.ServiceList{seed}
				var out []byte
				_ = json.Encode(&list, &out)
				return json.Decode(string(out), into)
			case servicecatalog.ModelName + "." + servicecatalog.OpListSpecialties:
				list := servicecatalog.SpecialtyList{specialty}
				var out []byte
				_ = json.Encode(&list, &out)
				return json.Decode(string(out), into)
			}
			return nil
		},
	}

	m, err := ui.Browser(mock, &testIDGen{}, "")
	if err != nil {
		t.Fatalf("ui.Browser: %v", err)
	}

	cv := itemsPanel(t, m)
	cv.Init(nil)
	m.Activate()

	// Seleccionar la tarjeta debe llenar el formulario desde el registro
	// cacheado (byID). Los selectAction/saveAction/deleteAction propios de
	// CrudView son no exportados — manejar Presenter directamente ejercita
	// el mismo camino real de despacho de ops que este test protege
	// (upsert_service / delete_service, no el nombre de la tool
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
	if mock.lastOp != servicecatalog.ModelName+"."+servicecatalog.OpUpsertService {
		t.Errorf("expected last op %q after save, got %q", servicecatalog.ModelName+"."+servicecatalog.OpUpsertService, mock.lastOp)
	}

	updater, ok := cv.Presenter.(view.Updater)
	if !ok {
		t.Fatal("expected Presenter to implement view.Updater")
	}
	var uerr error
	updater.Update([]string{seed.Id}, record, []string{"name"}, func(err error) { uerr = err })
	if uerr != nil {
		t.Fatalf("Update: %v", uerr)
	}
	if mock.lastOp != servicecatalog.ModelName+"."+servicecatalog.OpUpdateService {
		t.Errorf("expected last op %q after update, got %q", servicecatalog.ModelName+"."+servicecatalog.OpUpdateService, mock.lastOp)
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
	if mock.lastOp != servicecatalog.ModelName+"."+servicecatalog.OpDeleteService {
		t.Errorf("expected last op %q after delete, got %q", servicecatalog.ModelName+"."+servicecatalog.OpDeleteService, mock.lastOp)
	}
}
