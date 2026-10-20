package tests

import (
	"testing"

	servicecatalog "github.com/veltylabs/service_catalog"
)

func TestSentinelErrorTexts(t *testing.T) {
	cases := []struct {
		err  error
		text string
	}{
		{servicecatalog.ErrNotFound, "service not found"},
		{servicecatalog.ErrAlreadyExists, "service already exists"},
		{servicecatalog.ErrSpecialtyNotFound, "specialty not found"},
		{servicecatalog.ErrSpecialtyInUse, "specialty in use"},
		{servicecatalog.ErrSpecialtyPrefixExists, "specialty prefix already exists"},
		{servicecatalog.ErrSpecialtySlugExists, "specialty slug already exists"},
	}

	for _, c := range cases {
		if c.err.Error() != c.text {
			t.Errorf("expected error text %q, got %q", c.text, c.err.Error())
		}
	}
}
