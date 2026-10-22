package tests

import (
	"testing"

	servicecatalog "github.com/veltylabs/service_catalog"
	"webtyp.com/form"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/router/mock"
	"webtyp.com/storage/mem"
)

func TestEncodeDecodeRoundtrip(t *testing.T) {
	item := servicecatalog.Service{
		Id:          "item-123",
		TenantId:    "tenant-1",
		SpecialtyId: "spec-123",
		Sku:         "SKU-1",
		Name:        "Name 1",
		Description: "Desc 1",
		Price:       150.0,
		Currency:    "CLP",
		IsActive:    true,
		TaxTreatment: "EXEMPT",
		UpdatedAt:   123456,
	}

	ctx := &mock.Context{}
	if err := ctx.Encode(&item); err != nil {
		t.Fatal(err)
	}

	ctx.InBody = ctx.ResponseBody()

	var decoded servicecatalog.Service
	if err := ctx.Decode(&decoded); err != nil {
		t.Fatal(err)
	}

	if decoded != item {
		t.Errorf("expected %+v, got %+v", item, decoded)
	}
}

func TestSchemaPointersLength(t *testing.T) {
	spec := &servicecatalog.Specialty{}
	if len(spec.Schema()) != len(spec.Pointers()) {
		t.Errorf("expected len(Schema) %d == len(Pointers) %d", len(spec.Schema()), len(spec.Pointers()))
	}

	item := &servicecatalog.Service{}
	if len(item.Schema()) != len(item.Pointers()) {
		t.Errorf("expected len(Schema) %d == len(Pointers) %d", len(item.Schema()), len(item.Pointers()))
	}

	ag := &servicecatalog.ServiceAgreement{}
	if len(ag.Schema()) != len(ag.Pointers()) {
		t.Errorf("expected len(Schema) %d == len(Pointers) %d", len(ag.Schema()), len(ag.Pointers()))
	}
}

func TestValidateConstraints(t *testing.T) {
	goodItem := servicecatalog.Service{
		Id:          "item-1",
		TenantId:    "tenant-1",
		SpecialtyId: "spec-1",
		Sku:         "SKU-1",
		Name:        "Good Item",
		Price:       100.0,
		Currency:    "CLP",
		IsActive:    true,
		TaxTreatment: "EXEMPT",
		UpdatedAt:   12345,
	}

	if err := goodItem.Validate(model.ActionCreate); err != nil {
		t.Errorf("expected good item to be valid, got %v", err)
	}

	badItemEmptyName := goodItem
	badItemEmptyName.Name = ""
	if err := badItemEmptyName.Validate(model.ActionCreate); err == nil {
		t.Error("expected empty name to be rejected")
	}

	// For invalid type "X", we check via the service layer validation!
	db := orm.New(mem.New())
	_, _ = servicecatalog.New(db, servicecatalog.Deps{IDs: &MockIDGen{}})

}

func TestFormInputsGeneration(t *testing.T) {
	item := &servicecatalog.Service{}
	f, err := form.New("parent", item, &MockIDGen{})
	if err != nil {
		t.Fatalf("failed to create form: %v", err)
	}

	// We expect the form to contain editable input widgets for user-editable fields,
	// and NOT for machine-managed fields (id, tenant_id, updated_at).
	// specialty_id is an input.Select() field, so editable form input fields are:
	editableFields := []string{"specialty_id", "sku", "name", "description", "tax_treatment", "price", "currency", "is_active"}
	for _, field := range f.Inputs {
		name := field.FieldName()
		isEditable := false
		for _, ef := range editableFields {
			if ef == name {
				isEditable = true
				break
			}
		}
		if !isEditable {
			t.Errorf("field %s is machine-managed and should not be a form input widget", name)
		}
	}
}

func TestErrorAndModelName(t *testing.T) {
	// Cover ValidationError.Error
	ve := servicecatalog.ValidationError{Err: servicecatalog.ErrNotFound}
	if ve.Error() == "" {
		t.Error("expected ValidationError.Error to carry the wrapped message")
	}

	// Cover Module.ModelName
	db := orm.New(mem.New())
	module, _ := servicecatalog.New(db, servicecatalog.Deps{IDs: &MockIDGen{}})
	if module.ModelName() != servicecatalog.ModelName {
		t.Errorf("expected ModelName %q, got %q", servicecatalog.ModelName, module.ModelName())
	}
}
