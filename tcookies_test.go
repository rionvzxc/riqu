package riqu

import "testing"

func TestTCookiesOperations(t *testing.T) {
	cookies := TCookies{}
	cookies.Add(
		TCookie{Name: "first", Value: "one"},
		TCookie{Name: "second", Value: "two"},
		TCookie{Name: "first", Value: "three"},
	)

	if cookies[0].Name != "first" || cookies[0].Value != "one" || cookies[1].Name != "second" || cookies[1].Value != "two" {
		t.Fatalf("Add() did not preserve individual cookies: %#v", cookies)
	}
	if !cookies.Exist("first") || cookies.Exist("missing") {
		t.Fatalf("Exist() returned unexpected results")
	}
	if got := cookies.Get("second"); got == nil || got.Value != "two" {
		t.Fatalf("Get(second) = %#v", got)
	}
	if got := cookies.GetAll("first"); len(got) != 2 || got[0].Value != "one" || got[1].Value != "three" {
		t.Fatalf("GetAll(first) = %#v", got)
	}
	cookies.Set(TCookies{nil, {Name: "remove", Value: "1"}, {Name: "keep", Value: "2"}, {Name: "remove", Value: "3"}})
	cookies.Del("remove")
	if len(cookies) != 2 || !cookies.Exist("keep") {
		t.Fatalf("Del() did not remove matching and nil cookies: %#v", cookies)
	}
	if cookies.Get("remove") != nil || cookies.Exist("remove") {
		t.Fatalf("removed cookies are still present")
	}
}

func TestTCookiesDelPreservesDuplicateOrder(t *testing.T) {
	cookies := TCookies{
		{Name: "remove"},
		{Name: "session", Value: "first"},
		{Name: "session", Value: "last"},
	}
	cookies.Del("remove")
	if len(cookies) != 2 || cookies.Get("session").Value != "first" || cookies[1].Value != "last" {
		t.Fatalf("Del changed the remaining cookie order: %v", cookies)
	}
}
