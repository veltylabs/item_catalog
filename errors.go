package servicecatalog

import (
	"webtyp.com/orm"
	"webtyp.com/router"
)

type domainError string

func (e domainError) Error() string { return string(e) }

const (
	ErrNotFound              domainError = "service not found"
	ErrAlreadyExists         domainError = "service already exists"
	ErrSpecialtyNotFound     domainError = "specialty not found"
	ErrSpecialtyInUse        domainError = "specialty in use"
	ErrSpecialtyPrefixExists domainError = "specialty prefix already exists"
	ErrSpecialtySlugExists   domainError = "specialty slug already exists"
	ErrNoRecords             domainError = "service_catalog: no records to save"
	ErrNoIds                 domainError = "service_catalog: no ids given"
	ErrSupplyLineExists      domainError = "service_catalog: the recipe already has this supply"
	ErrSupplyLineNotFound    domainError = "service_catalog: recipe line not found"
	ErrQuantity              domainError = "service_catalog: the quantity must be greater than zero"
)

// decodeError is a request body that could not be read: always 400.
type decodeError struct{ err error }

func (d decodeError) Error() string { return msgDecode + d.err.Error() }

const msgDecode = "service_catalog: cannot read the request: "

// writeError answers err with its status and, always, its text — the only thing an HTTP caller
// sees. 400 = unreadable or invalid input · 404 = not found · 409 = conflict · 500 = anything else.
func writeError(ctx router.Context, err error) {
	ctx.WriteStatus(statusOf(err))
	ctx.Write([]byte(err.Error()))
}

func statusOf(err error) int {
	switch e := err.(type) {
	case decodeError, ValidationError:
		return 400
	case domainError:
		switch e {
		case ErrNotFound, ErrSpecialtyNotFound, ErrSupplyLineNotFound:
			return 404
		case ErrAlreadyExists, ErrSpecialtyPrefixExists, ErrSpecialtySlugExists, ErrSupplyLineExists:
			return 409
		case ErrSpecialtyInUse, ErrNoRecords, ErrNoIds, ErrQuantity:
			return 400
		}
	}
	if orm.IsNotFound(err) {
		return 404
	}
	return 500
}

// ValidationError wraps a model validation failure: 400.
type ValidationError struct {
	Err error
}

func (v ValidationError) Error() string {
	return v.Err.Error()
}
