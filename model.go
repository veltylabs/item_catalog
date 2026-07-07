package itemcatalog

// orm:typed_fields
// CatalogItem represents a service (S) or product (P) in the catalog.
type CatalogItem struct {
	ID          string  `db:"pk"` // unixid — timestamp-ordered
	TenantID    string  `db:"not_null"`
	SKU         string  `db:"not_null"` // unique per tenant — enforced in service layer
	Name        string  `db:"not_null"`
	Description string  // optional — aids LLM context in MCP tools
	Category    string  // optional grouping label (e.g. "Columna", "Ginecología") — free text, not enforced
	Type        string  `db:"not_null"` // "S" = Service | "P" = Product
	Price       float64 `db:"not_null"` // base price (can be overridden in appointment-booking)
	Currency    string  `db:"not_null"` // ISO 4217: "CLP", "USD", etc.
	IsActive    bool    `db:"not_null"` // false = soft-deleted; preserved for referential integrity
	UpdatedAt   int64   `db:"not_null"` // Unix UTC — managed by service layer before db.Create/Update
}

// UIAdapter — port for presentation. Implemented by the host app.
// The module calls these methods; it never imports tinywasm/dom or components directly.
type UIAdapter interface {
	RenderItemList(items []CatalogItem, activeFilter string) string
	RenderItemForm(item *CatalogItem) string // nil = empty create form
	RenderFilterSelector(current string) string
}

// EventPublisher — compatible with tinywasm/sse but not coupled to it.
// Pass nil to disable event publishing.
type EventPublisher interface {
	Publish(event string, payload any) error
}

// CatalogService — the core business interface. Implemented by *Module.
type CatalogService interface {
	GetItem(tenantID, id string) (CatalogItem, error)
	FindBySKU(tenantID, sku string) (CatalogItem, error)
	ListItems(tenantID string, filter ItemFilter) ([]CatalogItem, error)
	CreateItem(item CatalogItem) (CatalogItem, error)
	UpdateItem(item CatalogItem) (CatalogItem, error)
	DeactivateItem(tenantID, id string) error
	DeleteItem(tenantID, id string) error
	ServiceExists(tenantID, serviceID string) (bool, error) // implements appointment-booking.CatalogReader
}

// ItemFilter for list queries.
type ItemFilter struct {
	Type       string // "S" | "P" | "" (all)
	ActiveOnly bool
	Limit      int
	Offset     int
}

// ListItemsArgs — arguments for list items MCP tool.
type ListItemsArgs struct {
	TenantID   string
	Type       string
	ActiveOnly bool
	Limit      int
	Offset     int
}

// GetItemArgs — arguments for get item MCP tool.
type GetItemArgs struct {
	TenantID string
	ID       string
}

// FindBySKUArgs — arguments for find by SKU MCP tool.
type FindBySKUArgs struct {
	TenantID string
	SKU      string
}

// DeactivateItemArgs — arguments for deactivate item MCP tool.
type DeactivateItemArgs struct {
	TenantID string
	ID       string
}

// DeleteItemArgs — arguments for delete item MCP tool.
type DeleteItemArgs struct {
	TenantID string
	ID       string
}
