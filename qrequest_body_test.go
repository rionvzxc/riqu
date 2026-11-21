package riqu

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendExplicitBodyOnGETAndHEAD(t *testing.T) {
	for _, method := range []string{"GET", "HEAD"} {
		t.Run(method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				data, err := io.ReadAll(request.Body)
				if err != nil || string(data) != "payload" {
					t.Errorf("unexpected body: %q, %v", data, err)
				}
				if request.Method != method || request.Header.Get("Content-Type") != "text/plain" {
					t.Errorf("unexpected request: %s, %v", request.Method, request.Header)
				}
			}))
			defer server.Close()
			if err := New().Url(server.URL).Method(method).Text("payload").Send(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRedirectBodyPolicy(t *testing.T) {
	for _, test := range []struct {
		method     string
		status     int
		wantMethod string
		wantBody   string
	}{
		{"POST", 301, "GET", ""},
		{"POST", 302, "GET", ""},
		{"POST", 303, "GET", ""},
		{"GET", 303, "GET", ""},
		{"HEAD", 303, "HEAD", ""},
		{"GET", 307, "GET", "payload"},
		{"HEAD", 308, "HEAD", "payload"},
		{"POST", 307, "POST", "payload"},
		{"POST", 308, "POST", "payload"},
	} {
		t.Run(fmt.Sprintf("%s/%d", test.method, test.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "/" {
					http.Redirect(w, request, "/final", test.status)
					return
				}
				data, err := io.ReadAll(request.Body)
				if err != nil || string(data) != test.wantBody || request.Method != test.wantMethod {
					t.Errorf("unexpected redirect: %s, %q, %v", request.Method, data, err)
				}
				wantContentType := ""
				if test.wantBody != "" {
					wantContentType = "text/plain"
				}
				if request.Header.Get("Content-Type") != wantContentType {
					t.Errorf("unexpected Content-Type: %q", request.Header.Get("Content-Type"))
				}
			}))
			defer server.Close()
			if err := New().Url(server.URL).Method(test.method).Text("payload").Send(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
