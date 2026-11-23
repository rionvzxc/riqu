package riqu

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseHeadersAndBinding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Count", "12")
		w.Header().Add("X-Tags", "one")
		w.Header().Add("X-Tags", "two")
		w.Header().Set("X-Empty", "")
	}))
	defer server.Close()
	err := New().GET(server.URL).OnResponse(func(response *QResponse) error {
		if response.Header("x-count", "missing") != "12" || response.Header("missing", "default") != "default" || response.Header("X-Empty", "default") != "" {
			t.Fatal("unexpected header lookup")
		}
		values := response.HeaderValues("x-tags")
		if len(values) != 2 || values[0] != "one" || values[1] != "two" {
			t.Fatalf("unexpected header values: %v", values)
		}
		values[0] = "changed"
		clone := response.AllHeaders()
		clone.Set("X-Count", "99")
		var target struct {
			Count int      `header:"x-count"`
			Tags  []string `header:"X-Tags"`
			Keep  string   `header:"missing"`
			Skip  string   `header:"-"`
		}
		target.Keep = "kept"
		if err := response.BindHeader(&target); err != nil {
			t.Fatal(err)
		}
		if target.Count != 12 || target.Tags[0] != "one" || target.Keep != "kept" {
			t.Fatalf("unexpected binding: %+v", target)
		}
		if err := response.BindHeader(nil); err == nil {
			t.Fatal("nil binding target must fail")
		}
		return nil
	}).Send()
	if err != nil {
		t.Fatal(err)
	}
}
