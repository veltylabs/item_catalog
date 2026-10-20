package servicecatalog

import (
	"webtyp.com/events"
	"webtyp.com/fmt"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/router"
	"webtyp.com/time"
)

type domainError string

func (e domainError) Error() string { return string(e) }

const (
	ErrNotFound              domainError = "service not found"
	ErrAlreadyExists         domainError = "service already exists"
	ErrSpecialtyNotFound     domainError = "specialty not found"
	ErrSpecialtyInUse        domainError = "specialty in use"
	ErrSpecialtyPrefixExists domainError = "specialty prefix already exists"
	ErrSpecialtySlugExists   domainError = "specialty slug already exists"
	ErrNoRecords             domainError = "service_catalog: no records to save"
	ErrNoIds                 domainError = "service_catalog: no ids given"
)

// decodeError is a request body that could not be read: always 400.
type decodeError struct{ err error }

func (d decodeError) Error() string { return msgDecode + d.err.Error() }

const msgDecode = "service_catalog: cannot read the request: "

// writeError answers err with its status and, always, its text — the only thing an HTTP caller
// sees. 400 = unreadable or invalid input · 404 = not found · 409 = conflict · 500 = anything else.
func writeError(ctx router.Context, err error) {
	ctx.WriteStatus(statusOf(err))
	ctx.Write([]byte(err.Error()))
}

func statusOf(err error) int {
	switch e := err.(type) {
	case decodeError, ValidationError:
		return 400
	case domainError:
		switch e {
		case ErrNotFound, ErrSpecialtyNotFound:
			return 404
		case ErrAlreadyExists, ErrSpecialtyPrefixExists, ErrSpecialtySlugExists:
			return 409
		case ErrSpecialtyInUse, ErrNoRecords, ErrNoIds:
			return 400
		}
	}
	if orm.IsNotFound(err) {
		return 404
	}
	return 500
}

const (
	OpListSpecialties = "list_specialties"
	OpGetSpecialty    = "get_specialty"
	OpUpsertSpecialty = "upsert_specialty"
	OpDeleteSpecialty = "delete_specialty"

	OpListServices      = "list_services"
	OpGetService        = "get_service"
	OpFindServiceBySKU  = "find_service_by_sku"
	OpCreateService     = "create_service"
	OpUpdateService     = "update_service"
	OpUpsertService     = "upsert_service"
	OpDeactivateService = "deactivate_service"
	OpDeleteService     = "delete_service"

	OpListAgreements  = "list_agreements"
	OpUpsertAgreement = "upsert_agreement"
	OpDeleteAgreement = "delete_agreement"

	TopicSpecialtyCreated = "catalog.specialty.created"
	TopicSpecialtyUpdated = "catalog.specialty.updated"
	TopicSpecialtyDeleted = "catalog.specialty.deleted"

	TopicServiceCreated          = "catalog.service.created"
	TopicServiceUpdated          = "catalog.service.updated"
	TopicServiceDeactivated      = "catalog.service.deactivated"
	TopicServiceDeleted          = "catalog.service.deleted"
	TopicServiceAgreementCreated = "catalog.service_agreement.created"
	TopicServiceAgreementUpdated = "catalog.service_agreement.updated"
	TopicServiceAgreementDeleted = "catalog.service_agreement.deleted"
)

type ValidationError struct {
	Err error
}

func (v ValidationError) Error() string {
	return v.Err.Error()
}

type Deps struct {
	IDs       model.IDGenerator // requerido — el módulo NUNCA construye un generador
	Publisher events.Publisher  // opcional — nil desactiva la publicación de eventos
}

type Module struct {
	db  *orm.DB
	ids model.IDGenerator
	pub events.Publisher
}

func New(db *orm.DB, deps Deps) (*Module, error) {
	if deps.IDs == nil {
		return nil, fmt.Err("service_catalog: Deps.IDs is required")
	}
	return &Module{db: db, ids: deps.IDs, pub: deps.Publisher}, nil
}

// Specialty Service methods

func (m *Module) GetSpecialty(tenantId, id string) (Specialty, error) {
	var spec Specialty
	qb := m.db.Query(&spec).Where(Specialty_.Id).Eq(id)
	if tenantId != "" {
		qb = qb.Where(Specialty_.TenantId).Eq(tenantId)
	}
	_, err := ReadOneSpecialty(qb, &spec)
	if err != nil {
		if orm.IsNotFound(err) {
			return Specialty{}, ErrSpecialtyNotFound
		}
		return Specialty{}, err
	}
	return spec, nil
}

func (m *Module) GetSpecialtyByPrefix(tenantId, prefix string) (Specialty, error) {
	var spec Specialty
	qb := m.db.Query(&spec).Where(Specialty_.Prefix).Eq(prefix).Where(Specialty_.TenantId).Eq(tenantId)
	_, err := ReadOneSpecialty(qb, &spec)
	if err != nil {
		if orm.IsNotFound(err) {
			return Specialty{}, ErrSpecialtyNotFound
		}
		return Specialty{}, err
	}
	return spec, nil
}

func (m *Module) GetSpecialtyBySlug(tenantId, slug string) (Specialty, error) {
	var spec Specialty
	qb := m.db.Query(&spec).Where(Specialty_.Slug).Eq(slug).Where(Specialty_.TenantId).Eq(tenantId)
	_, err := ReadOneSpecialty(qb, &spec)
	if err != nil {
		if orm.IsNotFound(err) {
			return Specialty{}, ErrSpecialtyNotFound
		}
		return Specialty{}, err
	}
	return spec, nil
}

func (m *Module) ListSpecialties(tenantId string) ([]Specialty, error) {
	var spec Specialty
	qb := m.db.Query(&spec).Where(Specialty_.TenantId).Eq(tenantId)
	results, err := ReadAllSpecialty(qb)
	if err != nil {
		return nil, err
	}
	list := make([]Specialty, len(results))
	for i, r := range results {
		list[i] = *r
	}
	return list, nil
}

func (m *Module) UpsertSpecialty(spec Specialty) (Specialty, error) {
	action := model.ActionCreate
	if spec.Id != "" {
		action = model.ActionUpdate
	}
	if spec.Id == "" {
		spec.Id = m.ids.NewID()
	}
	spec.UpdatedAt = time.Now()

	if err := spec.Validate(action); err != nil {
		return Specialty{}, ValidationError{Err: err}
	}

	// Validate prefix uniqueness per tenant
	existingPrefix, err := m.GetSpecialtyByPrefix(spec.TenantId, spec.Prefix)
	if err == nil {
		if existingPrefix.Id != spec.Id {
			return Specialty{}, ErrSpecialtyPrefixExists
		}
	} else if e, ok := err.(domainError); !ok || e != ErrSpecialtyNotFound {
		return Specialty{}, err
	}

	// Validate slug uniqueness per tenant
	existingSlug, err := m.GetSpecialtyBySlug(spec.TenantId, spec.Slug)
	if err == nil {
		if existingSlug.Id != spec.Id {
			return Specialty{}, ErrSpecialtySlugExists
		}
	} else if e, ok := err.(domainError); !ok || e != ErrSpecialtyNotFound {
		return Specialty{}, err
	}

	if action == model.ActionCreate {
		if err := m.db.Create(&spec); err != nil {
			return Specialty{}, err
		}
		if m.pub != nil {
			m.pub.Publish(events.Event{Topic: TopicSpecialtyCreated, Payload: &spec})
		}
		return spec, nil
	}

	// Verify specialty exists and belongs to tenant
	_, err = m.GetSpecialty(spec.TenantId, spec.Id)
	if err != nil {
		if orm.IsNotFound(err) || err == ErrSpecialtyNotFound {
			if err := m.db.Create(&spec); err != nil {
				return Specialty{}, err
			}
			if m.pub != nil {
				m.pub.Publish(events.Event{Topic: TopicSpecialtyCreated, Payload: &spec})
			}
			return spec, nil
		}
		return Specialty{}, err
	}

	if err := m.db.Update(&spec, orm.Eq(Specialty_.Id, spec.Id), orm.Eq(Specialty_.TenantId, spec.TenantId)); err != nil {
		return Specialty{}, err
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicSpecialtyUpdated, Payload: &spec})
	}
	return spec, nil
}

func (m *Module) DeleteSpecialty(tenantId, id string) error {
	spec, err := m.GetSpecialty(tenantId, id)
	if err != nil {
		return err
	}

	// Check if any catalog.service is using this specialty
	var item Service
	qb := m.db.Query(&item).Where(Service_.SpecialtyId).Eq(id)
	if spec.TenantId != "" {
		qb = qb.Where(Service_.TenantId).Eq(spec.TenantId)
	}
	items, err := ReadAllService(qb)
	if err != nil {
		return err
	}
	if len(items) > 0 {
		return ErrSpecialtyInUse
	}

	if spec.TenantId != "" {
		if err := m.db.Delete(&spec, orm.Eq(Specialty_.Id, id), orm.Eq(Specialty_.TenantId, spec.TenantId)); err != nil {
			return err
		}
	} else {
		if err := m.db.Delete(&spec, orm.Eq(Specialty_.Id, id)); err != nil {
			return err
		}
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicSpecialtyDeleted, Payload: &spec})
	}
	return nil
}

// Service methods

func (m *Module) GetService(tenantId, id string) (Service, error) {
	var item Service
	qb := m.db.Query(&item).Where(Service_.Id).Eq(id)
	if tenantId != "" {
		qb = qb.Where(Service_.TenantId).Eq(tenantId)
	}
	_, err := ReadOneService(qb, &item)
	if err != nil {
		if orm.IsNotFound(err) {
			return Service{}, ErrNotFound
		}
		return Service{}, err
	}
	return item, nil
}

func (m *Module) FindBySKU(tenantId, sku string) (Service, error) {
	var item Service
	qb := m.db.Query(&item).Where(Service_.Sku).Eq(sku).Where(Service_.TenantId).Eq(tenantId)
	_, err := ReadOneService(qb, &item)
	if err != nil {
		if orm.IsNotFound(err) {
			return Service{}, ErrNotFound
		}
		return Service{}, err
	}
	return item, nil
}

func (m *Module) ListServices(tenantId string, filter ServiceFilter) ([]Service, error) {
	var item Service
	qb := m.db.Query(&item).Where(Service_.TenantId).Eq(tenantId)
	if filter.SpecialtyId != "" {
		qb = qb.Where(Service_.SpecialtyId).Eq(filter.SpecialtyId)
	}

	if filter.ActiveOnly {
		qb = qb.Where(Service_.IsActive).Eq(true)
	}
	if filter.Limit > 0 {
		qb = qb.Limit(int(filter.Limit))
	}
	if filter.Offset > 0 {
		qb = qb.Offset(int(filter.Offset))
	}
	results, err := ReadAllService(qb)
	if err != nil {
		return nil, err
	}
	items := make([]Service, len(results))
	for i, r := range results {
		items[i] = *r
	}
	return items, nil
}

func (m *Module) CreateService(item Service) (Service, error) {
	if item.Id == "" {
		item.Id = m.ids.NewID()
	}
	item.UpdatedAt = time.Now()

	if err := item.Validate(model.ActionCreate); err != nil {
		return Service{}, ValidationError{Err: err}
	}

	// Validate SKU uniqueness per tenant
	existing, err := m.FindBySKU(item.TenantId, item.Sku)
	if err == nil {
		if existing.Id != "" {
			return Service{}, ErrAlreadyExists
		}
	} else if e, ok := err.(domainError); !ok || e != ErrNotFound {
		return Service{}, err
	}

	if err := m.db.Create(&item); err != nil {
		return Service{}, err
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicServiceCreated, Payload: &item})
	}
	return item, nil
}

func (m *Module) UpdateService(item Service) (Service, error) {
	if err := item.Validate(model.ActionUpdate); err != nil {
		return Service{}, ValidationError{Err: err}
	}

	// Verify item exists and belongs to tenant
	existing, err := m.GetService(item.TenantId, item.Id)
	if err != nil {
		return Service{}, err
	}
	if item.TenantId == "" {
		item.TenantId = existing.TenantId
	}

	item.UpdatedAt = time.Now()
	if err := m.db.Update(&item, orm.Eq(Service_.Id, item.Id), orm.Eq(Service_.TenantId, item.TenantId)); err != nil {
		return Service{}, err
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicServiceUpdated, Payload: &item})
	}
	return item, nil
}

func (m *Module) DeactivateService(tenantId, id string) error {
	item, err := m.GetService(tenantId, id)
	if err != nil {
		return err
	}
	item.IsActive = false
	item.UpdatedAt = time.Now()
	if err := m.db.Update(&item, orm.Eq(Service_.Id, item.Id), orm.Eq(Service_.TenantId, item.TenantId)); err != nil {
		return err
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicServiceDeactivated, Payload: &item})
	}
	return nil
}

func (m *Module) DeleteService(tenantId, id string) error {
	item, err := m.GetService(tenantId, id)
	if err != nil {
		return err
	}
	if item.TenantId != "" {
		if err := m.db.Delete(&item, orm.Eq(Service_.Id, item.Id), orm.Eq(Service_.TenantId, item.TenantId)); err != nil {
			return err
		}
	} else {
		if err := m.db.Delete(&item, orm.Eq(Service_.Id, item.Id)); err != nil {
			return err
		}
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicServiceDeleted, Payload: &item})
	}
	return nil
}

func (m *Module) ServiceExists(tenantId, serviceId string) (bool, error) {
	item, err := m.GetService(tenantId, serviceId)
	if err != nil {
		if e, ok := err.(domainError); ok && e == ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return item.IsActive, nil
}

// ServiceCard devuelve nombre, precio de lista, moneda y estado de un servicio
// del catálogo. Satisface CatalogReader.ServiceCard de appointment_booking
// estructuralmente. Ausente, de otro tenant o que no es servicio ⇒ ErrNotFound.
func (m *Module) ServiceCard(tenantId, serviceId string) (name string, price float64, currency string, active bool, err error) {
	if tenantId == "" || serviceId == "" {
		return "", 0, "", false, ErrNotFound
	}
	item, e := m.GetService(tenantId, serviceId)
	if e != nil {
		return "", 0, "", false, e
	}

	return item.Name, item.Price, item.Currency, item.IsActive, nil
}

func (m *Module) ListServiceAgreements(tenantId, serviceId string) ([]ServiceAgreement, error) {
	var a ServiceAgreement
	qb := m.db.Query(&a).Where(ServiceAgreement_.TenantId).Eq(tenantId)
	if serviceId != "" {
		qb = qb.Where(ServiceAgreement_.ServiceId).Eq(serviceId)
	}
	results, err := ReadAllServiceAgreement(qb)
	if err != nil {
		return nil, err
	}
	items := make([]ServiceAgreement, len(results))
	for i, r := range results {
		items[i] = *r
	}
	return items, nil
}

func (m *Module) GetServiceAgreement(tenantId, id string) (ServiceAgreement, error) {
	var a ServiceAgreement
	qb := m.db.Query(&a).Where(ServiceAgreement_.Id).Eq(id).Where(ServiceAgreement_.TenantId).Eq(tenantId)
	_, err := ReadOneServiceAgreement(qb, &a)
	if err != nil {
		if orm.IsNotFound(err) {
			return ServiceAgreement{}, ErrNotFound
		}
		return ServiceAgreement{}, err
	}
	return a, nil
}

func (m *Module) UpsertServiceAgreement(a ServiceAgreement) (ServiceAgreement, error) {
	action := model.ActionCreate
	if a.Id != "" {
		action = model.ActionUpdate
	}
	if a.Id == "" {
		a.Id = m.ids.NewID()
	}
	a.UpdatedAt = time.Now()

	if err := a.Validate(action); err != nil {
		return ServiceAgreement{}, ValidationError{Err: err}
	}

	if action == model.ActionCreate {
		if err := m.db.Create(&a); err != nil {
			return ServiceAgreement{}, err
		}
		if m.pub != nil {
			m.pub.Publish(events.Event{Topic: TopicServiceAgreementCreated, Payload: &a})
		}
		return a, nil
	}

	// Verify agreement exists and belongs to tenant
	_, err := m.GetServiceAgreement(a.TenantId, a.Id)
	if err != nil {
		return ServiceAgreement{}, err
	}

	if err := m.db.Update(&a, orm.Eq(ServiceAgreement_.Id, a.Id), orm.Eq(ServiceAgreement_.TenantId, a.TenantId)); err != nil {
		return ServiceAgreement{}, err
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicServiceAgreementUpdated, Payload: &a})
	}
	return a, nil
}

func (m *Module) DeleteServiceAgreement(tenantId, id string) error {
	a, err := m.GetServiceAgreement(tenantId, id)
	if err != nil {
		return err
	}

	if err := m.db.Delete(&a, orm.Eq(ServiceAgreement_.Id, id), orm.Eq(ServiceAgreement_.TenantId, tenantId)); err != nil {
		return err
	}
	if m.pub != nil {
		m.pub.Publish(events.Event{Topic: TopicServiceAgreementDeleted, Payload: &a})
	}
	return nil
}

// ModelName is this module's identity: mcp.HarvestOps qualifies every op
// below as "service_catalog.<name>" on the wire, and view.go's two NewXView
// constructors pass this same constant as view.Ops.Module so the client
// composes the identical qualified name.
const ModelName = "service_catalog"

func (m *Module) ModelName() string { return ModelName }

// UpsertSpecialtyArgs carries the plural batch for OpUpsertSpecialty.
type UpsertSpecialtyArgs struct {
	Records []Specialty
}

func (a *UpsertSpecialtyArgs) ModelName() string     { return "upsert_specialty_args" }
func (a *UpsertSpecialtyArgs) Schema() []model.Field { return nil }
func (a *UpsertSpecialtyArgs) Pointers() []any       { return nil }
func (a *UpsertSpecialtyArgs) IsNil() bool           { return a == nil }
func (a *UpsertSpecialtyArgs) EncodeFields(w model.FieldWriter) {
	arr := w.Array("records", len(a.Records))
	for i := range a.Records {
		arr.Object(&a.Records[i])
	}
	arr.Close()
}
func (a *UpsertSpecialtyArgs) DecodeFields(r model.FieldReader) {
	if arr, ok := r.Array("records"); ok {
		a.Records = make([]Specialty, arr.Len())
		for i := 0; i < arr.Len(); i++ {
			arr.Object(i, &a.Records[i])
		}
	}
}

// DeleteSpecialtiesArgs carries the IDs to remove for OpDeleteSpecialty.
type DeleteSpecialtiesArgs struct {
	Ids []string
}

func (a *DeleteSpecialtiesArgs) ModelName() string     { return "delete_specialties_args" }
func (a *DeleteSpecialtiesArgs) Schema() []model.Field { return nil }
func (a *DeleteSpecialtiesArgs) Pointers() []any       { return nil }
func (a *DeleteSpecialtiesArgs) IsNil() bool           { return a == nil }
func (a *DeleteSpecialtiesArgs) EncodeFields(w model.FieldWriter) {
	arr := w.Array("ids", len(a.Ids))
	for _, id := range a.Ids {
		arr.String(id)
	}
	arr.Close()
}
func (a *DeleteSpecialtiesArgs) DecodeFields(r model.FieldReader) {
	if arr, ok := r.Array("ids"); ok {
		a.Ids = make([]string, arr.Len())
		for i := 0; i < arr.Len(); i++ {
			a.Ids[i] = arr.String(i)
		}
	}
}

// UpsertServiceArgs carries the plural batch for OpUpsertService.
type UpsertServiceArgs struct {
	Records []Service
}

func (a *UpsertServiceArgs) ModelName() string     { return "upsert_item_args" }
func (a *UpsertServiceArgs) Schema() []model.Field { return nil }
func (a *UpsertServiceArgs) Pointers() []any       { return nil }
func (a *UpsertServiceArgs) IsNil() bool           { return a == nil }
func (a *UpsertServiceArgs) EncodeFields(w model.FieldWriter) {
	arr := w.Array("records", len(a.Records))
	for i := range a.Records {
		arr.Object(&a.Records[i])
	}
	arr.Close()
}
func (a *UpsertServiceArgs) DecodeFields(r model.FieldReader) {
	if arr, ok := r.Array("records"); ok {
		a.Records = make([]Service, arr.Len())
		for i := 0; i < arr.Len(); i++ {
			arr.Object(i, &a.Records[i])
		}
	}
}

// DeleteServicesArgs carries the IDs to remove for OpDeleteService.
type DeleteServicesArgs struct {
	Ids []string
}

func (a *DeleteServicesArgs) ModelName() string     { return "delete_items_args" }
func (a *DeleteServicesArgs) Schema() []model.Field { return nil }
func (a *DeleteServicesArgs) Pointers() []any       { return nil }
func (a *DeleteServicesArgs) IsNil() bool           { return a == nil }
func (a *DeleteServicesArgs) EncodeFields(w model.FieldWriter) {
	arr := w.Array("ids", len(a.Ids))
	for _, id := range a.Ids {
		arr.String(id)
	}
	arr.Close()
}
func (a *DeleteServicesArgs) DecodeFields(r model.FieldReader) {
	if arr, ok := r.Array("ids"); ok {
		a.Ids = make([]string, arr.Len())
		for i := 0; i < arr.Len(); i++ {
			a.Ids[i] = arr.String(i)
		}
	}
}

// UpdateServiceArgs carries the field patch for OpUpdateService.
type UpdateServiceArgs struct {
	Ids    []string
	Fields []string
	Record Service
}

func (a *UpdateServiceArgs) ModelName() string     { return "update_item_args" }
func (a *UpdateServiceArgs) Schema() []model.Field { return nil }
func (a *UpdateServiceArgs) Pointers() []any       { return nil }
func (a *UpdateServiceArgs) IsNil() bool           { return a == nil }
func (a *UpdateServiceArgs) EncodeFields(w model.FieldWriter) {
	ids := w.Array("ids", len(a.Ids))
	for _, id := range a.Ids {
		ids.String(id)
	}
	ids.Close()
	fields := w.Array("fields", len(a.Fields))
	for _, f := range a.Fields {
		fields.String(f)
	}
	fields.Close()
	w.Object("record", &a.Record)
}
func (a *UpdateServiceArgs) DecodeFields(r model.FieldReader) {
	if arr, ok := r.Array("ids"); ok {
		a.Ids = make([]string, arr.Len())
		for i := 0; i < arr.Len(); i++ {
			a.Ids[i] = arr.String(i)
		}
	}
	if arr, ok := r.Array("fields"); ok {
		a.Fields = make([]string, arr.Len())
		for i := 0; i < arr.Len(); i++ {
			a.Fields[i] = arr.String(i)
		}
	}
	r.Object("record", &a.Record)
}

func (m *Module) MountOperations(reg router.OperationRegistry) {
	reg.Operation(OpListSpecialties, m.opListSpecialties).Requires("specialty", model.Read).Accepts(&ListSpecialtiesArgs{})
	reg.Operation(OpGetSpecialty, m.opGetSpecialty).Requires("specialty", model.Read).Accepts(&GetSpecialtyArgs{})
	reg.Operation(OpUpsertSpecialty, m.opUpsertSpecialty).Requires("specialty", model.Create|model.Update).Accepts(&UpsertSpecialtyArgs{})
	reg.Operation(OpDeleteSpecialty, m.opDeleteSpecialty).Requires("specialty", model.Delete).Accepts(&DeleteSpecialtiesArgs{})

	reg.Operation(OpListServices, m.opListServices).Requires("service", model.Read).Accepts(&ListServicesArgs{})
	reg.Operation(OpGetService, m.opGetService).Requires("service", model.Read).Accepts(&GetServiceArgs{})
	reg.Operation(OpFindServiceBySKU, m.opFindServiceBySKU).Requires("service", model.Read).Accepts(&FindServiceBySKUArgs{})
	reg.Operation(OpCreateService, m.opCreateService).Requires("service", model.Create).Accepts(&Service{})
	reg.Operation(OpUpdateService, m.opUpdateService).Requires("service", model.Update).Accepts(&UpdateServiceArgs{})
	reg.Operation(OpUpsertService, m.opUpsertService).Requires("service", model.Create|model.Update).Accepts(&UpsertServiceArgs{})
	reg.Operation(OpDeactivateService, m.opDeactivateService).Requires("service", model.Update).Accepts(&DeactivateServiceArgs{})
	reg.Operation(OpDeleteService, m.opDeleteService).Requires("service", model.Delete).Accepts(&DeleteServicesArgs{})

	reg.Operation(OpListAgreements, m.opListServiceAgreements).Requires("service_agreement", model.Read).Accepts(&ListAgreementsArgs{})
	reg.Operation(OpUpsertAgreement, m.opUpsertServiceAgreement).Requires("service_agreement", model.Create|model.Update).Accepts(&ServiceAgreement{})
	reg.Operation(OpDeleteAgreement, m.opDeleteServiceAgreement).Requires("service_agreement", model.Delete).Accepts(&DeleteAgreementArgs{})
}

var _ router.OperationModule = (*Module)(nil)

func (m *Module) opListSpecialties(ctx router.Context) {
	var args ListSpecialtiesArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	list, err := m.ListSpecialties(args.TenantId)
	if err != nil {
		writeError(ctx, err)
		return
	}
	resList := make(SpecialtyList, len(list))
	for i := range list {
		resList[i] = &list[i]
	}
	if err := ctx.Encode(&resList); err != nil {
		writeError(ctx, err)
	}
}

func (m *Module) opGetSpecialty(ctx router.Context) {
	var args GetSpecialtyArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	spec, err := m.GetSpecialty(args.TenantId, args.Id)
	if err != nil {
		writeError(ctx, err)
		return
	}
	if err := ctx.Encode(&spec); err != nil {
		writeError(ctx, err)
	}
}

func (m *Module) opUpsertSpecialty(ctx router.Context) {
	var args UpsertSpecialtyArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Records) == 0 {
		writeError(ctx, ErrNoRecords)
		return
	}
	var last Specialty
	for _, spec := range args.Records {
		out, err := m.UpsertSpecialty(spec)
		if err != nil {
			writeError(ctx, err)
			return
		}
		last = out
	}
	if err := ctx.Encode(&last); err != nil {
		writeError(ctx, err)
	}
}

func (m *Module) opDeleteSpecialty(ctx router.Context) {
	var args DeleteSpecialtiesArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Ids) == 0 {
		writeError(ctx, ErrNoIds)
		return
	}
	for _, id := range args.Ids {
		if err := m.DeleteSpecialty("", id); err != nil {
			writeError(ctx, err)
			return
		}
	}
	ctx.WriteStatus(200)
}

func (m *Module) opListServices(ctx router.Context) {
	var args ListServicesArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	filter := ServiceFilter{SpecialtyId: args.SpecialtyId, ActiveOnly: args.ActiveOnly, Limit: args.Limit, Offset: args.Offset}
	items, err := m.ListServices(args.TenantId, filter)
	if err != nil {
		writeError(ctx, err)
		return
	}
	list := make(ServiceList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	if err := ctx.Encode(&list); err != nil {
		writeError(ctx, err)
	}
}

func (m *Module) opGetService(ctx router.Context) {
	var args GetServiceArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	item, err := m.GetService(args.TenantId, args.Id)
	if err != nil {
		writeError(ctx, err)
		return
	}
	if err := ctx.Encode(&item); err != nil {
		writeError(ctx, err)
	}
}

func (m *Module) opFindServiceBySKU(ctx router.Context) {
	var args FindServiceBySKUArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	item, err := m.FindBySKU(args.TenantId, args.Sku)
	if err != nil {
		writeError(ctx, err)
		return
	}
	if err := ctx.Encode(&item); err != nil {
		writeError(ctx, err)
	}
}

func (m *Module) opCreateService(ctx router.Context) {
	var item Service
	if err := ctx.Decode(&item); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	created, err := m.CreateService(item)
	if err != nil {
		writeError(ctx, err)
		return
	}
	if err := ctx.Encode(&created); err != nil {
		writeError(ctx, err)
	}
}

func (m *Module) opUpdateService(ctx router.Context) {
	var args UpdateServiceArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Ids) == 0 {
		writeError(ctx, ErrNoIds)
		return
	}
	for _, id := range args.Ids {
		existing, err := m.GetService("", id)
		if err != nil {
			writeError(ctx, err)
			return
		}
		for _, f := range args.Fields {
			switch f {
			case "name":
				existing.Name = args.Record.Name
			case "description":
				existing.Description = args.Record.Description
			case "price":
				existing.Price = args.Record.Price
			case "sku":
				existing.Sku = args.Record.Sku
			case "is_active":
				existing.IsActive = args.Record.IsActive
			case "specialty_id":
				existing.SpecialtyId = args.Record.SpecialtyId
			}
		}
		if _, err := m.UpdateService(existing); err != nil {
			writeError(ctx, err)
			return
		}
	}
	ctx.WriteStatus(200)
}

func (m *Module) opUpsertService(ctx router.Context) {
	var args UpsertServiceArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Records) == 0 {
		writeError(ctx, ErrNoRecords)
		return
	}
	var last Service
	for _, item := range args.Records {
		var out Service
		var err error
		if item.Id == "" {
			out, err = m.CreateService(item)
		} else {
			out, err = m.UpdateService(item)
			if err != nil && (err == ErrNotFound || orm.IsNotFound(err)) {
				out, err = m.CreateService(item)
			}
		}
		if err != nil {
			writeError(ctx, err)
			return
		}
		last = out
	}
	if err := ctx.Encode(&last); err != nil {
		writeError(ctx, err)
	}
}

func (m *Module) opDeactivateService(ctx router.Context) {
	var args DeactivateServiceArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if err := m.DeactivateService(args.TenantId, args.Id); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}

func (m *Module) opDeleteService(ctx router.Context) {
	var args DeleteServicesArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if len(args.Ids) == 0 {
		writeError(ctx, ErrNoIds)
		return
	}
	for _, id := range args.Ids {
		if err := m.DeleteService("", id); err != nil {
			writeError(ctx, err)
			return
		}
	}
	ctx.WriteStatus(200)
}

func (m *Module) opListServiceAgreements(ctx router.Context) {
	var args ListAgreementsArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	items, err := m.ListServiceAgreements(args.TenantId, args.ServiceId)
	if err != nil {
		writeError(ctx, err)
		return
	}
	list := make(ServiceAgreementList, len(items))
	for i := range items {
		list[i] = &items[i]
	}
	if err := ctx.Encode(&list); err != nil {
		writeError(ctx, err)
	}
}

func (m *Module) opUpsertServiceAgreement(ctx router.Context) {
	var a ServiceAgreement
	if err := ctx.Decode(&a); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	out, err := m.UpsertServiceAgreement(a)
	if err != nil {
		writeError(ctx, err)
		return
	}
	if err := ctx.Encode(&out); err != nil {
		writeError(ctx, err)
	}
}

func (m *Module) opDeleteServiceAgreement(ctx router.Context) {
	var args DeleteAgreementArgs
	if err := ctx.Decode(&args); err != nil {
		writeError(ctx, decodeError{err})
		return
	}
	if err := m.DeleteServiceAgreement(args.TenantId, args.Id); err != nil {
		writeError(ctx, err)
		return
	}
	ctx.WriteStatus(200)
}
