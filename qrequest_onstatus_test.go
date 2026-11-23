package riqu

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestResponseCallbackErrors(t *testing.T) {
	callbackErr := errors.New("callback failed")
	for _, handler := range []string{"response", "status", "statuses"} {
		for _, mode := range []TResponseMode{VLastResponse, VAllResponses} {
			t.Run(fmt.Sprintf("%s/%d", handler, mode), func(t *testing.T) {
				requests, callbacks, errorCallbacks := 0, 0, 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
					requests++
					if request.URL.Path == "/" {
						http.Redirect(w, request, "/final", http.StatusFound)
						return
					}
					w.WriteHeader(http.StatusOK)
				}))
				defer server.Close()
				request := New().GET(server.URL).
					RedirectOptions(TRedirectOptions{Max: 1, Responses: mode}).
					OnError(func(err error) {
						errorCallbacks++
						if err != callbackErr {
							t.Errorf("unexpected OnError value: %v", err)
						}
					})
				callback := func(response *QResponse) error {
					callbacks++
					return callbackErr
				}
				switch handler {
				case "response":
					request.OnResponse(callback)
				case "status":
					request.OnStatus(302, callback).OnStatus(200, callback)
				case "statuses":
					request.OnStatuses([]int{302, 200}, callback)
				}
				if err := request.Send(); err != callbackErr {
					t.Fatalf("Send returned %v, want callback error", err)
				}
				wantRequests := 2
				if mode == VAllResponses {
					wantRequests = 1
				}
				if callbacks != 1 || errorCallbacks != 1 || requests != wantRequests {
					t.Fatalf("unexpected counts: response %d, error %d, requests %d", callbacks, errorCallbacks, requests)
				}
			})
		}
	}
}

func TestResponseBindingErrorWithoutOnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		fmt.Fprint(w, `{"ok":true}garbage`)
	}))
	defer server.Close()
	var result TMap
	err := New().GET(server.URL).OnResponse(func(response *QResponse) error {
		return response.BindJSON(&result)
	}).Send()
	if err == nil {
		t.Fatal("expected JSON error from Send without OnError")
	}
}

func TestStatusHandlersDispatchOnceAndFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/" {
			http.Redirect(w, request, "/final", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "missing")
	}))
	defer server.Close()
	for _, mode := range []TResponseMode{VAllResponses, VLastResponse} {
		var called []int
		fallback := 0
		request := New().GET(server.URL).
			RedirectOptions(TRedirectOptions{Max: 1, Responses: mode}).
			OnResponse(func(*QResponse) error { fallback++; return nil }).
			OnStatuses([]int{302, 404, 404}, func(response *QResponse) error {
				called = append(called, response.StatusCode())
				return nil
			}).
			OnStatus(404, func(response *QResponse) error {
				called = append(called, 404)
				text, err := response.Text()
				if err != nil || text != "missing" {
					t.Errorf("unexpected body: %q, %v", text, err)
				}
				return nil
			})
		if err := request.Send(); err != nil {
			t.Fatal(err)
		}
		want := []int{404}
		if mode == VAllResponses {
			want = []int{302, 404}
		}
		if !reflect.DeepEqual(called, want) || fallback != 0 {
			t.Fatalf("unexpected dispatch: %v, fallback %d", called, fallback)
		}
		request.OnStatus(404, nil)
		if err := request.Send(); err != nil || fallback != 1 {
			t.Fatalf("fallback failed: %v, count %d", err, fallback)
		}
	}
}
