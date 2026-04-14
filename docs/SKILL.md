# Item Catalog Skill

The `item-catalog` module allows you to manage a list of services and products.

## Key Capabilities
- **Search**: Find items by SKU or list all items for a tenant.
- **Manage**: Create, update, deactivate, or delete items.
- **Categorization**: Items are either Services (`S`) or Products (`P`).

## Data Structure
Items have a `price` and a `currency` (ISO 4217). The `SKU` must be unique within a tenant.

## Example Usage
To create a new service:
1. Call `create_catalog_item` with `type: "S"`.
2. Provide a unique `sku`.
3. Specify `price` and `currency`.
