package riqu

import (
	"errors"
	"strings"
	"testing"
)

func TestHandleResponseReturnsCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	body := &tTestReadCloser{
		Reader:   strings.NewReader("response body"),
		closeErr: closeErr,
	}
	response := &QResponse{body: body}

	err := New().handleResponse(response)
	if !errors.Is(err, closeErr) {
		t.Fatalf("expected close error, got %v", err)
	}
	if body.closeCount != 1 {
		t.Fatalf("expected body to be closed once, got %d", body.closeCount)
	}
}

func TestHandleResponsePreservesCallbackError(t *testing.T) {
	callbackErr := errors.New("callback failed")
	for _, closeErr := range []error{nil, errors.New("close failed")} {
		body := &tTestReadCloser{Reader: strings.NewReader("response body"), closeErr: closeErr}
		request := New().OnResponse(func(*QResponse) error {
			return callbackErr
		})
		if err := request.handleResponse(&QResponse{body: body}); err != callbackErr {
			t.Fatalf("expected callback error, got %v", err)
		}
		if body.closeCount != 1 {
			t.Fatalf("expected body closed once, got %d", body.closeCount)
		}
	}
}

func TestHandleResponseClosesBodyWhenCallbackPanics(t *testing.T) {
	body := &tTestReadCloser{Reader: strings.NewReader("response body")}
	response := &QResponse{body: body}
	request := New().OnResponse(func(*QResponse) error {
		panic("callback panic")
	})

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected callback panic")
			}
		}()
		request.handleResponse(response)
	}()

	if body.closeCount != 1 {
		t.Fatalf("expected body to be closed once, got %d", body.closeCount)
	}
}
