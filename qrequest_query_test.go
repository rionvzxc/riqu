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
		AddQuery(TValues{"lang": {"discard"}}).
		Query(values).
		Query(TValues{"remove": nil}).
		AddQuery(TValues{"lang": {"en"}, "tag": {"go", "http"}})
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
	err := New().AddQuery(TValues{"lang": {"coba"}}).GET(server.URL + "?lang=id").
		OnSuccess(func(response *QResponse) { text, readErr = response.Text() }).Send()
	if err != nil || readErr != nil || text != "lang=redirect" {
		t.Fatalf("unexpected redirect result: %q, %v, %v", text, err, readErr)
	}
}
