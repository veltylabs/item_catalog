//go:build browser

package ui

import (
	servicecatalog "github.com/veltylabs/service_catalog"
	"webtyp.com/dom"
	"webtyp.com/fmt"
	"webtyp.com/layout/crudview"
	"webtyp.com/layout/platformd"
	"webtyp.com/model"
	"webtyp.com/router"
)

// errListSpecialties is logged when the specialty options cannot be loaded:
// the form then has an empty select, never a silent one.
const errListSpecialties = "service_catalog: could not load specialties for the service form:"

// Browser construye la vista de este módulo para el registro en modules/browser.go.
func Browser(caller router.Caller, ids model.IDGenerator, tenantID string) (platformd.UIModule, error) {
	servicesView, err := crudview.New(crudview.Config{
		ParentID:  ID + ".services",
		Presenter: servicecatalog.NewView(caller),
		IDs:       ids,
		NewRecord: func() model.Model {
			return &servicecatalog.Service{
				TenantId: tenantID,
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
		Presenter: servicecatalog.NewSpecialtyView(caller),
		IDs:       ids,
		NewRecord: func() model.Model {
			return &servicecatalog.Specialty{
				IsPublished: true,
			}
		},
	})
	if err != nil {
		return nil, err
	}
	servicesMod := platformd.NewUIModule(ID+".services", "Servicios", IconServices, servicesView)
	specialtiesMod := platformd.NewUIModule(ID+".specialties", "Especialidades", IconSpecialties, specialtiesView)

	servicesView.SetOptions("tax_treatment",
		fmt.KeyValue{Key: servicecatalog.TaxTaxed, Value: "Taxed"},
		fmt.KeyValue{Key: servicecatalog.TaxExempt, Value: "Exempt"},
	)

	out := &servicecatalog.SpecialtyList{}
	caller.Call(servicecatalog.ModelName+"."+servicecatalog.OpListSpecialties, nil, out, func(err error) {
		if err != nil {
			dom.Log(errListSpecialties, err)
			return
		}
		var opts []fmt.KeyValue
		for _, s := range *out {
			opts = append(opts, fmt.KeyValue{Key: s.Id, Value: s.Name})
		}
		servicesView.SetOptions("specialty_id", opts...)
	})

	return platformd.NewGroupUIModule(ID, Label, IconCatalog, []platformd.UIModule{
		servicesMod,
		specialtiesMod,
	}), nil
}
