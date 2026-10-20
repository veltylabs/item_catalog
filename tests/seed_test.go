package tests

import (
	"testing"

	servicecatalog "github.com/veltylabs/service_catalog"
	"github.com/veltylabs/service_catalog/seed"
	"webtyp.com/events/mock"
	"webtyp.com/orm"
	"webtyp.com/storage/mem"
)

func TestSeedLoad(t *testing.T) {
	db := orm.New(mem.New())
	broker := &mock.Broker{}
	idGen := &testIDGen{}

	m, err := servicecatalog.New(db, servicecatalog.Deps{
		IDs:       idGen,
		Publisher: broker,
	})
	if err != nil {
		t.Fatalf("failed to create module: %v", err)
	}

	data, err := seed.Load(m, "test-tenant")
	if err != nil {
		t.Fatalf("seed.Load failed: %v", err)
	}

	if len(data.Specialties) != 3 {
		t.Errorf("expected 3 specialties, got %d", len(data.Specialties))
	}
	if len(data.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(data.Items))
	}
}
