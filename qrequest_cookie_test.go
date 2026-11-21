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
				Header(THeader{"Cookie": {"session=header; keep=value"}}).
				Cookie(TValues{"session": {"old"}}).CookieJar(jar).
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
