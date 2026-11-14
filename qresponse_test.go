package riqu

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestNilResponseMethodsReturnErrors(t *testing.T) {
	var response *QResponse

	if _, err := response.Data(); err == nil {
		t.Fatal("expected Data on nil response to fail")
	}
	if _, err := response.Text(); err == nil {
		t.Fatal("expected Text on nil response to fail")
	}
	if err := response.JSON(nil); err == nil {
		t.Fatal("expected JSON on nil response to fail")
	}
	if err := response.File(nil); err == nil {
		t.Fatal("expected File on nil response to fail")
	}
}

func TestResponseHandlesUnavailableAndEmptyBodies(t *testing.T) {
	response := &QResponse{}
	if _, err := response.Data(); err == nil {
		t.Fatal("expected Data without a body to fail")
	}
	if err := response.JSON(nil); err == nil {
		t.Fatal("expected JSON without a body to fail")
	}
	if err := response.File(nil); err == nil {
		t.Fatal("expected File without a body to fail")
	}

	response = &QResponse{body: io.NopCloser(strings.NewReader(""))}
	data, err := response.Data()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("expected empty data, got %q", data)
	}
	if _, err := response.Data(); err == nil {
		t.Fatal("expected second body read to fail")
	}

	response = &QResponse{body: io.NopCloser(strings.NewReader(""))}
	var jsonData any
	if err := response.JSON(&jsonData); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF for empty JSON response, got %v", err)
	}
}

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
