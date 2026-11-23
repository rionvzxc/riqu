package riqu

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseCookiesAndBinding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &TCookie{Name: "session", Value: "one", Path: "/"})
		http.SetCookie(w, &TCookie{Name: "session", Value: "two", Path: "/other"})
		http.SetCookie(w, &TCookie{Name: "count", Value: "12"})
	}))
	defer server.Close()
	err := New().GET(server.URL).OnResponse(func(response *QResponse) error {
		cookie := response.Cookie("session")
		if cookie == nil || cookie.Value != "one" || cookie.Path != "/" || response.Cookie("missing") != nil {
			t.Fatal("unexpected first cookie")
		}
		if len(response.Cookies("session")) != 2 || len(response.Cookies("missing")) != 0 || len(response.AllCookies()) != 3 {
			t.Fatal("unexpected cookie lists")
		}
		var target struct {
			Session []string `cookie:"session"`
			Count   int      `cookie:"count"`
		}
		if err := response.BindCookie(&target); err != nil {
			t.Fatal(err)
		}
		if target.Count != 12 || len(target.Session) != 2 || target.Session[1] != "two" {
			t.Fatalf("unexpected cookie binding: %+v", target)
		}
		return nil
	}).Send()
	if err != nil {
		t.Fatal(err)
	}
}

func TestResponseCookieCopies(t *testing.T) {
	response := &QResponse{resCookie: []*TCookie{
		{Name: "session", Value: "original", Unparsed: []string{"original"}},
		nil,
	}}
	for _, cookie := range []*TCookie{
		response.Cookie("session"),
		response.Cookies("session")[0],
		response.AllCookies()[0],
	} {
		cookie.Value = "changed"
		cookie.Unparsed[0] = "changed"
	}
	all := response.AllCookies()
	all[0] = nil
	if response.resCookie[0].Value != "original" || response.resCookie[0].Unparsed[0] != "original" || all[1] != nil {
		t.Fatal("cookie getters must not expose internal cookie state")
	}
}
