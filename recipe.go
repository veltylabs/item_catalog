package servicecatalog

import (
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/router"
	"webtyp.com/time"
)

// The supply recipe of a service: what one performance uses. clinical_encounter reads it (ServiceRecipe) to load the
// supplies of an attention, and supply_manager takes them out of stock; neither module is imported here.
const (
	OpListServiceSupplies = "list_service_supplies"
	OpUpsertServiceSupply = "upsert_service_supply"
	OpDeleteServiceSupply = "delete_service_supply"

	resServiceSupply = "service_supply"
)

// ListServiceSupplies returns the recipe of one service (or of every service when serviceId is ""), in insertion order.
func (m *Module) ListServiceSupplies(tenantId, serviceId string) ([]ServiceSupply, error) {
	var proxy ServiceSupply
	qb := m.db.Query(&proxy).Where(ServiceSupply_.TenantId).Eq(tenantId)
	if serviceId != "" {
		qb = qb.Where(ServiceSupply_.ServiceId).Eq(serviceId)
	}
	list, err := ReadAllServiceSupply(qb)
	if err != nil {
		return nil, err
	}
	out := make([]ServiceSupply, len(list))
	for i, l := range list {
		out[i] = *l
	}
	return out, nil
}

func (m *Module) getServiceSupply(tenantId, id string) (ServiceSupply, error) {
	var proxy ServiceSupply
	qb := m.db.Query(&proxy).Where(ServiceSupply_.Id).Eq(id).Where(ServiceSupply_.TenantId).Eq(tenantId)
	got, err := ReadOneServiceSupply(qb, &proxy)
	if orm.IsNotFound(err) {
		return ServiceSupply{}, ErrSupplyLineNotFound
	}
	if err != nil {
		return ServiceSupply{}, err
	}
	return *got, nil
}

// UpsertServiceSupply creates a recipe line (no id) or updates one. The service must exist in the tenant (ErrNotFound),
// the quantity must be > 0 (ErrQuantity) and a supply appears once per service (ErrSupplyLineExists).
func (m *Module) UpsertServiceSupply(s ServiceSupply) (ServiceSupply, error) {
	action := model.ActionCreate
	var existing ServiceSupply
	if s.Id != "" {
		var err error
		if existing, err = m.getServiceSupply(s.TenantId, s.Id); err != nil {
			return ServiceSupply{}, err
		}
		action = model.ActionUpdate
	} else {
		s.Id = m.ids.NewID()
	}
	s.UpdatedAt = time.Now()
	if err := s.Validate(action); err != nil {
		return ServiceSupply{}, ValidationError{Err: err}
	}
	if s.Quantity <= 0 {
		return ServiceSupply{}, ErrQuantity
	}
	if _, err := m.GetService(s.TenantId, s.ServiceId); err != nil {
		return ServiceSupply{}, err
	}
	if action == model.ActionCreate || existing.ItemId != s.ItemId || existing.ServiceId != s.ServiceId {
		lines, err := m.ListServiceSupplies(s.TenantId, s.ServiceId)
		if err != nil {
			return ServiceSupply{}, err
		}
		for _, l := range lines {
			if l.ItemId == s.ItemId && l.Id != s.Id {
				return ServiceSupply{}, ErrSupplyLineExists
			}
		}
	}
	if action == model.ActionCreate {
		if err := m.db.Create(&s); err != nil {
			return ServiceSupply{}, err
		}
		return s, nil
	}
	if err := m.db.Update(&s, orm.Eq(ServiceSupply_.Id, s.Id), orm.Eq(ServiceSupply_.TenantId, s.TenantId)); err != nil {
		return ServiceSupply{}, err
	}
	return s, nil
}

// DeleteServiceSupply removes one recipe line.
func (m *Module) DeleteServiceSupply(tenantId, id string) error {
	s, err := m.getServiceSupply(tenantId, id)
	if err != nil {
		return err
	}
	return m.db.Delete(&s, orm.Eq(ServiceSupply_.Id, s.Id), orm.Eq(ServiceSupply_.TenantId, s.TenantId))
}

// ServiceRecipe is the reader other modules declare in a narrow interface of their own: the supply item ids of a service's
// recipe and the quantity of each, at the same index. An unknown or foreign service is ErrNotFound; a service without
// recipe returns two empty slices.
func (m *Module) ServiceRecipe(tenantId, serviceId string) (itemIds []string, quantities []float64, err error) {
	if _, err := m.GetService(tenantId, serviceId); err != nil {
		return nil, nil, err
	}
	lines, err := m.ListServiceSupplies(tenantId, serviceId)
	if err != nil {
		return nil, nil, err
	}
	itemIds = make([]string, len(lines))
	quantities = make([]float64, len(lines))
	for i, l := range lines {
		itemIds[i], quantities[i] = l.ItemId, l.Quantity
	}
	return itemIds, quantities, nil
}

func (m *Module) opListServiceSupplies(ctx router.Context) {
	var args ListServiceSuppliesArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	items, err := m.ListServiceSupplies(args.TenantId, args.ServiceId)
	if err != nil {
		writeError(ctx, err)
		return
	}
	list := make(ServiceSupplyList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	if err := ctx.Encode(&list); err != nil {
		writeError(ctx, err)
	}
}

func (m *Module) opUpsertServiceSupply(ctx router.Context) {
	var s ServiceSupply
	if err := ctx.Decode(&s); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	out, err := m.UpsertServiceSupply(s)
	if err != nil {
		writeError(ctx, err)
		return
	}
	if err := ctx.Encode(&out); err != nil {
		writeError(ctx, err)
	}
}

func (m *Module) opDeleteServiceSupply(ctx router.Context) {
	var args DeleteServiceSupplyArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if err := m.DeleteServiceSupply(args.TenantId, args.Id); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}
