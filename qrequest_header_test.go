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
	if err := New().GET(server.URL).Header(THeader{"host": {"virtual.example"}}).Send(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"virtual.example", "virtual.example", other.Listener.Addr().String()} {
		if got := <-hosts; got != want {
			t.Fatalf("unexpected Host: %q, want %q", got, want)
		}
	}
}
