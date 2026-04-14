# Database Schema

```mermaid
erDiagram
    catalog_item {
        string id PK
        string tenant_id
        string sku
        string name
        string description
        string type
        float price
        string currency
        bool is_active
        int updated_at
    }
```
