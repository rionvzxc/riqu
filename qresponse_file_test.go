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

func TestResponseFileInvalidResponse(t *testing.T) {
	var response *QResponse
	if err := response.File(nil); err == nil {
		t.Fatal("expected File on nil response to fail")
	}
	response = &QResponse{}
	if err := response.File(nil); err == nil {
		t.Fatal("expected File without a body to fail")
	}
}

func TestResponseFileUsesFallbackNameForRootURL(t *testing.T) {
	dir := t.TempDir()
	response := &QResponse{
		resUrl: "https://example.com/",
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
			body := &tTestReadCloser{Reader: tt.reader, closeErr: tt.closeErr}
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
