package ui

// ID is this module's identity: RBAC resource prefix on the server, nav
// route on the client. "catalog_item", not "item_catalog" — the wire-facing
// name the rest of the app already keys off (RBAC, nav, tests), independent
// of this package's own name.
const ID = "catalog_item"

// Label is the nav item's display text.
const Label = "Catálogo"
