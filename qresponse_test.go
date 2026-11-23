package riqu

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNilResponseMetadata(t *testing.T) {
	var response *QResponse
	if response.ContentLength() != -1 {
		t.Fatal("expected unknown length for nil response")
	}
	if response.StatusCode() != 0 || response.Status() != "" || response.Url() != "" || response.Method() != "" {
		t.Fatal("unexpected nil response metadata")
	}
	if response.Header("missing", "default") != "default" || response.AllHeaders() != nil || response.HeaderValues("x") != nil {
		t.Fatal("unexpected nil response headers")
	}
	if response.Cookie("x") != nil || response.Cookies("x") != nil || response.AllCookies() != nil {
		t.Fatal("unexpected nil response cookies")
	}
	if response.BindHeader(&struct{}{}) == nil || response.BindCookie(&struct{}{}) == nil {
		t.Fatal("binding a nil response must fail")
	}
}

func TestResponseContentLength(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/known":
			w.Header().Set("Content-Length", "5")
			io.WriteString(w, "hello")
		case "/unknown":
			w.(http.Flusher).Flush()
			io.WriteString(w, "hello")
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	for _, test := range []struct {
		path   string
		length int64
	}{
		{"/known", 5}, {"/unknown", -1}, {"/empty", 0},
	} {
		t.Run(test.path, func(t *testing.T) {
			err := New().GET(server.URL + test.path).OnSuccess(func(response *QResponse) {
				if response.ContentLength() != test.length {
					t.Errorf("ContentLength() = %d, want %d", response.ContentLength(), test.length)
				}
			}).Send()
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
