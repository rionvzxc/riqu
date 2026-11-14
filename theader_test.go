package riqu

import (
	"reflect"
	"sort"
	"testing"
)

func TestTHeaderCanonicalizesNames(t *testing.T) {
	header := THeader{}
	header.Set("content-type", "text/plain")
	header.Add("CONTENT-TYPE", "application/json")

	if got := header.Get("Content-Type"); got != "text/plain" {
		t.Fatalf("expected first header value %q, got %q", "text/plain", got)
	}
	if got := header.GetAll("content-type"); !reflect.DeepEqual(got, []string{"text/plain", "application/json"}) {
		t.Fatalf("expected canonical header values, got %#v", got)
	}
	if !header.Exist("CONTENT-TYPE") {
		t.Fatal("expected canonical header name to exist")
	}
	if _, exists := header["Content-Type"]; !exists {
		t.Fatal("expected stored header key to be canonical")
	}
}
func TestTHeaderSetReplacesAndDelRemoves(t *testing.T) {
	header := THeader{"X-Test": {"old", "value"}}
	header.Set("x-test", "new")

	if got := header.GetAll("X-Test"); !reflect.DeepEqual(got, []string{"new"}) {
		t.Fatalf("expected replaced header values, got %#v", got)
	}
	header.Del("X-TEST")

	if header.Exist("x-test") {
		t.Fatal("expected header key to be deleted")
	}
}
func TestTHeaderConvertsLiteralKeysToCanonicalHTTPHeader(t *testing.T) {
	header := THeader{"content-type": {"text/plain"}, "CONTENT-TYPE": {"application/json"}}
	converted := map[string][]string(header.Clone())

	got := converted["Content-Type"]
	sort.Strings(got)

	if !reflect.DeepEqual(got, []string{"application/json", "text/plain"}) {
		t.Fatalf("expected merged canonical values, got %#v", got)
	}
}
func TestTHeaderCloneIsIndependent(t *testing.T) {
	header := THeader{"X-Test": {"first", "second"}, "nil": nil, "empty": {}}
	cloned := header.Clone()
	cloned["X-Test"][0] = "changed"
	cloned["X-New"] = []string{"value"}

	if header.Get("x-test") != "first" {
		t.Fatal("expected cloned values not to mutate original map")
	}
	if header.Exist("x-new") {
		t.Fatal("expected cloned map additions not to mutate original map")
	}
	if cloned["Nil"] != nil || cloned["Empty"] == nil {
		t.Fatal("expected Clone to preserve nil and empty slices")
	}
}
func TestTHeaderClonePreservesNilMap(t *testing.T) {
	var header THeader

	if header.Clone() != nil {
		t.Fatal("expected nil map clone to remain nil")
	}
}
