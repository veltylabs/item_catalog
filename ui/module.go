package ui

import "webtyp.com/svg"

// ID is this module's identity: RBAC resource prefix on the server, nav
// route on the client. "service", not "service_catalog" — the wire-facing
// name the rest of the app already keys off (RBAC, nav, tests), independent
// of this package's own name.
const ID = "service_catalog"

// Label is the nav item's display text.
const Label = "Catálogo"

const (
	IconCatalog     = svg.Icon("service_catalog")
	IconServices    = svg.Icon("service_catalog_services")
	IconSpecialties = svg.Icon("service_catalog_specialties")
)
