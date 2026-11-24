package riqu

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/basic" {
			username, password, isValid := request.BasicAuth()
			if !isValid || username != "user" || password != "pass:word" {
				t.Errorf("unexpected Basic credentials: %q, %q, %t", username, password, isValid)
			}
		} else if request.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("unexpected Bearer header: %q", request.Header.Get("Authorization"))
		}
		if len(request.Header.Values("Authorization")) != 1 {
			t.Error("expected one Authorization value")
		}
	}))
	defer server.Close()
	request := New().BearerAuth("old").BasicAuth("user", "pass:word")
	if err := request.GET(server.URL + "/basic").Send(); err != nil {
		t.Fatal(err)
	}
	if err := request.BearerAuth("token").GET(server.URL + "/bearer").Send(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthenticationRedirectScope(t *testing.T) {
	for _, configure := range []func(*QRequest){
		func(request *QRequest) { request.BasicAuth("user", "password") },
		func(request *QRequest) { request.BearerAuth("token") },
	} {
		other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			if request.Header.Get("Authorization") != "" {
				t.Error("cross-origin redirect received credentials")
			}
		}))
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			if request.Header.Get("Authorization") == "" {
				t.Error("same-origin request lost credentials")
			}
			if request.URL.Path == "/" {
				http.Redirect(w, request, "/same", http.StatusFound)
			} else {
				http.Redirect(w, request, other.URL, http.StatusFound)
			}
		}))
		request := New().GET(server.URL)
		configure(request)
		err := request.Send()
		server.Close()
		other.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}
