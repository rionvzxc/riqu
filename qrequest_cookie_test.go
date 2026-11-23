package riqu

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRedirectManualCookieScope(t *testing.T) {
	for _, test := range []struct {
		name   string
		cookie TCookie
		start  string
		final  string
		want   string
	}{
		{"different path", TCookie{Path: "/other"}, "/start", "/final", "manual"},
		{"path boundary", TCookie{Path: "/other"}, "/start", "/otherwise", "manual"},
		{"matching path", TCookie{Path: "/other"}, "/start", "/other/final", "new"},
		{"default path", TCookie{}, "/scope/start", "/final", "manual"},
		{"matching default path", TCookie{}, "/scope/start", "/scope/final", "new"},
		{"invalid domain", TCookie{Path: "/", Domain: "unrelated.invalid"}, "/start", "/final", "manual"},
		{"secure on HTTP", TCookie{Path: "/", Secure: true}, "/start", "/final", "manual"},
		{"delete matching path", TCookie{Path: "/", MaxAge: -1}, "/start", "/final", ""},
		{"delete different path", TCookie{Path: "/other", MaxAge: -1}, "/start", "/final", "manual"},
		{"expired matching path", TCookie{Path: "/", Expires: time.Unix(1, 0)}, "/start", "/final", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.URL.Path == test.start {
					cookie := test.cookie
					cookie.Name = "session"
					cookie.Value = "new"
					http.SetCookie(w, &cookie)
					http.Redirect(w, request, test.final, http.StatusFound)
					return
				}
				want := "keep=value"

				if test.want != "" {
					want += "; session=" + test.want
				}
				if got := request.Header.Get("Cookie"); got != want {
					t.Errorf("Cookie = %q, want %q", got, want)
				}
			}))
			defer server.Close()
			jar, err := NewCookieJar(nil)

			if err != nil {
				t.Fatal(err)
			}
			request := New().GET(server.URL+test.start).
				SetHeader("Cookie", "keep=value").SetCookie("session", "manual").CookieJar(jar)

			for i := 0; i < 2; i++ {
				if err := request.Send(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestRedirectManualCookieReturnsOutsideChangedScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/start":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "new", Path: "/other"})
			http.Redirect(w, request, "/other/step", http.StatusFound)
		case "/other/step":
			if got := request.Header.Get("Cookie"); got != "session=new" {
				t.Errorf("Cookie inside scope = %q", got)
			}
			http.Redirect(w, request, "/final", http.StatusFound)
		default:
			if got := request.Header.Get("Cookie"); got != "session=manual" {
				t.Errorf("Cookie outside scope = %q", got)
			}
		}
	}))
	defer server.Close()
	jar, err := NewCookieJar(nil)

	if err != nil {
		t.Fatal(err)
	}
	if err := New().GET(server.URL+"/start").SetCookie("session", "manual").CookieJar(jar).Send(); err != nil {
		t.Fatal(err)
	}
}

func TestRedirectUpdatesAndDeletesManualCookies(t *testing.T) {
	for _, maxAge := range []int{0, -1} {
		t.Run(fmt.Sprintf("MaxAge=%d", maxAge), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "/" {
					http.SetCookie(w, &http.Cookie{Name: "session", Value: "new", Path: "/", MaxAge: maxAge})
					http.Redirect(w, request, "/final", http.StatusFound)
					return
				}
				fmt.Fprint(w, request.Header.Get("Cookie"))
			}))
			defer server.Close()
			jar, err := NewCookieJar(nil)
			if err != nil {
				t.Fatal(err)
			}
			var text string
			var readErr error
			request := New().GET(server.URL).
				SetAllHeaders(THeader{"Cookie": {"session=header; keep=value"}}).
				SetAllCookies(TValues{"session": {"old"}}).CookieJar(jar).
				OnResponse(func(response *QResponse) error { text, readErr = response.Text(); return nil })
			for i := 0; i < 2; i++ {
				if err := request.Send(); err != nil {
					t.Fatal(err)
				}
				want := "keep=value; session=new"
				if maxAge < 0 {
					want = "keep=value"
				}
				if readErr != nil || text != want {
					t.Fatalf("unexpected redirected cookies: %q, want %q, error %v", text, want, readErr)
				}
			}
		})
	}
}

func TestRequestCookieSetAndAdd(t *testing.T) {
	values := TValues{"session": {"old"}, "keep": {"yes"}}
	request := New().SetAllCookies(values).
		SetCookie("session", "one", "two").AddCookie("session", "three")
	values["keep"][0] = "changed"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sessions []string
		for _, cookie := range r.Cookies() {
			if cookie.Name == "session" {
				sessions = append(sessions, cookie.Value)
			}
		}
		if len(sessions) != 3 || sessions[0] != "one" || sessions[1] != "two" || sessions[2] != "three" {
			t.Errorf("unexpected cookies: %v", sessions)
		}
		cookie, err := r.Cookie("keep")
		if err != nil || cookie.Value != "yes" {
			t.Errorf("request cookies must be copied: %v", err)
		}
	}))
	defer server.Close()
	if err := request.GET(server.URL).Send(); err != nil {
		t.Fatal(err)
	}
	request.SetCookie("session")
	if len(request.cookies) != 1 || request.cookies[0].Name != "keep" {
		t.Fatalf("SetCookie without values must remove the named cookie: %v", request.cookies)
	}
	request.SetAllCookies(nil)
	if len(request.cookies) != 0 {
		t.Fatal("SetAllCookies(nil) must clear cookies")
	}
}
