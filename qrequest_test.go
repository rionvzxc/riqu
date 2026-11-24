package riqu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
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
		SetAllCookies(TValues{"session": {"secret"}}).
		OnResponse(func(res *QResponse) error {
			responseBody, responseErr = res.Text()
			return nil
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
		SetAllHeaders(THeader{"Authorization": []string{"Bearer secret"}}).
		SetAllCookies(TValues{"session": {"secret"}}).
		OnResponse(func(res *QResponse) error {
			responseBody, responseErr = res.Text()
			return nil
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
		RedirectOptions(TRedirectOptions{Max: -1}).
		RedirectOptions(TRedirectOptions{Max: 1, Responses: VLastResponse}).
		OnResponse(func(res *QResponse) error {
			responseBody, responseErr = res.Text()
			return nil
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

func TestSendUsesBodyContentType(t *testing.T) {
	tests := []struct {
		name        string
		configure   func(*QRequest)
		contentType string
	}{
		{
			name:        "text",
			configure:   func(request *QRequest) { request.Text("plain text") },
			contentType: "text/plain",
		},
		{
			name:        "HTML",
			configure:   func(request *QRequest) { request.HTML("<p>html</p>") },
			contentType: "text/html",
		},
		{
			name:        "JSON",
			configure:   func(request *QRequest) { request.JSON(TValues{"name": {"riqu"}}) },
			contentType: "application/json",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contentType := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				contentType <- request.Header.Get("Content-Type")
				fmt.Fprint(w, "ok")
			}))
			defer server.Close()

			request := New().POST(server.URL)
			test.configure(request)
			if err := request.Send(); err != nil {
				t.Fatal(err)
			}
			if actual := <-contentType; actual != test.contentType {
				t.Fatalf("expected Content-Type %q, got %q", test.contentType, actual)
			}
		})
	}
}

func TestHeaderContentTypeOverridesBodyContentType(t *testing.T) {
	for _, isHeaderFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("header first=%t", isHeaderFirst), func(t *testing.T) {
			contentType := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				contentType <- request.Header.Get("Content-Type")
				fmt.Fprint(w, "ok")
			}))
			defer server.Close()

			request := New().POST(server.URL)
			if isHeaderFirst {
				request.SetAllHeaders(THeader{"Content-Type": []string{"application/custom"}}).Text("plain text")
			} else {
				request.Text("plain text").SetAllHeaders(THeader{"Content-Type": []string{"application/custom"}})
			}
			if err := request.Send(); err != nil {
				t.Fatal(err)
			}
			if actual := <-contentType; actual != "application/custom" {
				t.Fatalf("expected explicit Content-Type to win, got %q", actual)
			}
		})
	}
}

func TestBodySerializationIsDeferredUntilSend(t *testing.T) {
	request := New().POST("http://example.test").
		Text("plain text")
	if _, isValidType := request.body.value.(string); !isValidType {
		t.Fatal("Text body was converted to bytes before Send")
	}
	request.Form(TValues{"name": {"riqu"}})
	if _, isValidType := request.body.value.(TValues); !isValidType {
		t.Fatal("Form body was encoded before Send")
	}
	request.HTML("<p>html</p>")
	if _, isValidType := request.body.value.(string); !isValidType {
		t.Fatal("HTML body was converted to bytes before Send")
	}
}

type tCountingJSONMarshaler struct {
	calls *int
}

func (this tCountingJSONMarshaler) MarshalJSON() ([]byte, error) {
	*this.calls++
	return []byte(`{"value":"sent"}`), nil
}

func TestJSONMarshalingIsDeferredUntilSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(w, r.Body)
	}))
	defer server.Close()

	calls := 0
	request := New().POST(server.URL).JSON(tCountingJSONMarshaler{calls: &calls})
	if calls != 0 {
		t.Fatal("JSON was marshaled before Send")
	}
	if err := request.Send(); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected JSON to be marshaled once in Send, got %d calls", calls)
	}
}

func TestMultipartStreamsAndReplaysFilesOn307(t *testing.T) {
	type tRequestResult struct {
		path             string
		contentType      string
		contentLength    int64
		transferEncoding []string
		fields           []string
		fileContent      string
		err              error
	}

	contents := strings.Repeat("streamed multipart content\n", 8192)
	filePath := filepath.Join(t.TempDir(), "sample.txt")

	if err := os.WriteFile(filePath, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	results := make(chan tRequestResult, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		result := tRequestResult{
			path:             request.URL.Path,
			contentType:      request.Header.Get("Content-Type"),
			contentLength:    request.ContentLength,
			transferEncoding: append([]string(nil), request.TransferEncoding...),
		}
		result.err = request.ParseMultipartForm(1024)

		if result.err == nil {
			result.fields = append([]string(nil), request.MultipartForm.Value["name"]...)
			file, _, err := request.FormFile("upload")

			if err != nil {
				result.err = err
			} else {
				data, readErr := io.ReadAll(file)
				closeErr := file.Close()
				result.err = errors.Join(readErr, closeErr)
				result.fileContent = string(data)
			}
			if request.MultipartForm != nil {
				request.MultipartForm.RemoveAll()
			}
		}
		results <- result

		if request.URL.Path == "/start" {
			http.Redirect(w, request, "/final", http.StatusTemporaryRedirect)
			return
		}
		fmt.Fprint(w, "done")
	}))
	defer server.Close()

	const contentType = "multipart/form-data; boundary=riqu-test-boundary"
	var responseBody string
	var responseErr error
	err := New().POST(server.URL + "/start").
		SetAllHeaders(THeader{"Content-Type": []string{contentType}}).
		FormMultipart(TMultipart{
			Form: TValues{"name": {"riqu", "second"}},
			File: TValues{"upload": {filePath}},
		}).
		OnResponse(func(response *QResponse) error {
			responseBody, responseErr = response.Text()
			return nil
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
	for _, expectedPath := range []string{"/start", "/final"} {
		result := <-results

		if result.path != expectedPath {
			t.Fatalf("expected request to %q, got %q", expectedPath, result.path)
		}
		if result.err != nil {
			t.Fatalf("multipart request %q failed: %v", result.path, result.err)
		}
		if result.contentType != contentType {
			t.Fatalf("expected explicit Content-Type %q, got %q", contentType, result.contentType)
		}
		if result.contentLength != -1 || len(result.transferEncoding) == 0 {
			t.Fatalf("expected streamed request with unknown length, got length %d and transfer encoding %v", result.contentLength, result.transferEncoding)
		}
		if !reflect.DeepEqual(result.fields, []string{"riqu", "second"}) || result.fileContent != contents {
			t.Fatalf("multipart data was not replayed correctly for %q", result.path)
		}
	}
}

func TestMultipartAddsMissingBoundary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if err := request.ParseMultipartForm(1024); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, request.FormValue("name"))
	}))
	defer server.Close()

	for _, contentType := range []string{"multipart/form-data", `multipart/form-data; boundary=""`} {
		t.Run(contentType, func(t *testing.T) {
			var responseBody string
			var responseErr error
			err := New().POST(server.URL).
				SetAllHeaders(THeader{"Content-Type": {contentType}}).
				FormMultipart(TMultipart{Form: TValues{"name": {"riqu"}}}).
				OnResponse(func(response *QResponse) error {
					responseBody, responseErr = response.Text()
					return nil
				}).
				Send()
			if err != nil {
				t.Fatal(err)
			}
			if responseErr != nil {
				t.Fatal(responseErr)
			}
			if responseBody != "riqu" {
				t.Fatalf("expected server to parse the multipart form, got %q", responseBody)
			}
		})
	}
}

func TestMultipartMissingFileReturnsSendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		io.Copy(io.Discard, request.Body)
		fmt.Fprint(w, "done")
	}))
	defer server.Close()

	var callbackErr error
	request := New().POST(server.URL).
		FormMultipart(TMultipart{
			File: TValues{"upload": {filepath.Join(t.TempDir(), "missing.txt")}},
		}).
		OnError(func(err error) {
			callbackErr = err
		})
	err := request.Send()
	if err == nil {
		t.Fatal("expected missing multipart file to fail during Send")
	}
	if callbackErr == nil {
		t.Fatal("expected multipart stream error to reach OnError")
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
				request.RedirectOptions(TRedirectOptions{Max: -1})
			},
		},
		{
			name: "invalid redirect response mode",
			configure: func(request *QRequest) {
				request.RedirectOptions(TRedirectOptions{Max: 1, Responses: TResponseMode(2)})
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
				request.URL("")
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
		isContentTypePrefix bool
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
			name:        "empty text",
			contentType: "text/plain",
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
			configure:         func(request *QRequest) { request.FormMultipart(TMultipart{}) },
			contentType:       "multipart/form-data; boundary=",
			isContentTypePrefix: true,
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
				if test.isContentTypePrefix && !strings.HasPrefix(contentType, test.contentType) {
					http.Error(w, fmt.Sprintf("unexpected content type: %q", contentType), http.StatusBadRequest)
					return
				}
				if !test.isContentTypePrefix && contentType != test.contentType {
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
			err := request.OnResponse(func(response *QResponse) error {
				responseBody, responseErr = response.Text()
				return nil
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
		SetAllHeaders(THeader{"X-Test": []string{"value"}}).
		SetAllHeaders(nil).
		SetAllCookies(TValues{"session": {"secret"}}).
		SetAllCookies(nil).
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
		RedirectOptions(TRedirectOptions{}).
		OnResponse(func(response *QResponse) error {
			statusCode = response.StatusCode()
			return nil
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
		OnResponse(func(response *QResponse) error {
			responseBody, responseErr = response.Text()
			return nil
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
