package item_catalog

import (
	itemcatalog "github.com/veltylabs/item_catalog"
	"webtyp.com/components/decktabs"
	. "webtyp.com/dom"
	"webtyp.com/fmt"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/svg"
	"webtyp.com/view"
)

// requireSpecialty injects the picker's current selection into every save.
// catalog_item.specialty_id is model.Text() (base kind, no input.* widget) —
// by design: it is a Ref column into specialty, and this ecosystem's rule is
// "input.X() only on what a person edits in a form directly", not on a
// foreign key a picker already resolves (same shape as
// appointment_booking's employeeServiceConfigLister overwriting staff_id).
// Without this, specialty_id always saves empty and CatalogItemModel's
// NotNull rejects every save — confirmed live: the "+" button produced no
// row and no visible error (item_catalog's own Browser wired no OnSaved
// notification either, so the failure was completely silent).
type requireSpecialty struct {
	view.Presenter
	sel *SignalString
}

func (p requireSpecialty) Save(recs []model.Model, done func(error)) {
	s, ok := p.Presenter.(view.Saver)
	if !ok {
		done(fmt.Err("requireSpecialty: underlying presenter cannot save"))
		return
	}
	chosen := p.sel.Get()
	if chosen == "" {
		done(fmt.Err("Seleccione una especialidad antes de guardar"))
		return
	}
	for _, rec := range recs {
		if item, ok := rec.(*itemcatalog.CatalogItem); ok {
			item.SpecialtyId = chosen
		}
	}
	s.Save(recs, done)
}

// Delete reenvía explícitamente: embeber una INTERFAZ solo promueve los
// métodos que view.Presenter mismo declara, nunca los de view.Deleter. A
// diferencia de requirePatient de clinical_encounter (cuyo presenter
// subyacente genuinamente no tiene operación de borrado), el de
// itemcatalog.NewView SÍ la tiene — Ops declara Delete: OpDeleteItem — así
// que omitir esto eliminaría silenciosamente el botón de pie de página "🗑"
// que crudview ya renderiza en cualquier otra pantalla.
func (p requireSpecialty) Delete(ids []string, done func(error)) {
	d, ok := p.Presenter.(view.Deleter)
	if !ok {
		done(fmt.Err("requireSpecialty: underlying presenter cannot delete"))
		return
	}
	d.Delete(ids, done)
}

var _ view.Presenter = requireSpecialty{}
var _ view.Saver = requireSpecialty{}
var _ view.Deleter = requireSpecialty{}

// Browser construye la vista de este módulo para el registro en modules/browser.go.
// Dos pestañas, no un crudview.New como la mayoría de los módulos: un elemento del
// catálogo necesita una especialidad para existir Y ser seleccionado antes de poder
// guardarse en absoluto (ver requireSpecialty), así que "Especialidades" tiene que ser
// accesible desde la misma pantalla — un inquilino sin especialidades no tendría otro
// camino para crear su primer elemento de catálogo. tenantID no se usa — ver el
// comentario en server.go sobre el mismo parámetro.
func Browser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error) {
	picker := newSpecialtyPicker(caller, nil)

	itemsView, err := crudview.New(crudview.Config{
		ParentID:  ID + ".items",
		Presenter: requireSpecialty{Presenter: itemcatalog.NewView(caller), sel: picker.sel},
		IDs:       ids,
		Context:   picker.render("catalog-item-specialty"),
	})
	if err != nil {
		return nil, err
	}

	specialtiesView, err := crudview.New(crudview.Config{
		ParentID:  ID + ".specialties",
		Presenter: itemcatalog.NewSpecialtyView(caller),
		IDs:       ids,
	})
	if err != nil {
		return nil, err
	}

	tabs := &decktabs.DeckTabs{
		Label: Label,
		Items: []decktabs.Item{
			{ID: "items", Label: "Servicios", Panel: itemsView},
			{ID: "specialties", Label: "Especialidades", Panel: specialtiesView},
		},
	}
	picker.load(nil)

	return platformd.NewUIModule(ID, Label, svg.Icon(ID), tabs), nil
}
