package riqu

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type testReadCloser struct {
	io.Reader
	closeErr   error
	closeCount int
}

func (this *testReadCloser) Close() error {
	this.closeCount++
	return this.closeErr
}

func TestResponseDataReturnsCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	body := &testReadCloser{
		Reader:   strings.NewReader("response body"),
		closeErr: closeErr,
	}
	response := &QResponse{body: body}

	data, err := response.Data()
	if string(data) != "response body" {
		t.Fatalf("unexpected response data: %q", data)
	}
	if !errors.Is(err, closeErr) {
		t.Fatalf("expected close error, got %v", err)
	}
	if body.closeCount != 1 {
		t.Fatalf("expected body to be closed once, got %d", body.closeCount)
	}
}

func TestResponseJSONReturnsCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	body := &testReadCloser{
		Reader:   strings.NewReader(`{"ok":true}`),
		closeErr: closeErr,
	}
	response := &QResponse{body: body}
	var data struct {
		OK bool `json:"ok"`
	}

	err := response.JSON(&data)
	if !data.OK {
		t.Fatal("expected JSON response to be decoded")
	}
	if !errors.Is(err, closeErr) {
		t.Fatalf("expected close error, got %v", err)
	}
	if body.closeCount != 1 {
		t.Fatalf("expected body to be closed once, got %d", body.closeCount)
	}
}

func TestHandleResponseReturnsCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	body := &testReadCloser{
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

func TestHandleResponseClosesBodyWhenCallbackPanics(t *testing.T) {
	body := &testReadCloser{Reader: strings.NewReader("response body")}
	response := &QResponse{body: body}
	request := New().OnSuccess(func(*QResponse) {
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
