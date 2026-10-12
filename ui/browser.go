package ui

import (
	itemcatalog "github.com/veltylabs/item_catalog"
	. "webtyp.com/dom"
	"webtyp.com/fmt"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/view"
)

// requireSpecialty injects the picker's current selection into every save.
type requireSpecialty struct {
	view.Presenter
	sel      *SignalString
	tenantID string
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
			if item.TenantId == "" && p.tenantID != "" {
				item.TenantId = p.tenantID
			}
		}
	}
	s.Save(recs, done)
}

func (p requireSpecialty) Update(ids []string, rec model.Model, fields []string, done func(error)) {
	u, ok := p.Presenter.(view.Updater)
	if !ok {
		done(fmt.Err("requireSpecialty: underlying presenter cannot update"))
		return
	}
	u.Update(ids, rec, fields, done)
}

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
var _ view.Updater = requireSpecialty{}
var _ view.Deleter = requireSpecialty{}

// Browser construye la vista de este módulo para el registro en modules/browser.go.
func Browser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error) {
	picker := newSpecialtyPicker(caller, nil)

	itemsView, err := crudview.New(crudview.Config{
		ParentID:  ID + ".items",
		Presenter: requireSpecialty{Presenter: itemcatalog.NewView(caller), sel: picker.sel, tenantID: tenantID},
		IDs:       ids,
		Context:   picker.render("catalog-item-specialty"),
		NewRecord: func() model.Model {
			return &itemcatalog.CatalogItem{
				Type:     itemcatalog.ItemTypeService,
				Currency: "CLP",
				Price:    0,
				IsActive: true,
			}
		},
	})
	if err != nil {
		return nil, err
	}

	specialtiesView, err := crudview.New(crudview.Config{
		ParentID:  ID + ".specialties",
		Presenter: itemcatalog.NewSpecialtyView(caller),
		IDs:       ids,
		NewRecord: func() model.Model {
			return &itemcatalog.Specialty{
				IsPublished: true,
			}
		},
	})
	if err != nil {
		return nil, err
	}
	servicesMod := platformd.NewUIModule(ID+".items", "Servicios", IconServices, itemsView)
	specialtiesMod := platformd.NewUIModule(ID+".specialties", "Especialidades", IconSpecialties, specialtiesView)

	picker.load(nil)

	return platformd.NewGroupUIModule(ID, Label, IconCatalog, []platformd.UIModule{
		servicesMod,
		specialtiesMod,
	}), nil
}

