package item_catalog

import (
	"webtyp.com/router"

	. "webtyp.com/dom"
	h "webtyp.com/html"

	itemcatalog "github.com/veltylabs/item_catalog"
)

// specialtyPicker loads the specialty list once and renders a <select> —
// same shape as appointment_booking's own staffPicker, minus FilterFn: no
// screen here needs to narrow the list, only to pick one.
type specialtyPicker struct {
	caller    router.Caller
	sel       *SignalString
	opts      *SignalNodes
	specialty []itemcatalog.Specialty
	onChange  func(id string)
}

func newSpecialtyPicker(caller router.Caller, onChange func(id string)) *specialtyPicker {
	return &specialtyPicker{caller: caller, sel: NewString(""), opts: NewNodes(), onChange: onChange}
}

func (p *specialtyPicker) load(then func()) {
	out := &itemcatalog.SpecialtyList{}
	p.caller.Call(itemcatalog.ModelName+"."+itemcatalog.OpListSpecialties, &itemcatalog.ListSpecialtiesArgs{}, out,
		func(err error) {
			if err != nil {
				return
			}
			p.specialty = make([]itemcatalog.Specialty, 0, len(*out))
			for _, s := range *out {
				p.specialty = append(p.specialty, *s)
			}
			p.rebuildOptions()
			if p.sel.Get() == "" && len(p.specialty) > 0 {
				p.sel.Set(p.specialty[0].Id)
			}
			if then != nil {
				then()
			}
		})
}

func (p *specialtyPicker) rebuildOptions() {
	opts := make([]*Element, 0, len(p.specialty))
	for _, s := range p.specialty {
		if s.Id == p.sel.Get() {
			opts = append(opts, h.SelectedOption(s.Id, s.Name))
		} else {
			opts = append(opts, h.Option(s.Id, s.Name))
		}
	}
	p.opts.Set(opts)
}

func (p *specialtyPicker) onSelectChange(id string) {
	p.sel.Set(id)
	p.rebuildOptions()
	if p.onChange != nil {
		p.onChange(id)
	}
}

func (p *specialtyPicker) render(selectName string) *Element {
	return h.Div().
		Child(h.Label().Text("Especialidad")).
		Child(NewElement("select").Attr("name", selectName).BindChildren(p.opts).
			OnChange(func(e Event) { p.onSelectChange(e.TargetValue()) }))
}
