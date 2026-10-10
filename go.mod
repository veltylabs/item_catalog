module github.com/veltylabs/item_catalog

go 1.26.8

require (
	webtyp.com/ddl v0.0.15
	webtyp.com/dom v0.13.23
	webtyp.com/events v0.0.5
	webtyp.com/fmt v1.0.0
	webtyp.com/form v0.4.30
	webtyp.com/html v0.0.27
	webtyp.com/input v0.0.18
	webtyp.com/json v0.5.29
	webtyp.com/layout v0.3.39
	webtyp.com/model v0.2.2
	webtyp.com/orm v0.12.8
	webtyp.com/router v0.4.1
	webtyp.com/storage v0.1.4
	webtyp.com/svg v0.3.14
	webtyp.com/time v0.5.7
	webtyp.com/unixid v0.3.0
	webtyp.com/view v0.6.27
)

require (
	webtyp.com/color v0.1.2 // indirect
	webtyp.com/components v0.8.16 // indirect
	webtyp.com/css v0.4.29 // indirect
	webtyp.com/date v0.0.9 // indirect
	webtyp.com/escape v0.1.0 // indirect
	webtyp.com/font v0.0.5 // indirect
	webtyp.com/icons v0.0.7 // indirect
	webtyp.com/lang v0.1.3 // indirect
	webtyp.com/msgtype v0.1.0 // indirect
	webtyp.com/widget v0.6.39 // indirect
)

replace (
	webtyp.com/components => ../../../webtyp/components
	webtyp.com/css => ../../../webtyp/css
	webtyp.com/layout => ../../../webtyp/layout
)
