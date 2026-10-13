package tests

import (
	"testing"

	itemcatalog "github.com/veltylabs/item_catalog"
	"webtyp.com/json"
	"webtyp.com/model"
	"webtyp.com/orm"
	"webtyp.com/router/mock"
	"webtyp.com/storage/mem"
)

// Every refused operation says why: over HTTP (rpc) the body is the only thing a caller sees, and
// a bare status left people guessing (a bare Specialty instead of UpsertSpecialtyArgs gave a
// silent 400).
func opRouter(t *testing.T) *mock.Router {
	t.Helper()
	module, err := itemcatalog.New(orm.New(mem.New()), itemcatalog.Deps{IDs: &MockIDGen{}})
	if err != nil {
		t.Fatal(err)
	}
	r := &mock.Router{}
	r.Configure(mock.Config{Authorize: func(string, model.Resource, model.Action) bool { return true }})
	module.MountOperations(r)
	return r
}

func invoke(r *mock.Router, op string, body []byte) *mock.Context {
	ctx := &mock.Context{InBody: body}
	ctx.SetUserID("u1")
	r.Invoke("OP", "/"+op, ctx)
	return ctx
}

func encode(t *testing.T, v model.Encodable) []byte {
	t.Helper()
	var b []byte
	if err := json.Encode(v, &b); err != nil {
		t.Fatal(err)
	}
	return b
}

func wantRefused(t *testing.T, ctx *mock.Context, status int, body string) {
	t.Helper()
	got := string(ctx.ResponseBody())
	if ctx.Status != status || got == "" || (body != "" && got != body) {
		t.Fatalf("got %d %q, want %d %q (never an empty body)", ctx.Status, got, status, body)
	}
}

func TestOpErrors_SayWhy(t *testing.T) {
	r := opRouter(t)

	t.Run("decode failure", func(t *testing.T) {
		wantRefused(t, invoke(r, itemcatalog.OpUpsertSpecialty, []byte("{not json")), 400, "")
	})
	t.Run("empty records", func(t *testing.T) {
		ctx := invoke(r, itemcatalog.OpUpsertSpecialty, encode(t, &itemcatalog.UpsertSpecialtyArgs{}))
		wantRefused(t, ctx, 400, itemcatalog.ErrNoRecords.Error())
	})
	t.Run("empty ids", func(t *testing.T) {
		ctx := invoke(r, itemcatalog.OpDeleteItem, encode(t, &itemcatalog.DeleteItemsArgs{}))
		wantRefused(t, ctx, 400, itemcatalog.ErrNoIds.Error())
	})
	t.Run("validation", func(t *testing.T) {
		args := &itemcatalog.UpsertSpecialtyArgs{Records: []itemcatalog.Specialty{{TenantId: "t1"}}}
		wantRefused(t, invoke(r, itemcatalog.OpUpsertSpecialty, encode(t, args)), 400, "")
	})
	t.Run("duplicate prefix", func(t *testing.T) {
		first := &itemcatalog.UpsertSpecialtyArgs{Records: []itemcatalog.Specialty{{TenantId: "t1", Prefix: "md", Slug: "medicina", Name: "Medicina"}}}
		if ctx := invoke(r, itemcatalog.OpUpsertSpecialty, encode(t, first)); ctx.Status != 0 && ctx.Status != 200 {
			t.Fatalf("first specialty: %d %s", ctx.Status, ctx.ResponseBody())
		}
		again := &itemcatalog.UpsertSpecialtyArgs{Records: []itemcatalog.Specialty{{TenantId: "t1", Prefix: "md", Slug: "otra", Name: "Otra"}}}
		wantRefused(t, invoke(r, itemcatalog.OpUpsertSpecialty, encode(t, again)), 409, itemcatalog.ErrSpecialtyPrefixExists.Error())
	})
	t.Run("not found", func(t *testing.T) {
		ctx := invoke(r, itemcatalog.OpGetSpecialty, encode(t, &itemcatalog.GetSpecialtyArgs{TenantId: "t1", Id: "nope"}))
		wantRefused(t, ctx, 404, itemcatalog.ErrSpecialtyNotFound.Error())
	})
}
