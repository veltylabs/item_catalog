package tests

import (
	"errors"
	"testing"

	itemcatalog "github.com/veltylabs/item_catalog"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/storage"
	"webtyp.com/storage/mem"
)

func TestServiceCard_EmptyArgsNotFound(t *testing.T) {
	db := orm.New(mem.New())
	module, err := itemcatalog.New(db, itemcatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, _, err = module.ServiceCard("", "")
	if err != itemcatalog.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestServiceCard_UnknownOrOtherTenantNotFound(t *testing.T) {
	db := orm.New(mem.New())
	module, err := itemcatalog.New(db, itemcatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}

	// Create specialty
	spec, _ := module.UpsertSpecialty(itemcatalog.Specialty{
		TenantId: "t1", Prefix: "aa", Slug: "a", Name: "A",
	})

	// Create service
	item, _ := module.CreateItem(itemcatalog.CatalogItem{
		TenantId:    "t1",
		SpecialtyId: spec.Id,
		Type:        itemcatalog.ItemTypeService,
		Sku:         "sku",
		Name:        "Service",
		Price:       100,
		Currency:    "USD",
		IsActive:    true,
	})

	_, _, _, _, err = module.ServiceCard("t2", item.Id)
	if err != itemcatalog.ErrNotFound {
		t.Fatalf("expected ErrNotFound for wrong tenant, got %v", err)
	}

	_, _, _, _, err = module.ServiceCard("t1", "unknown")
	if err != itemcatalog.ErrNotFound {
		t.Fatalf("expected ErrNotFound for unknown id, got %v", err)
	}
}

type errConn struct {
	storage.Conn
}

func (e *errConn) Compile(q storage.Query, m model.Model) (storage.Plan, error) {
	p, _ := e.Conn.Compile(q, m)
	return p, errors.New("db error")
}

func (e *errConn) DriverName() string {
	return "errConn"
}

func TestServiceCard_StorageErrorPropagates(t *testing.T) {
	// Use mem.New() as the underlying storage, but wrap it in our errConn
	underlying := mem.New()
	db := orm.New(&errConn{Conn: underlying})
	module, _ := itemcatalog.New(db, itemcatalog.Deps{IDs: &MockIDGen{}})

	_, _, _, _, err := module.ServiceCard("t", "id")
	if err == nil || err == itemcatalog.ErrNotFound {
		t.Fatalf("expected arbitrary error, got %v", err)
	}
	if err.Error() != "db error" {
		t.Fatalf("expected 'db error', got %v", err)
	}
}

func TestServiceCard_ProductIsNotFound(t *testing.T) {
	db := orm.New(mem.New())
	module, err := itemcatalog.New(db, itemcatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}

	spec, _ := module.UpsertSpecialty(itemcatalog.Specialty{
		TenantId: "t1", Prefix: "aa", Slug: "a", Name: "A",
	})

	item, _ := module.CreateItem(itemcatalog.CatalogItem{
		TenantId:    "t1",
		SpecialtyId: spec.Id,
		Type:        itemcatalog.ItemTypeProduct, // Es producto
		Sku:         "sku",
		Name:        "Product",
		Price:       100,
		Currency:    "USD",
		IsActive:    true,
	})

	_, _, _, _, err = module.ServiceCard("t1", item.Id)
	if err != itemcatalog.ErrNotFound {
		t.Fatalf("expected ErrNotFound for product, got %v", err)
	}
}

func TestServiceCard_ReturnsNamePriceCurrencyActive(t *testing.T) {
	db := orm.New(mem.New())
	module, err := itemcatalog.New(db, itemcatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}

	spec, _ := module.UpsertSpecialty(itemcatalog.Specialty{
		TenantId: "t1", Prefix: "aa", Slug: "a", Name: "A",
	})

	activeItem, _ := module.CreateItem(itemcatalog.CatalogItem{
		TenantId:    "t1",
		SpecialtyId: spec.Id,
		Type:        itemcatalog.ItemTypeService,
		Sku:         "sku1",
		Name:        "Active Service",
		Price:       1000.50,
		Currency:    "CLP",
		IsActive:    true,
	})

	inactiveItem, _ := module.CreateItem(itemcatalog.CatalogItem{
		TenantId:    "t1",
		SpecialtyId: spec.Id,
		Type:        itemcatalog.ItemTypeService,
		Sku:         "sku2",
		Name:        "Inactive Service",
		Price:       500,
		Currency:    "USD",
		IsActive:    true,
	})
	// Forzamos IsActive a false en base de datos ya que Validate de Model rechaza false por NotNull: true
	db.Update(&inactiveItem, orm.Eq("id", inactiveItem.Id))
	inactiveItem.IsActive = false
	db.Update(&inactiveItem, orm.Eq("id", inactiveItem.Id))

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
