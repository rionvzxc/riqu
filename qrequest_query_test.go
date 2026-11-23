package riqu

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestQueryUpdatesAndRepeatedSend(t *testing.T) {
	var received []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		received = append(received, request.URL.RawQuery)
	}))
	defer server.Close()
	values := TValues{"lang": {"coba"}, "q": {"go http & client"}}
	request := New().GET(server.URL + "?lang=id&keep=yes&remove=old").
		AddQuery("lang", "discard").
		SetQuery("lang", values["lang"]...).
		SetQuery("q", values["q"]...).
		SetQuery("remove").
		AddQuery("lang", "en").AddQuery("tag", "go", "http")
	values["lang"][0] = "mutated"
	for i := 0; i < 2; i++ {
		if err := request.Send(); err != nil {
			t.Fatal(err)
		}
	}
	want := url.Values{"lang": {"coba", "en"}, "keep": {"yes"}, "q": {"go http & client"}, "tag": {"go", "http"}}.Encode()
	if !reflect.DeepEqual(received, []string{want, want}) {
		t.Fatalf("unexpected queries: %v, want %q twice", received, want)
	}
}

func TestAddQueryPreservesURLValuesAndRedirectQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/" {
			if got := request.URL.Query()["lang"]; !reflect.DeepEqual(got, []string{"id", "coba"}) {
				t.Errorf("unexpected lang values: %v", got)
			}
			http.Redirect(w, request, "/final?lang=redirect", http.StatusFound)
			return
		}
		fmt.Fprint(w, request.URL.RawQuery)
	}))
	defer server.Close()
	var text string
	var readErr error
	err := New().AddQuery("lang", "coba").GET(server.URL + "?lang=id").
		OnResponse(func(response *QResponse) error { text, readErr = response.Text(); return nil }).Send()
	if err != nil || readErr != nil || text != "lang=redirect" {
		t.Fatalf("unexpected redirect result: %q, %v, %v", text, err, readErr)
	}
}

func TestSetAllQueriesReplacesURLAndConfiguredValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		fmt.Fprint(w, request.URL.RequestURI())
	}))
	defer server.Close()
	values := TValues{"lang": {"id"}, "tag": {"go", "http"}}
	request := New().GET(server.URL + "/?old=value").
		SetQuery("discard", "old").AddQuery("discard", "older").
		SetAllQueries(values).AddQuery("lang", "en").SetQuery("tag", "new")
	values["lang"][0] = "changed"
	var text string
	var readErr error
	request.OnResponse(func(response *QResponse) error { text, readErr = response.Text(); return nil })
	for i := 0; i < 2; i++ {
		if err := request.Send(); err != nil {
			t.Fatal(err)
		}
		if readErr != nil || text != "/?lang=id&lang=en&tag=new" {
			t.Fatalf("unexpected replacement query: %q, %v", text, readErr)
		}
	}
	for _, empty := range []TValues{nil, {}} {
		request.Url(server.URL + "/?old=value").SetAllQueries(empty)
		if err := request.Send(); err != nil {
			t.Fatal(err)
		}
		if readErr != nil || text != "/" {
			t.Fatalf("expected entire query to be cleared, got %q, %v", text, readErr)
		}
	}
}
