package tests

import (
	"testing"

	itemcatalog "github.com/veltylabs/item_catalog"
)

func TestSentinelErrorTexts(t *testing.T) {
	cases := []struct {
		err  error
		text string
	}{
		{itemcatalog.ErrNotFound, "item not found"},
		{itemcatalog.ErrAlreadyExists, "item already exists"},
		{itemcatalog.ErrSpecialtyNotFound, "specialty not found"},
		{itemcatalog.ErrSpecialtyInUse, "specialty in use"},
		{itemcatalog.ErrSpecialtyPrefixExists, "specialty prefix already exists"},
		{itemcatalog.ErrSpecialtySlugExists, "specialty slug already exists"},
	}

	for _, c := range cases {
		if c.err.Error() != c.text {
			t.Errorf("expected error text %q, got %q", c.text, c.err.Error())
		}
	}
}
