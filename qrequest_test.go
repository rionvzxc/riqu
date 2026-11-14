package riqu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestSendRejectsBlankURLAndMethod(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*QRequest)
	}{
		{
			name: "blank URL",
			configure: func(request *QRequest) {
				request.Url("")
			},
		},
		{
			name: "blank method",
			configure: func(request *QRequest) {
				request.Method("")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := New().GET("http://127.0.0.1:1")
			test.configure(request)
			if err := request.Send(); err == nil {
				t.Fatal("expected Send to reject blank configuration")
			}
		})
	}
}

func TestSendAcceptsNilAndEmptyBodies(t *testing.T) {
	tests := []struct {
		name              string
		configure         func(*QRequest)
		contentType       string
		contentTypePrefix bool
		checkBody         func(*http.Request) error
	}{
		{
			name: "nil data",
			configure: func(request *QRequest) {
				request.Data(nil)
			},
			checkBody: func(request *http.Request) error {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					return err
				}
				if len(body) != 0 {
					return fmt.Errorf("expected empty body, got %q", body)
				}
				return nil
			},
		},
		{
			name: "empty text",
			configure: func(request *QRequest) {
				request.Text("")
			},
			checkBody: func(request *http.Request) error {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					return err
				}
				if len(body) != 0 {
					return fmt.Errorf("expected empty body, got %q", body)
				}
				return nil
			},
		},
		{
			name:        "nil form",
			configure:   func(request *QRequest) { request.Form(nil) },
			contentType: "application/x-www-form-urlencoded",
			checkBody: func(request *http.Request) error {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					return err
				}
				if len(body) != 0 {
					return fmt.Errorf("expected empty form body, got %q", body)
				}
				return nil
			},
		},
		{
			name:        "nil JSON",
			configure:   func(request *QRequest) { request.JSON(nil) },
			contentType: "application/json",
			checkBody: func(request *http.Request) error {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					return err
				}
				if string(body) != "null" {
					return fmt.Errorf("expected JSON null, got %q", body)
				}
				return nil
			},
		},
		{
			name:              "nil multipart maps",
			configure:         func(request *QRequest) { request.FormMultipart(nil, nil) },
			contentType:       "multipart/form-data; boundary=",
			contentTypePrefix: true,
			checkBody: func(request *http.Request) error {
				if err := request.ParseMultipartForm(1024); err != nil {
					return err
				}
				if len(request.MultipartForm.Value) != 0 || len(request.MultipartForm.File) != 0 {
					return fmt.Errorf("expected empty multipart form, got %#v", request.MultipartForm)
				}
				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				contentType := request.Header.Get("Content-Type")
				if test.contentTypePrefix && !strings.HasPrefix(contentType, test.contentType) {
					http.Error(w, fmt.Sprintf("unexpected content type: %q", contentType), http.StatusBadRequest)
					return
				}
				if !test.contentTypePrefix && contentType != test.contentType {
					http.Error(w, fmt.Sprintf("unexpected content type: %q", contentType), http.StatusBadRequest)
					return
				}
				if err := test.checkBody(request); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				fmt.Fprint(w, "ok")
			}))
			defer server.Close()

			var responseBody string
			var responseErr error
			request := New().POST(server.URL)
			test.configure(request)
			err := request.OnSuccess(func(response *QResponse) {
				responseBody, responseErr = response.Text()
			}).Send()
			if err != nil {
				t.Fatal(err)
			}
			if responseErr != nil {
				t.Fatal(responseErr)
			}
			if responseBody != "ok" {
				t.Fatalf("unexpected response: %q", responseBody)
			}
		})
	}
}

func TestSendClearsNilHeadersAndCookies(t *testing.T) {
	seenHeaders := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHeaders <- r.Header.Clone()
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	err := New().GET(server.URL).
		Header(THeader{"X-Test": []string{"value"}}).
		Header(nil).
		Cookie(TValue{"session": "secret"}).
		Cookie(nil).
		Send()
	if err != nil {
		t.Fatal(err)
	}
	header := <-seenHeaders
	if header.Get("X-Test") != "" || header.Get("Cookie") != "" {
		t.Fatalf("expected nil header and cookie inputs to clear values, got %#v", header)
	}
}

func TestSendZeroRedirectConfigurationDoesNotFollow(t *testing.T) {
	finalRequested := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		finalRequested <- struct{}{}
		fmt.Fprint(w, "final")
	}))
	defer server.Close()

	var statusCode int
	err := New().GET(server.URL + "/start").
		Redirect(TRedirect{}).
		OnSuccess(func(response *QResponse) {
			statusCode = response.StatusCode
		}).
		Send()
	if err != nil {
		t.Fatal(err)
	}
	if statusCode != http.StatusFound {
		t.Fatalf("expected redirect response status %d, got %d", http.StatusFound, statusCode)
	}
	select {
	case <-finalRequested:
		t.Fatal("zero redirect limit should not follow the redirect")
	default:
	}
}

func TestSendAllowsZeroTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond)
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	var responseBody string
	var responseErr error
	err := New().GET(server.URL).
		Timeout(0).
		OnSuccess(func(response *QResponse) {
			responseBody, responseErr = response.Text()
		}).
		Send()
	if err != nil {
		t.Fatal(err)
	}
	if responseErr != nil {
		t.Fatal(responseErr)
	}
	if responseBody != "ok" {
		t.Fatalf("unexpected response: %q", responseBody)
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
