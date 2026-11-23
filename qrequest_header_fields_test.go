package riqu

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestUserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if got := request.UserAgent(); got != "riqu-example/1.0" {
			t.Errorf("unexpected User-Agent: %q", got)
		}
		if len(request.Header.Values("User-Agent")) != 1 {
			t.Error("expected one User-Agent value")
		}
	}))
	defer server.Close()
	err := New().GET(server.URL).
		SetHeader("User-Agent", "old").UserAgent("riqu-example/1.0").Send()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRequestContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Content-Type"); got != "application/custom" {
			t.Errorf("unexpected Content-Type: %q", got)
		}
		if len(request.Header.Values("Content-Type")) != 1 {
			t.Error("expected one Content-Type value")
		}
	}))
	defer server.Close()
	err := New().POST(server.URL).
		ContentType("text/plain").ContentType("application/custom").JSON(TMap{"ok": true}).Send()
	if err != nil {
		t.Fatal(err)
	}
}
