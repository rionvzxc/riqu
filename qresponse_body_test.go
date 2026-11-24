package riqu

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

var _ io.WriterTo = (*QResponse)(nil)

type tTestStreamWriter struct {
	bytes.Buffer
	closeCount int
}

func (this *tTestStreamWriter) Close() error {
	this.closeCount++
	return nil
}

type tTestWriterFunc func([]byte) (int, error)

func (this tTestWriterFunc) Write(data []byte) (int, error) {
	return this(data)
}

func TestResponseWriteTo(t *testing.T) {
	body := &tTestReadCloser{Reader: strings.NewReader("payload")}
	response := &QResponse{body: body}
	writer := &tTestStreamWriter{}
	written, err := response.WriteTo(writer)

	if err != nil || written != 7 || writer.String() != "payload" {
		t.Fatalf("written=%d, data=%q, error=%v", written, writer.String(), err)
	}
	if body.closeCount != 1 || writer.closeCount != 0 {
		t.Fatalf("body closes=%d, writer closes=%d", body.closeCount, writer.closeCount)
	}
	if _, err := response.WriteTo(writer); err == nil {
		t.Fatal("expected second read to fail")
	}
}

func TestResponseWriteToPreservesErrorsAndCount(t *testing.T) {
	writeErr := errors.New("write failed")
	closeErr := errors.New("close failed")
	body := &tTestReadCloser{Reader: strings.NewReader("payload"), closeErr: closeErr}
	response := &QResponse{body: body}
	written, err := response.WriteTo(tTestWriterFunc(func(data []byte) (int, error) {
		return 2, writeErr
	}))

	if written != 2 || !errors.Is(err, writeErr) || !errors.Is(err, closeErr) || body.closeCount != 1 {
		t.Fatalf("written=%d, error=%v, closes=%d", written, err, body.closeCount)
	}
}

func TestResponseWriteToReadErrorAndShortWrite(t *testing.T) {
	readErr := errors.New("read failed")
	body := &tTestReadCloser{Reader: io.MultiReader(strings.NewReader("partial"), iotest.ErrReader(readErr))}
	var buffer bytes.Buffer
	written, err := (&QResponse{body: body}).WriteTo(&buffer)

	if written != 7 || buffer.String() != "partial" || !errors.Is(err, readErr) || body.closeCount != 1 {
		t.Fatalf("written=%d, data=%q, error=%v, closes=%d", written, buffer.String(), err, body.closeCount)
	}
	body = &tTestReadCloser{Reader: strings.NewReader("payload")}
	written, err = (&QResponse{body: body}).WriteTo(tTestWriterFunc(func([]byte) (int, error) {
		return 1, nil
	}))

	if written != 1 || !errors.Is(err, io.ErrShortWrite) || body.closeCount != 1 {
		t.Fatalf("written=%d, error=%v, closes=%d", written, err, body.closeCount)
	}
}

func TestResponseWriteToClosesBodyOnPanic(t *testing.T) {
	body := &tTestReadCloser{Reader: strings.NewReader("payload")}
	response := &QResponse{body: body}
	var recovered any

	func() {
		defer func() { recovered = recover() }()
		response.WriteTo(tTestWriterFunc(func([]byte) (int, error) {
			panic("writer panic")
		}))
	}()

	if recovered != "writer panic" || body.closeCount != 1 {
		t.Fatalf("panic=%v, closes=%d", recovered, body.closeCount)
	}
}

func TestResponseWriteToInvalidInputs(t *testing.T) {
	var response *QResponse

	if _, err := response.WriteTo(io.Discard); err == nil {
		t.Fatal("expected nil response error")
	}
	if _, err := (&QResponse{}).WriteTo(io.Discard); err == nil {
		t.Fatal("expected unavailable body error")
	}
	body := &tTestReadCloser{Reader: strings.NewReader("payload")}
	response = &QResponse{body: body}

	if _, err := response.WriteTo(nil); err == nil || response.bodyRead || body.closeCount != 0 {
		t.Fatal("nil writer must fail without consuming the body")
	}
	if _, err := response.WriteTo(io.Discard); err != nil {
		t.Fatal(err)
	}
}

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
		isErrorExpected  bool
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
			if (err != nil) != tt.isErrorExpected {
				t.Fatalf("JSON returned %v, want error: %t", err, tt.isErrorExpected)
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

type tTestPanicJSON struct{}

func (this *tTestPanicJSON) UnmarshalJSON(data []byte) error {
	panic("custom JSON decoder panic")
}

func TestResponseBindJSONClosesBodyWhenDecoderPanics(t *testing.T) {
	for _, isThroughCallback := range []bool{false, true} {
		body := &tTestReadCloser{Reader: strings.NewReader(`{"ok":true}`)}
		response := &QResponse{body: body}
		var target tTestPanicJSON
		var recovered any

		func() {
			defer func() {
				recovered = recover()
			}()

			if isThroughCallback {
				New().OnResponse(func(res *QResponse) error {
					return res.BindJSON(&target)
				}).handleResponse(response)
			} else {
				response.BindJSON(&target)
			}
		}()

		if recovered != "custom JSON decoder panic" {
			t.Fatalf("unexpected panic: %v", recovered)
		}
		if body.closeCount != 1 {
			t.Fatalf("expected body closed once, got %d (callback: %v)", body.closeCount, isThroughCallback)
		}
		if _, err := response.Data(); err == nil {
			t.Fatal("expected body to remain marked as consumed")
		}
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
