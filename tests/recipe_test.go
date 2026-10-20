package tests

import (
	"testing"

	servicecatalog "github.com/veltylabs/service_catalog"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
)

// recipeService builds a module with one service ("Limpieza dental") in tenant-1 and returns both.
func recipeService(t *testing.T) (*servicecatalog.Module, servicecatalog.Service) {
	t.Helper()
	m, err := servicecatalog.New(orm.New(mem.New()), servicecatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}
	spec, err := m.UpsertSpecialty(servicecatalog.Specialty{TenantId: "tenant-1", Prefix: "do", Slug: "dental", Name: "Dental"})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := m.CreateService(servicecatalog.Service{TenantId: "tenant-1", SpecialtyId: spec.Id, Sku: "do1", Name: "Limpieza dental", Price: 20000, Currency: "CLP", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	return m, svc
}

func TestServiceRecipe_LinesInOrder(t *testing.T) {
	m, svc := recipeService(t)
	for _, l := range []servicecatalog.ServiceSupply{
		{TenantId: "tenant-1", ServiceId: svc.Id, ItemId: "gloves", Quantity: 2},
		{TenantId: "tenant-1", ServiceId: svc.Id, ItemId: "gauze", Quantity: 1},
	} {
		if _, err := m.UpsertServiceSupply(l); err != nil {
			t.Fatalf("UpsertServiceSupply(%s): %v", l.ItemId, err)
		}
	}

	ids, qty, err := m.ServiceRecipe("tenant-1", svc.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != "gloves" || qty[0] != 2 || ids[1] != "gauze" || qty[1] != 1 {
		t.Errorf("got %v %v, want [gloves gauze] [2 1]", ids, qty)
	}
}

func TestServiceRecipe_Rules(t *testing.T) {
	m, svc := recipeService(t)
	line, err := m.UpsertServiceSupply(servicecatalog.ServiceSupply{TenantId: "tenant-1", ServiceId: svc.Id, ItemId: "gloves", Quantity: 2})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		line servicecatalog.ServiceSupply
		want error
	}{
		{"same supply twice", servicecatalog.ServiceSupply{TenantId: "tenant-1", ServiceId: svc.Id, ItemId: "gloves", Quantity: 1}, servicecatalog.ErrSupplyLineExists},
		{"negative quantity", servicecatalog.ServiceSupply{TenantId: "tenant-1", ServiceId: svc.Id, ItemId: "gauze", Quantity: -1}, servicecatalog.ErrQuantity},
		{"unknown service", servicecatalog.ServiceSupply{TenantId: "tenant-1", ServiceId: "nope", ItemId: "gauze", Quantity: 1}, servicecatalog.ErrNotFound},
		{"service of another tenant", servicecatalog.ServiceSupply{TenantId: "tenant-2", ServiceId: svc.Id, ItemId: "gauze", Quantity: 1}, servicecatalog.ErrNotFound},
	}
	for _, c := range cases {
		if _, err := m.UpsertServiceSupply(c.line); err != c.want {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}

	if _, err := m.UpsertServiceSupply(servicecatalog.ServiceSupply{TenantId: "tenant-1", ServiceId: svc.Id, ItemId: "gauze", Quantity: 0}); err == nil {
		t.Error("a zero quantity must be refused")
	} else if _, ok := err.(servicecatalog.ValidationError); !ok {
		t.Errorf("zero quantity (the field is required): got %T %v, want a ValidationError", err, err)
	}

	line.Quantity = 3
	if got, err := m.UpsertServiceSupply(line); err != nil || got.Quantity != 3 {
		t.Errorf("updating its own line is not a duplicate: %v %v", got, err)
	}
	if err := m.DeleteServiceSupply("tenant-2", line.Id); err != servicecatalog.ErrSupplyLineNotFound {
		t.Errorf("delete from another tenant: got %v, want ErrSupplyLineNotFound", err)
	}
	if err := m.DeleteServiceSupply("tenant-1", line.Id); err != nil {
		t.Fatal(err)
	}
	if ids, _, err := m.ServiceRecipe("tenant-1", svc.Id); err != nil || len(ids) != 0 {
		t.Errorf("after deleting, the recipe is empty: %v %v", ids, err)
	}
	if _, _, err := m.ServiceRecipe("tenant-1", "nope"); err != servicecatalog.ErrNotFound {
		t.Errorf("recipe of an unknown service: got %v, want ErrNotFound", err)
	}
}
