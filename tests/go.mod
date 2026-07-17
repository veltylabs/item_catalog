module github.com/veltylabs/item_catalog/tests

go 1.25.2

replace github.com/veltylabs/item_catalog => ../

require (
	github.com/tinywasm/events v0.0.2
	github.com/tinywasm/fmt v0.25.3
	github.com/tinywasm/model v0.0.16
	github.com/tinywasm/orm v0.11.1
	github.com/tinywasm/router v0.1.15
	github.com/tinywasm/storage v0.0.2
	github.com/veltylabs/item_catalog v0.0.0-00010101000000-000000000000
)

require (
	github.com/tinywasm/ddl v0.0.4 // indirect
	github.com/tinywasm/form v0.2.16 // indirect
	github.com/tinywasm/json v0.5.13 // indirect
	github.com/tinywasm/time v0.5.0 // indirect
	github.com/tinywasm/view v0.1.1 // indirect
)
