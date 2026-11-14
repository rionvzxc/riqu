package riqu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendPreservesCookieOnSameOriginRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, cookie.Value)
	}))
	defer server.Close()

	var responseBody string
	var responseErr error
	err := New().GET(server.URL + "/start").
		Cookie(TValue{"session": "secret"}).
		OnSuccess(func(res *QResponse) {
			responseBody, responseErr = res.Text()
		}).
		Send()
	if err != nil {
		t.Fatal(err)
	}
	if responseErr != nil {
		t.Fatal(responseErr)
	}
	if responseBody != "secret" {
		t.Fatalf("expected redirected request to include cookie, got %q", responseBody)
	}
}

func TestSendDoesNotRestoreSensitiveHeadersAfterCrossOriginRedirect(t *testing.T) {
	seenB := make(chan http.Header, 2)
	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenB <- r.Header.Clone()
		if r.URL.Path == "/step1" {
			http.Redirect(w, r, "/step2", http.StatusFound)
			return
		}
		fmt.Fprint(w, "done")
	}))
	defer serverB.Close()

	seenA := make(chan http.Header, 1)
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenA <- r.Header.Clone()
		http.Redirect(w, r, serverB.URL+"/step1", http.StatusFound)
	}))
	defer serverA.Close()

	var responseBody string
	var responseErr error
	err := New().GET(serverA.URL + "/start").
		Header(THeader{"Authorization": []string{"Bearer secret"}}).
		Cookie(TValue{"session": "secret"}).
		OnSuccess(func(res *QResponse) {
			responseBody, responseErr = res.Text()
		}).
		Send()
	if err != nil {
		t.Fatal(err)
	}
	if responseErr != nil {
		t.Fatal(responseErr)
	}
	if responseBody != "done" {
		t.Fatalf("unexpected final response: %q", responseBody)
	}
	if header := <-seenA; header.Get("Authorization") != "Bearer secret" || header.Get("Cookie") != "session=secret" {
		t.Fatalf("initial origin did not receive configured credentials: %#v", header)
	}
	for i := 0; i < 2; i++ {
		header := <-seenB
		if header.Get("Authorization") != "" || header.Get("Cookie") != "" {
			t.Fatalf("cross-origin redirect received sensitive headers: %#v", header)
		}
	}
}

func TestSendValidatesCurrentConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(w, r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	var responseBody string
	var responseErr error
	err := New().POST(server.URL).
		JSON(func() {}).
		Data([]byte("valid body")).
		Timeout(-time.Second).
		Timeout(time.Second).
		Redirect(TRedirect{Max: -1}).
		Redirect(TRedirect{Max: 1, ResponseMode: VRedirectResponseLast}).
		OnSuccess(func(res *QResponse) {
			responseBody, responseErr = res.Text()
		}).
		Send()
	if err != nil {
		t.Fatal(err)
	}
	if responseErr != nil {
		t.Fatal(responseErr)
	}
	if responseBody != "valid body" {
		t.Fatalf("expected latest body configuration to be used, got %q", responseBody)
	}
}

func TestSendRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*QRequest)
	}{
		{
			name: "negative timeout",
			configure: func(request *QRequest) {
				request.Timeout(-time.Second)
			},
		},
		{
			name: "negative redirect limit",
			configure: func(request *QRequest) {
				request.Redirect(TRedirect{Max: -1})
			},
		},
		{
			name: "invalid redirect response mode",
			configure: func(request *QRequest) {
				request.Redirect(TRedirect{Max: 1, ResponseMode: TRedirectResponseMode(2)})
			},
		},
		{
			name: "invalid JSON body",
			configure: func(request *QRequest) {
				request.JSON(func() {})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := New().GET("http://127.0.0.1:1")
			test.configure(request)
			if err := request.Send(); err == nil {
				t.Fatal("expected Send to return a configuration error")
			}
		})
	}
}

func TestSendTimeoutCoversRedirectChain(t *testing.T) {
	secondRequestStarted := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			time.Sleep(700 * time.Millisecond)
			http.Redirect(w, r, "/slow", http.StatusFound)
			return
		}
		secondRequestStarted <- struct{}{}
		<-r.Context().Done()
	}))
	defer server.Close()

	start := time.Now()
	err := New().GET(server.URL + "/start").
		Timeout(1600 * time.Millisecond).
		Send()
	elapsed := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline error, got %v", err)
	}
	select {
	case <-secondRequestStarted:
	default:
		t.Fatal("redirect target was not requested")
	}
	if elapsed >= 2*time.Second {
		t.Fatalf("timeout appears to have restarted for redirect: elapsed %v", elapsed)
	}
}
