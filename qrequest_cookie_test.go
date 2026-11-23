package riqu

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
				OnSuccess(func(response *QResponse) { text, readErr = response.Text() })
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
