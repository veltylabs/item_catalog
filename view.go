package itemcatalog

import (
	"github.com/tinywasm/model"
	"github.com/tinywasm/router"
	"github.com/tinywasm/view"
)

// Item projects a CatalogItem as a view.Item — the ONLY view-specific code this record
// carries (view.Itemizer). The Presenter's internal index (built from this on Reload)
// replaces the old manual byID/WithFill lookup.
func (m *CatalogItem) Item() view.Item {
	return view.Item{ID: m.Id, Label: m.Name, Description: m.Sku}
}

// NewView builds the catalog item Presenter — the tech-agnostic engine a renderer (crudview,
// or any other) wraps. It is THIS module's job to build it (importing only view+model+router);
// the app decides which renderer draws it.
func NewView(caller router.Caller) view.Presenter {
	record := &CatalogItem{}

	return view.New(
		caller,
		record,
		OpListItems,
		func() model.ModelSlice { return &CatalogItemList{} },
		view.WithTitle("Catálogo"),
		view.WithSaveOp(OpUpsertItem),
		view.WithDeleteOp(OpDeleteItem),
	)
}
