package itemcatalog

import (
	"github.com/tinywasm/model"
	"github.com/tinywasm/router"
	"github.com/tinywasm/view"
)

// NewView builds the catalog item Presenter — the tech-agnostic engine a renderer (crudview,
// or any other) wraps. It is THIS module's job to build it (importing only view+model+router);
// the app decides which renderer draws it.
func NewView(caller router.Caller) view.Presenter {
	var byID []*CatalogItem
	record := &CatalogItem{}

	return view.New(
		caller,
		record,
		OpListItems,
		func() model.FielderSlice { return &CatalogItemList{} },
		func(list model.FielderSlice) []view.Item {
			l := list.(*CatalogItemList)
			items := make([]view.Item, l.Len())
			byID = make([]*CatalogItem, l.Len())
			for i := 0; i < l.Len(); i++ {
				it := l.At(i).(*CatalogItem)
				byID[i] = it
				items[i] = view.Item{ID: it.Id, Label: it.Name, Description: it.Sku}
			}
			return items
		},
		view.WithTitle("Catálogo"),
		view.WithSaveOp(OpUpsertItem),
		view.WithDeleteOp(OpDeleteItem),
		view.WithFill(func(id string) model.Model {
			if id == "" {
				return nil
			}
			for _, it := range byID {
				if it != nil && it.Id == id {
					return it
				}
			}
			return nil
		}),
	)
}
