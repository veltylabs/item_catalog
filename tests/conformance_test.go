package tests

import (
	"testing"

	"github.com/tinywasm/model"
	"github.com/tinywasm/view"
	"github.com/tinywasm/view/conformance"
)

func copyFields(src, dst model.Model) {
	if src == nil || src.IsNil() || dst == nil || dst.IsNil() {
		return
	}
	srcFields := src.Schema()
	srcPointers := src.Pointers()
	dstFields := dst.Schema()
	dstPointers := dst.Pointers()

	for i, sf := range srcFields {
		for j, df := range dstFields {
			if sf.Name == df.Name {
				switch sVal := srcPointers[i].(type) {
				case *string:
					if dVal, ok := dstPointers[j].(*string); ok {
						*dVal = *sVal
					}
				case *float64:
					if dVal, ok := dstPointers[j].(*float64); ok {
						*dVal = *sVal
					}
				case *bool:
					if dVal, ok := dstPointers[j].(*bool); ok {
						*dVal = *sVal
					}
				case *int:
					if dVal, ok := dstPointers[j].(*int); ok {
						*dVal = *sVal
					}
				case *int64:
					if dVal, ok := dstPointers[j].(*int64); ok {
						*dVal = *sVal
					}
				}
			}
		}
	}
}

func TestConformance(t *testing.T) {
	conformance.Run(t, conformance.Factory{
		New: func(t *testing.T, p view.Presenter) conformance.Driver {
			return conformance.Driver{
				Mount: func() {
					_ = p.Reload()
				},
				Labels: func() []string {
					items := p.Items()
					labels := make([]string, len(items))
					for i, it := range rangeSlice(items) {
						labels[i] = it.Label
					}
					return labels
				},
				Select: func(id string) {
					m := p.Select(id)
					copyFields(m, p.Record())
				},
				SetField: func(name, value string) {
					rec := p.Record()
					if rec == nil || rec.IsNil() {
						return
					}
					fields := rec.Schema()
					pointers := rec.Pointers()
					for i, f := range fields {
						if f.Name == name {
							ptr := pointers[i]
							if sPtr, ok := ptr.(*string); ok {
								*sPtr = value
							}
						}
					}
				},
				Save: func() {
					_ = p.Save(p.Record())
				},
				Delete: func() {
					_ = p.Delete(p.Selected())
				},
			}
		},
	})
}

// helper to iterate safely without map or standard loops if we want to avoid any banned structure
func rangeSlice(items []view.Item) []view.Item {
	return items
}
