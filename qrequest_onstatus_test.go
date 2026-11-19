package riqu

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

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
	for _, mode := range []TRedirectResponseMode{VRedirectResponseAll, VRedirectResponseLast} {
		var called []int
		fallback := 0
		request := New().GET(server.URL).
			Redirect(TRedirect{Max: 1, ResponseMode: mode}).
			OnSuccess(func(*QResponse) { fallback++ }).
			OnStatuses([]int{302, 404, 404}, func(response *QResponse) {
				called = append(called, response.StatusCode)
			}).
			OnStatus(404, func(response *QResponse) {
				called = append(called, 404)
				text, err := response.Text()
				if err != nil || text != "missing" {
					t.Errorf("unexpected body: %q, %v", text, err)
				}
			})
		if err := request.Send(); err != nil {
			t.Fatal(err)
		}
		want := []int{404}
		if mode == VRedirectResponseAll {
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
