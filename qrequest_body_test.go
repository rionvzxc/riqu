package riqu

import (
	"compress/gzip"
	"bytes"
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
			if err := New().URL(server.URL).Method(method).Text("payload").Send(); err != nil {
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
			if err := New().URL(server.URL).Method(test.method).Text("payload").Send(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRedirectBodyEncoding(t *testing.T) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)

	if _, err := writer.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for _, isCrossOrigin := range []bool{false, true} {
		for _, status := range []int{301, 302, 303, 307, 308} {
			t.Run(fmt.Sprintf("crossOrigin=%v/status=%d", isCrossOrigin, status), func(t *testing.T) {
				var targetURL string
				handler := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
					isBodyExpected := request.URL.Path == "/" || status == 307 || status == 308

					if isBodyExpected {
						if request.Method != "POST" || request.Header.Get("Content-Encoding") != "gzip" {
							t.Errorf("%s: expected POST with gzip encoding, got %s %q", request.URL.Path, request.Method, request.Header.Get("Content-Encoding"))
						}
						reader, err := gzip.NewReader(request.Body)

						if err != nil {
							t.Error(err)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						data, err := io.ReadAll(reader)
						reader.Close()

						if err != nil || string(data) != "payload" {
							t.Errorf("unexpected decoded body: %q, %v", data, err)
						}
					} else {
						data, err := io.ReadAll(request.Body)

						if request.Method != "GET" || len(data) != 0 || err != nil {
							t.Errorf("expected GET without body, got %s %q, %v", request.Method, data, err)
						}
						if got := request.Header.Get("Content-Encoding"); got != "" {
							t.Errorf("request without body retained Content-Encoding: %q", got)
						}
					}
					switch request.URL.Path {
					case "/":
						http.Redirect(w, request, targetURL+"/middle", status)
					case "/middle":
						http.Redirect(w, request, "/final", http.StatusPermanentRedirect)
					}
				})
				server := httptest.NewServer(handler)
				defer server.Close()
				targetURL = server.URL

				if isCrossOrigin {
					other := httptest.NewServer(handler)
					defer other.Close()
					targetURL = other.URL
				}
				isCalled := false
				err := New().POST(server.URL).Data(buffer.Bytes()).
					SetHeader("Content-Encoding", "gzip").
					OnResponse(func(res *QResponse) error {
						isCalled = true

						if res.StatusCode() != http.StatusOK || res.URL() != targetURL+"/final" {
							t.Errorf("unexpected final response: %d %s", res.StatusCode(), res.URL())
						}
						return nil
					}).Send()

				if err != nil {
					t.Fatal(err)
				}
				if !isCalled {
					t.Fatal("expected final response callback")
				}
			})
		}
	}
}
