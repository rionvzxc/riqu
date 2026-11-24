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

func TestRequestContentTypeWithoutBody(t *testing.T) {
	for _, method := range []string{"GET", "HEAD", "POST"} {
		for _, status := range []int{200, 301, 302, 303, 307, 308} {
			t.Run(method+"/"+http.StatusText(status), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
					want := "application/custom"

					if request.URL.Path == "/final" && (status == 303 || method == "POST" && (status == 301 || status == 302)) {
						want = ""
					}
					if got := request.Header.Get("Content-Type"); got != want {
						t.Errorf("%s %s: Content-Type = %q, want %q", request.Method, request.URL.Path, got, want)
					}
					if request.URL.Path == "/" && status != 200 {
						http.Redirect(w, request, "/final", status)
					}
				}))
				defer server.Close()

				if err := New().URL(server.URL).Method(method).ContentType("application/custom").Send(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
