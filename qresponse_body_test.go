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
	if err := response.BindJSON(nil); err == nil {
		t.Fatal("expected JSON on nil response to fail")
	}
}

func TestResponseHandlesUnavailableAndEmptyBodies(t *testing.T) {
	response := &QResponse{}
	if _, err := response.Data(); err == nil {
		t.Fatal("expected Data without a body to fail")
	}
	if err := response.BindJSON(nil); err == nil {
		t.Fatal("expected JSON without a body to fail")
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
	if err := response.BindJSON(&jsonData); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF for empty JSON response, got %v", err)
	}
}

func TestResponseDataReturnsCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	body := &tTestReadCloser{
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

func TestResponseBindJSONReturnsCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	body := &tTestReadCloser{
		Reader:   strings.NewReader(`{"ok":true}`),
		closeErr: closeErr,
	}
	response := &QResponse{body: body}
	var data struct {
		OK bool `json:"ok"`
	}

	err := response.BindJSON(&data)
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

func TestResponseBindJSONRejectsTrailingData(t *testing.T) {
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
			body := &tTestReadCloser{Reader: strings.NewReader(`{"ok":true}` + tt.trailing)}
			response := &QResponse{body: body}
			var data map[string]bool

			err := response.BindJSON(&data)
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

type tTestReadCloser struct {
	io.Reader
	closeErr   error
	closeCount int
}

func (this *tTestReadCloser) Close() error {
	this.closeCount++
	return this.closeErr
}
