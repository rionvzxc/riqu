package riqu

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
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

func TestResponseJSONRejectsTrailingData(t *testing.T) {
	for _, tt := range []struct {
		name     string
		trailing string
		wantErr  bool
	}{
		{"whitespace", " \t\r\n", false},
		{"invalid data", "garbage", true},
		{"another value", " null", true},
		{"incomplete value", " {", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := &testReadCloser{Reader: strings.NewReader(`{"ok":true}` + tt.trailing)}
			response := &QResponse{body: body}
			var data map[string]bool

			err := response.JSON(&data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("JSON returned %v, want error: %t", err, tt.wantErr)
			}
			if !data["ok"] {
				t.Fatal("expected first JSON value to be decoded")
			}
			if body.closeCount != 1 {
				t.Fatalf("expected body to be closed once, got %d", body.closeCount)
			}
		})
	}
}

func TestResponseFileUsesFallbackNameForRootURL(t *testing.T) {
	dir := t.TempDir()
	response := &QResponse{
		RequestUrl: "https://example.com/",
		body:       io.NopCloser(strings.NewReader("response body")),
	}
	if err := response.File(&TFileOptions{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "download"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "response body" {
		t.Fatalf("unexpected download content: %q", data)
	}
}

func TestResponseFileReplacesDestinationOnlyOnSuccess(t *testing.T) {
	readErr := errors.New("read failed")
	closeErr := errors.New("close failed")
	for _, tt := range []struct {
		name     string
		reader   io.Reader
		closeErr error
		wantErr  error
		wantFile string
	}{
		{"success", strings.NewReader("new content"), nil, nil, "new content"},
		{"read failure", io.MultiReader(strings.NewReader("partial"), iotest.ErrReader(readErr)), nil, readErr, "old content"},
		{"close failure", strings.NewReader("new content"), closeErr, closeErr, "old content"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			const filename = "download.txt"
			path := filepath.Join(dir, filename)
			if err := os.WriteFile(path, []byte("old content"), 0600); err != nil {
				t.Fatal(err)
			}
			body := &testReadCloser{Reader: tt.reader, closeErr: tt.closeErr}
			response := &QResponse{body: body}
			if err := response.File(&TFileOptions{Dir: dir, Filename: filename}); !errors.Is(err, tt.wantErr) {
				t.Fatalf("File returned %v, want %v", err, tt.wantErr)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tt.wantFile {
				t.Fatalf("destination contains %q, want %q", data, tt.wantFile)
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Name() != filename {
				t.Fatalf("unexpected download directory entries: %v", entries)
			}
			if body.closeCount != 1 {
				t.Fatalf("expected body to be closed once, got %d", body.closeCount)
			}
		})
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
