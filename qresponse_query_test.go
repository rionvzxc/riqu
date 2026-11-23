package riqu

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestResponseQueryUsesEachRedirectURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/start" {
			http.Redirect(w, request, "/final?page=2&tag=go&tag=http&empty=&q=go+http", http.StatusFound)
		}
	}))
	defer server.Close()
	for _, mode := range []TRedirectResponseMode{VRedirectResponseAll, VRedirectResponseLast} {
		var pages []string
		err := New().GET(server.URL + "/start?page=1").
			Redirect(TRedirect{Max: 1, ResponseMode: mode}).
			OnSuccess(func(response *QResponse) {
				pages = append(pages, response.Query("page", "missing"))
				if response.Query("missing", "fallback") != "fallback" {
					t.Fatal("expected default for missing parameter")
				}
				if response.StatusCode() != http.StatusOK {
					return
				}
				if response.Query("empty", "fallback") != "" || response.Query("q", "") != "go http" {
					t.Fatal("unexpected decoded query values")
				}
				values := response.QueryValues("tag")
				if !reflect.DeepEqual(values, []string{"go", "http"}) {
					t.Fatalf("unexpected repeated values: %v", values)
				}
				values[0] = "changed"
				all := response.AllQueries()
				all["tag"][0] = "changed"
				all.Set("page", "99")
				var target struct {
					Page int      `query:"page"`
					Tags []string `query:"tag"`
				}
				if err := response.BindQuery(&target); err != nil {
					t.Fatal(err)
				}
				if target.Page != 2 || !reflect.DeepEqual(target.Tags, []string{"go", "http"}) {
					t.Fatalf("query copies changed binding: %+v", target)
				}
			}).Send()
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"2"}
		if mode == VRedirectResponseAll {
			want = []string{"1", "2"}
		}
		if !reflect.DeepEqual(pages, want) {
			t.Fatalf("unexpected redirect queries: %v, want %v", pages, want)
		}
	}
}

func TestResponseQueryParsingErrors(t *testing.T) {
	response := &QResponse{resUrl: "https://example.com/?page=2&bad=%zz"}
	if response.Query("page", "") != "2" || response.Query("bad", "fallback") != "fallback" {
		t.Fatal("expected successfully parsed values to remain available")
	}
	var target struct {
		Page int `query:"page"`
	}
	err := response.BindQuery(&target)
	if err == nil || !strings.HasPrefix(err.Error(), "riqu: ") || target.Page != 0 {
		t.Fatalf("expected parsing error before binding, got %+v, %v", target, err)
	}
	invalidURL := &QResponse{resUrl: ":invalid"}
	if invalidURL.Query("page", "default") != "default" || invalidURL.BindQuery(&target) == nil {
		t.Fatal("expected invalid URL to return defaults and a binding error")
	}
	valid := &QResponse{resUrl: "https://example.com/?page=bad"}
	if valid.BindQuery(&target) == nil || valid.BindQuery(nil) == nil {
		t.Fatal("expected conversion and invalid target errors")
	}
}

func TestNilResponseQuery(t *testing.T) {
	var response *QResponse
	if response.Query("page", "default") != "default" || response.QueryValues("page") != nil || response.AllQueries() != nil {
		t.Fatal("unexpected nil response query values")
	}
	if response.BindQuery(&struct{}{}) == nil {
		t.Fatal("expected binding a nil response to fail")
	}
}
