package itemcatalog

// CatalogService — the core business interface. Implemented by *Module.
type CatalogService interface {
	GetItem(tenantId, id string) (CatalogItem, error)
	FindBySKU(tenantId, sku string) (CatalogItem, error)
	ListItems(tenantId string, filter ItemFilter) ([]CatalogItem, error)
	CreateItem(item CatalogItem) (CatalogItem, error)
	UpdateItem(item CatalogItem) (CatalogItem, error)
	DeactivateItem(tenantId, id string) error
	DeleteItem(tenantId, id string) error
	ServiceExists(tenantId, serviceId string) (bool, error) // implements appointment-booking.CatalogReader

	// Agreement methods
	ListAgreements(tenantId, catalogItemId string) ([]Agreement, error)
	UpsertAgreement(Agreement) (Agreement, error)
	DeleteAgreement(tenantId, id string) error
}
