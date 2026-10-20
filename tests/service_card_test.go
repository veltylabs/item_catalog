package tests

import (
	"testing"

	servicecatalog "github.com/veltylabs/service_catalog"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/storage"
	"webtyp.com/storage/mem"
)

func TestServiceCard_EmptyArgsNotFound(t *testing.T) {
	db := orm.New(mem.New())
	module, err := servicecatalog.New(db, servicecatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, _, err = module.ServiceCard("", "")
	if err != servicecatalog.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestServiceCard_UnknownOrOtherTenantNotFound(t *testing.T) {
	db := orm.New(mem.New())
	module, err := servicecatalog.New(db, servicecatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}

	// Create specialty
	spec, err := module.UpsertSpecialty(servicecatalog.Specialty{
		TenantId: "t1", Prefix: "aa", Slug: "a", Name: "A",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create service
	item, err := module.CreateService(servicecatalog.Service{
		TenantId:    "t1",
		SpecialtyId: spec.Id,
		Sku:         "sku",
		Name:        "Service",
		Price:       100,
		Currency:    "USD",
		IsActive:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, _, err = module.ServiceCard("t2", item.Id)
	if err != servicecatalog.ErrNotFound {
		t.Fatalf("expected ErrNotFound for wrong tenant, got %v", err)
	}

	_, _, _, _, err = module.ServiceCard("t1", "unknown")
	if err != servicecatalog.ErrNotFound {
		t.Fatalf("expected ErrNotFound for unknown id, got %v", err)
	}
}

type errConn struct {
	storage.Conn
}

func (e *errConn) Compile(q storage.Query, m model.Model) (storage.Plan, error) {
	p, _ := e.Conn.Compile(q, m)
	return p, fmt.Err("db error")
}

func TestServiceCard_StorageErrorPropagates(t *testing.T) {
	// Use mem.New() as the underlying storage, but wrap it in our errConn
	underlying := mem.New()
	db := orm.New(&errConn{Conn: underlying})
	module, err := servicecatalog.New(db, servicecatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, _, err = module.ServiceCard("t", "id")
	if err == nil || err == servicecatalog.ErrNotFound {
		t.Fatalf("expected arbitrary error, got %v", err)
	}
	if err.Error() != "db error" {
		t.Fatalf("expected 'db error', got %v", err)
	}
}

func TestServiceCard_ReturnsNamePriceCurrencyActive(t *testing.T) {
	db := orm.New(mem.New())
	module, err := servicecatalog.New(db, servicecatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}

	spec, err := module.UpsertSpecialty(servicecatalog.Specialty{
		TenantId: "t1", Prefix: "aa", Slug: "a", Name: "A",
	})
	if err != nil {
		t.Fatal(err)
	}

	activeItem, err := module.CreateService(servicecatalog.Service{
		TenantId:    "t1",
		SpecialtyId: spec.Id,
		Sku:         "sku1",
		Name:        "Active Service",
		Price:       1000.50,
		Currency:    "CLP",
		IsActive:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	inactiveItem, err := module.CreateService(servicecatalog.Service{
		TenantId:    "t1",
		SpecialtyId: spec.Id,
		Sku:         "sku2",
		Name:        "Inactive Service",
		Price:       500,
		Currency:    "USD",
		IsActive:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Forzamos IsActive a false en base de datos ya que Validate de Model rechaza false por NotNull: true
	inactiveItem.IsActive = false
	if err := db.Update(&inactiveItem, orm.Eq("id", inactiveItem.Id)); err != nil {
		t.Fatal(err)
	}

	// Test activo
	name, price, currency, active, err := module.ServiceCard("t1", activeItem.Id)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if name != "Active Service" || price != 1000.50 || currency != "CLP" || active != true {
		t.Fatalf("got unexpected active item fields: %q, %v, %q, %v", name, price, currency, active)
	}

	// Test inactivo
	name, price, currency, active, err = module.ServiceCard("t1", inactiveItem.Id)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if name != "Inactive Service" || price != 500 || currency != "USD" || active != false {
		t.Fatalf("got unexpected inactive item fields: %q, %v, %q, %v", name, price, currency, active)
	}
}
