package riqu

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestCanceledContextAndReuse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	request := New().GET(server.URL).Context(ctx).OnError(func(err error) { calls++ })
	if err := request.Send(); !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("error=%v, callbacks=%d", err, calls)
	}
	if err := request.Context(nil).Send(); err != nil {
		t.Fatal(err)
	}
}

func TestRequestContextCancelsBodyAndRedirect(t *testing.T) {
	for _, isRedirect := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isRedirect {
				if r.URL.Path != "/" {
					t.Error("redirect continued after cancellation")
				}
				http.Redirect(w, r, "/final", 302)
				return
			}
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		}))
		calls := 0
		err := New().GET(server.URL).Context(ctx).
			RedirectOptions(TRedirectOptions{Max: 2, Responses: VAllResponses}).
			OnResponse(func(res *QResponse) error {
				cancel()
				if isRedirect {
					return nil
				}
				_, err := res.Data()
				return err
			}).OnError(func(error) { calls++ }).Send()
		cancel()
		server.Close()
		if !errors.Is(err, context.Canceled) || calls != 1 {
			t.Fatalf("redirect=%v: error=%v, callbacks=%d", isRedirect, err, calls)
		}
	}
}

func TestRequestContextDeadlineWithTimeoutDisabled(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if err := New().GET("http://unused.invalid").Context(ctx).Timeout(0).Send(); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected parent deadline error, got %v", err)
	}
}

func TestRequestContextAndTimeoutUseEarlierDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	for _, isParentFirst := range []bool{false, true} {
		parentTimeout, requestTimeout := 5*time.Second, 20*time.Millisecond

		if isParentFirst {
			parentTimeout, requestTimeout = requestTimeout, parentTimeout
		}
		ctx, cancel := context.WithTimeout(context.Background(), parentTimeout)
		err := New().GET(server.URL).Context(ctx).Timeout(requestTimeout).Send()
		parentErr := ctx.Err()
		cancel()

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected deadline error, got %v", err)
		}
		if !isParentFirst && parentErr != nil {
			t.Fatalf("request timeout canceled parent context: %v", parentErr)
		}
	}
}
