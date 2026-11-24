package riqu

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

type tCookieTestPublicSuffixList struct {
	suffix string
}

func (this tCookieTestPublicSuffixList) PublicSuffix(domain string) string {
	if domain == this.suffix || strings.HasSuffix(domain, "."+this.suffix) {
		return this.suffix
	}
	return domain[strings.LastIndex(domain, ".")+1:]
}

func (this tCookieTestPublicSuffixList) String() string {
	return "riqu test suffix list"
}

func TestRedirectCookieDomainAndPublicSuffixScope(t *testing.T) {
	for _, test := range []struct {
		name   string
		origin string
		target string
		domain string
		suffix string
		isDeletion bool
		want   []string
	}{
		{"parent domain to child", "example.test", "a.example.test", "example.test", "", false, []string{"new"}},
		{"host only to child", "example.test", "a.example.test", "", "", false, nil},
		{"parent domain to sibling", "a.example.test", "b.example.test", "example.test", "", false, []string{"new"}},
		{"host only to sibling", "a.example.test", "b.example.test", "", "", false, nil},
		{"leading dot parent domain", "a.example.test", "a.example.test", ".example.test", "", false, []string{"new"}},
		{"unrelated target", "a.example.test", "other.test", "example.test", "", false, nil},
		{"invalid sibling domain", "a.example.test", "a.example.test", "b.example.test", "", false, []string{"manual"}},
		{"reject public suffix", "a.example.test", "a.example.test", "test", "test", false, []string{"manual"}},
		{"reject public suffix deletion", "a.example.test", "a.example.test", "test", "test", true, []string{"manual"}},
		{"accept registrable parent", "a.example.test", "a.example.test", "example.test", "test", false, []string{"new"}},
		{"delete registrable parent", "a.example.test", "a.example.test", "example.test", "test", true, nil},
		{"reject private suffix", "a.tenant.test", "a.tenant.test", "tenant.test", "tenant.test", false, []string{"manual"}},
		{"host only under private suffix", "a.tenant.test", "a.tenant.test", "", "tenant.test", false, []string{"new"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				requests.Add(1)

				if request.URL.Path == "/start" {
					cookie := &http.Cookie{Name: "session", Value: "new", Domain: test.domain, Path: "/"}

					if test.isDeletion {
						cookie.MaxAge = -1
					}
					http.SetCookie(w, cookie)
					http.Redirect(w, request, "http://"+test.target+"/final", http.StatusFound)
					return
				}
				var values []string

				for _, cookie := range request.Cookies() {
					if cookie.Name == "session" {
						values = append(values, cookie.Value)
					}
				}
				if !reflect.DeepEqual(values, test.want) {
					t.Errorf("target %s received session %v, want %v", request.Host, values, test.want)
				}
			}))
			defer server.Close()
			var dialer net.Dialer
			transport := &QTransport{httpTransport: &http.Transport{
				DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return dialer.DialContext(ctx, network, server.Listener.Addr().String())
				},
			}}
			defer transport.CloseIdleConnections()
			options := &TJarOptions{}

			if test.suffix != "" {
				options.PublicSuffixList = tCookieTestPublicSuffixList{suffix: test.suffix}
			}
			jar, err := NewCookieJar(options)

			if err != nil {
				t.Fatal(err)
			}
			// Mutating the caller's options must not change either jar's policy.
			options.PublicSuffixList = nil
			err = New().GET("http://"+test.origin+"/start").Transport(transport).
				SetCookie("session", "manual").CookieJar(jar).Send()

			if err != nil {
				t.Fatal(err)
			}
			if got := requests.Load(); got != 2 {
				t.Fatalf("received %d requests, want 2", got)
			}
		})
	}
}

func TestRedirectCookieHTTPScope(t *testing.T) {
	for _, scheme := range []string{"http", "https"} {
		for _, isSecure := range []bool{false, true} {
			for _, isCookieDeletion := range []bool{false, true} {
				name := scheme

				if isSecure {
					name += "/secure"
				} else {
					name += "/plain"
				}
				if isCookieDeletion {
					name += "/delete"
				}
				t.Run(name, func(t *testing.T) {
					var origin, bridge string
					var requests atomic.Int32
					handler := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
						requests.Add(1)

						if request.URL.Path == "/start" {
							cookie := &http.Cookie{Name: "session", Value: "new", Path: "/", Secure: isSecure}

							if isCookieDeletion {
								cookie.MaxAge = -1
							}
							http.SetCookie(w, cookie)
							http.Redirect(w, request, bridge+"/bridge", http.StatusFound)
							return
						}
						isHTTPS := request.TLS != nil
						want := "new"

						if isCookieDeletion || isSecure && !isHTTPS {
							want = ""
						}
						if request.URL.Path == "/final" && isSecure && !isHTTPS {
							want = "manual"
						}
						var values []string

						for _, cookie := range request.Cookies() {
							if cookie.Name == "session" {
								values = append(values, cookie.Value)
							}
						}
						var expected []string

						if want != "" {
							expected = []string{want}
						}
						if !reflect.DeepEqual(values, expected) {
							t.Errorf("%s (TLS %v) received session %v, want %v", request.URL.Path, isHTTPS, values, expected)
						}
						if request.URL.Path == "/bridge" {
							http.Redirect(w, request, origin+"/final", http.StatusFound)
						}
					})
					plainServer := httptest.NewServer(handler)
					defer plainServer.Close()
					tlsServer := httptest.NewTLSServer(handler)
					defer tlsServer.Close()
					origin, bridge = plainServer.URL, tlsServer.URL

					if scheme == "https" {
						origin, bridge = tlsServer.URL, plainServer.URL
					}
					// Trust only the local test certificate; keep TLS verification enabled.
					transport := &QTransport{httpTransport: tlsServer.Client().Transport.(*http.Transport).Clone()}
					defer transport.CloseIdleConnections()
					jar, err := NewCookieJar(nil)

					if err != nil {
						t.Fatal(err)
					}
					err = New().GET(origin+"/start").Transport(transport).
						SetCookie("session", "manual").CookieJar(jar).Send()

					if err != nil {
						t.Fatal(err)
					}
					if got := requests.Load(); got != 3 {
						t.Fatalf("received %d requests, want 3", got)
					}
				})
			}
		}
	}
}
