package ui

import (
	itemcatalog "github.com/veltylabs/item_catalog"
	"webtyp.com/dom"
	"webtyp.com/fmt"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/router"
)

// errListSpecialties is logged when the specialty options cannot be loaded:
// the form then has an empty select, never a silent one.
const errListSpecialties = "item_catalog: could not load specialties for the item form:"

// Browser construye la vista de este módulo para el registro en modules/browser.go.
func Browser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error) {
	itemsView, err := crudview.New(crudview.Config{
		ParentID:  ID + ".items",
		Presenter: itemcatalog.NewView(caller),
		IDs:       ids,
		NewRecord: func() model.Model {
			return &itemcatalog.CatalogItem{
				TenantId: tenantID,
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

	out := &itemcatalog.SpecialtyList{}
	caller.Call(itemcatalog.ModelName+"."+itemcatalog.OpListSpecialties, nil, out, func(err error) {
		if err != nil {
			dom.Log(errListSpecialties, err)
			return
		}
		var opts []fmt.KeyValue
		for _, s := range *out {
			opts = append(opts, fmt.KeyValue{Key: s.Id, Value: s.Name})
		}
		itemsView.SetOptions("specialty_id", opts...)
	})

	return platformd.NewGroupUIModule(ID, Label, IconCatalog, []platformd.UIModule{
		servicesMod,
		specialtiesMod,
	}), nil
}
