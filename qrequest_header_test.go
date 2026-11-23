package riqu

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostHeaderRedirectScope(t *testing.T) {
	hosts := make(chan string, 3)
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		hosts <- request.Host
	}))
	defer other.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		hosts <- request.Host
		if request.URL.Path == "/" {
			http.Redirect(w, request, "/same", http.StatusFound)
			return
		}
		http.Redirect(w, request, other.URL, http.StatusFound)
	}))
	defer server.Close()
	if err := New().GET(server.URL).SetAllHeaders(THeader{"host": {"virtual.example"}}).Send(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"virtual.example", "virtual.example", other.Listener.Addr().String()} {
		if got := <-hosts; got != want {
			t.Fatalf("unexpected Host: %q, want %q", got, want)
		}
	}
}

func TestRequestHeaderSetAndAdd(t *testing.T) {
	header := THeader{"X-Keep": {"original"}, "X-Value": {"old"}}
	request := New().SetAllHeaders(header).
		SetHeader("x-value", "one", "two").AddHeader("X-Value", "three")
	header["X-Keep"][0] = "changed"
	if request.header.Get("X-Keep") != "original" {
		t.Fatal("request headers must be copied")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Values("X-Value"); len(got) != 3 || got[0] != "one" || got[1] != "two" || got[2] != "three" {
			t.Errorf("unexpected header values: %v", got)
		}
	}))
	defer server.Close()
	if err := request.GET(server.URL).Send(); err != nil {
		t.Fatal(err)
	}
	request.SetAllHeaders(nil).AddHeader("X-New", "value")
	if len(request.header) != 1 || request.header.Get("X-New") != "value" {
		t.Fatalf("unexpected replaced headers: %v", request.header)
	}
}
