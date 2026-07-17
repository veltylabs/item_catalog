package tests

import (
	"testing"

	"github.com/tinywasm/model"
	itemcatalog "github.com/veltylabs/item_catalog"
)

type fakeWriter struct{}

func (f fakeWriter) String(name, val string)        {}
func (f fakeWriter) Int(name string, val int64)     {}
func (f fakeWriter) Float(name string, val float64) {}
func (f fakeWriter) Bool(name string, val bool)     {}
func (f fakeWriter) Bytes(name string, val []byte)  {}
func (f fakeWriter) Null(name string)               {}
func (f fakeWriter) Raw(name, val string)           {}
func (f fakeWriter) Object(name string, val model.Encodable) {}
func (f fakeWriter) Array(name string, n int) model.ArrayWriter { return fakeArrayWriter{} }

type fakeArrayWriter struct{}

func (f fakeArrayWriter) String(val string)         {}
func (f fakeArrayWriter) Int(val int64)             {}
func (f fakeArrayWriter) Float(val float64)         {}
func (f fakeArrayWriter) Bool(val bool)             {}
func (f fakeArrayWriter) Bytes(val []byte)          {}
func (f fakeArrayWriter) Object(val model.Encodable) {}
func (f fakeArrayWriter) Close()                    {}

type fakeReader struct{}

func (f fakeReader) String(name string) (string, bool)             { return "", true }
func (f fakeReader) Int(name string) (int64, bool)                 { return 0, true }
func (f fakeReader) Float(name string) (float64, bool)             { return 0.0, true }
func (f fakeReader) Bool(name string) (bool, bool)                 { return false, true }
func (f fakeReader) Bytes(name string) ([]byte, bool)              { return nil, true }
func (f fakeReader) Object(name string, into model.Decodable) bool { return true }
func (f fakeReader) Array(name string) (model.ArrayReader, bool)   { return fakeArrayReader{}, true }
func (f fakeReader) Raw(name string) (string, bool)                { return "", true }

type fakeArrayReader struct{}

func (f fakeArrayReader) Len() int                               { return 0 }
func (f fakeArrayReader) String(i int) string                     { return "" }
func (f fakeArrayReader) Int(i int) int64                         { return 0 }
func (f fakeArrayReader) Float(i int) float64                     { return 0.0 }
func (f fakeArrayReader) Bool(i int) bool                         { return false }
func (f fakeArrayReader) Bytes(i int) []byte                      { return nil }
func (f fakeArrayReader) Object(i int, into model.Decodable) bool { return true }

func TestORMBoilerplate(t *testing.T) {
	fw := fakeWriter{}
	fr := fakeReader{}

	// CatalogItem
	{
		m := &itemcatalog.CatalogItem{}
		_ = m.ModelName()
		_ = m.Schema()
		_ = m.Pointers()
		_ = m.IsNil()
		m.EncodeFields(fw)
		m.DecodeFields(fr)
		_ = m.Validate(0)

		l := &itemcatalog.CatalogItemList{}
		_ = l.Schema()
		_ = l.Pointers()
		_ = l.Len()
		_ = l.IsNil()
		_ = l.Append()
		_ = l.At(0)
		l.EncodeFields(fw)
		l.DecodeFields(fr)
	}

	// Agreement
	{
		m := &itemcatalog.Agreement{}
		_ = m.ModelName()
		_ = m.Schema()
		_ = m.Pointers()
		_ = m.IsNil()
		m.EncodeFields(fw)
		m.DecodeFields(fr)
		_ = m.Validate(0)

		l := &itemcatalog.AgreementList{}
		_ = l.Schema()
		_ = l.Pointers()
		_ = l.Len()
		_ = l.IsNil()
		_ = l.Append()
		_ = l.At(0)
		l.EncodeFields(fw)
		l.DecodeFields(fr)
	}

	// ItemFilter
	{
		m := &itemcatalog.ItemFilter{}
		_ = m.ModelName()
		_ = m.Schema()
		_ = m.Pointers()
		_ = m.IsNil()
		m.EncodeFields(fw)
		m.DecodeFields(fr)

		l := &itemcatalog.ItemFilterList{}
		_ = l.Schema()
		_ = l.Pointers()
		_ = l.Len()
		_ = l.IsNil()
		_ = l.Append()
		_ = l.At(0)
		l.EncodeFields(fw)
		l.DecodeFields(fr)
	}

	// ListItemsArgs
	{
		m := &itemcatalog.ListItemsArgs{}
		_ = m.ModelName()
		_ = m.Schema()
		_ = m.Pointers()
		_ = m.IsNil()
		m.EncodeFields(fw)
		m.DecodeFields(fr)

		l := &itemcatalog.ListItemsArgsList{}
		_ = l.Schema()
		_ = l.Pointers()
		_ = l.Len()
		_ = l.IsNil()
		_ = l.Append()
		_ = l.At(0)
		l.EncodeFields(fw)
		l.DecodeFields(fr)
	}

	// GetItemArgs
	{
		m := &itemcatalog.GetItemArgs{}
		_ = m.ModelName()
		_ = m.Schema()
		_ = m.Pointers()
		_ = m.IsNil()
		m.EncodeFields(fw)
		m.DecodeFields(fr)

		l := &itemcatalog.GetItemArgsList{}
		_ = l.Schema()
		_ = l.Pointers()
		_ = l.Len()
		_ = l.IsNil()
		_ = l.Append()
		_ = l.At(0)
		l.EncodeFields(fw)
		l.DecodeFields(fr)
	}

	// FindBySKUArgs
	{
		m := &itemcatalog.FindBySKUArgs{}
		_ = m.ModelName()
		_ = m.Schema()
		_ = m.Pointers()
		_ = m.IsNil()
		m.EncodeFields(fw)
		m.DecodeFields(fr)

		l := &itemcatalog.FindBySKUArgsList{}
		_ = l.Schema()
		_ = l.Pointers()
		_ = l.Len()
		_ = l.IsNil()
		_ = l.Append()
		_ = l.At(0)
		l.EncodeFields(fw)
		l.DecodeFields(fr)
	}

	// DeactivateItemArgs
	{
		m := &itemcatalog.DeactivateItemArgs{}
		_ = m.ModelName()
		_ = m.Schema()
		_ = m.Pointers()
		_ = m.IsNil()
		m.EncodeFields(fw)
		m.DecodeFields(fr)

		l := &itemcatalog.DeactivateItemArgsList{}
		_ = l.Schema()
		_ = l.Pointers()
		_ = l.Len()
		_ = l.IsNil()
		_ = l.Append()
		_ = l.At(0)
		l.EncodeFields(fw)
		l.DecodeFields(fr)
	}

	// DeleteItemArgs
	{
		m := &itemcatalog.DeleteItemArgs{}
		_ = m.ModelName()
		_ = m.Schema()
		_ = m.Pointers()
		_ = m.IsNil()
		m.EncodeFields(fw)
		m.DecodeFields(fr)

		l := &itemcatalog.DeleteItemArgsList{}
		_ = l.Schema()
		_ = l.Pointers()
		_ = l.Len()
		_ = l.IsNil()
		_ = l.Append()
		_ = l.At(0)
		l.EncodeFields(fw)
		l.DecodeFields(fr)
	}

	// ListAgreementsArgs
	{
		m := &itemcatalog.ListAgreementsArgs{}
		_ = m.ModelName()
		_ = m.Schema()
		_ = m.Pointers()
		_ = m.IsNil()
		m.EncodeFields(fw)
		m.DecodeFields(fr)

		l := &itemcatalog.ListAgreementsArgsList{}
		_ = l.Schema()
		_ = l.Pointers()
		_ = l.Len()
		_ = l.IsNil()
		_ = l.Append()
		_ = l.At(0)
		l.EncodeFields(fw)
		l.DecodeFields(fr)
	}

	// DeleteAgreementArgs
	{
		m := &itemcatalog.DeleteAgreementArgs{}
		_ = m.ModelName()
		_ = m.Schema()
		_ = m.Pointers()
		_ = m.IsNil()
		m.EncodeFields(fw)
		m.DecodeFields(fr)

		l := &itemcatalog.DeleteAgreementArgsList{}
		_ = l.Schema()
		_ = l.Pointers()
		_ = l.Len()
		_ = l.IsNil()
		_ = l.Append()
		_ = l.At(0)
		l.EncodeFields(fw)
		l.DecodeFields(fr)
	}
}
