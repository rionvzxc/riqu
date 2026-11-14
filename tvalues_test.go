package riqu

import (
	"reflect"
	"testing"
)

func TestTValuesSetReplacesValues(t *testing.T) {
	values := TValues{"name": {"old", "value"}}
	values.Set("name", "first", "second")

	if got := values.GetAll("name"); !reflect.DeepEqual(got, []string{"first", "second"}) {
		t.Fatalf("expected replaced values, got %#v", got)
	}
}
func TestTValuesAddAppendsValues(t *testing.T) {
	values := TValues{"name": {"first"}}
	values.Add("name", "second", "third")
	values.Add("other", "value")

	if got := values.GetAll("name"); !reflect.DeepEqual(got, []string{"first", "second", "third"}) {
		t.Fatalf("expected appended values, got %#v", got)
	}
	if got := values.Get("other"); got != "value" {
		t.Fatalf("expected first value %q, got %q", "value", got)
	}
}
func TestTValuesGetAndGetAll(t *testing.T) {
	values := TValues{"name": {"first", "second"}}

	if got := values.Get("name"); got != "first" {
		t.Fatalf("expected first value %q, got %q", "first", got)
	}
	all := values.GetAll("name")
	if !reflect.DeepEqual(all, []string{"first", "second"}) {
		t.Fatalf("expected all values, got %#v", all)
	}
	all[0] = "changed"

	if got := values.Get("name"); got != "first" {
		t.Fatalf("expected GetAll result not to mutate map, got %q", got)
	}
}
func TestTValuesGetMissingOrEmpty(t *testing.T) {
	values := TValues{"empty": {}}

	if got := values.Get("missing"); got != "" {
		t.Fatalf("expected empty value for missing key, got %q", got)
	}
	if got := values.Get("empty"); got != "" {
		t.Fatalf("expected empty value for empty key, got %q", got)
	}
	if got := values.GetAll("missing"); len(got) != 0 {
		t.Fatalf("expected no values for missing key, got %#v", got)
	}
}
func TestTValuesDelRemovesValues(t *testing.T) {
	values := TValues{"name": {"first", "second"}}
	values.Del("name")

	if _, ok := values["name"]; ok {
		t.Fatal("expected key to be deleted")
	}
}
func TestTValuesExist(t *testing.T) {
	values := TValues{"name": {"value"}, "empty": {}}

	if !values.Exist("name") {
		t.Fatal("expected existing key to be found")
	}
	if !values.Exist("empty") {
		t.Fatal("expected key with empty values to be found")
	}
	if values.Exist("missing") {
		t.Fatal("expected missing key not to be found")
	}
}
func TestTValuesSetEmptyClearsValues(t *testing.T) {
	values := TValues{"name": {"first"}}
	values.Set("name")

	if _, ok := values["name"]; !ok {
		t.Fatal("expected Set with no values to replace key with an empty slice")
	}
	if got := values.GetAll("name"); len(got) != 0 {
		t.Fatalf("expected no values after empty Set, got %#v", got)
	}
}
func TestTValuesCloneIsIndependent(t *testing.T) {
	values := TValues{"name": {"first", "second"}, "nil": nil, "empty": {}}
	cloned := values.Clone()
	cloned["name"][0] = "changed"
	cloned["new"] = []string{"value"}

	if values.Get("name") != "first" {
		t.Fatal("expected cloned values not to mutate original map")
	}
	if values.Exist("new") {
		t.Fatal("expected cloned map additions not to mutate original map")
	}
	if cloned["nil"] != nil || cloned["empty"] == nil {
		t.Fatal("expected Clone to preserve nil and empty slices")
	}
}
func TestTValuesClonePreservesNilMap(t *testing.T) {
	var values TValues

	if values.Clone() != nil {
		t.Fatal("expected nil map clone to remain nil")
	}
}
