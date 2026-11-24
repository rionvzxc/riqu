package riqu

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaxResponseSizeAcrossConsumers(t *testing.T) {
	for _, consumer := range []string{"data", "text", "json", "file", "writer"} {
		for _, limit := range []int64{0, 4, 5} {
			t.Run(fmt.Sprintf("%s/%d", consumer, limit), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `"abc"`) }))
				defer server.Close()
				dir := t.TempDir()
				path := filepath.Join(dir, "result")
				os.WriteFile(path, []byte("old"), 0600)
				failures := 0
				err := New().GET(server.URL).MaxResponseSize(limit).OnResponse(func(res *QResponse) error {
					switch consumer {
					case "data":
						_, err := res.Data()
						return err
					case "text":
						_, err := res.Text()
						return err
					case "json":
						var value string
						return res.BindJSON(&value)
					case "file":
						return res.File(&TFileOptions{Dir: dir, Filename: "result"})
					default:
						_, err := res.WriteTo(io.Discard)
						return err
					}
				}).OnError(func(error) { failures++ }).Send()
				if limit == 4 {
					if err == nil || err.Error() != "riqu: response body exceeds maximum size" || failures != 1 {
						t.Fatalf("err=%v failures=%d", err, failures)
					}
					if consumer == "file" {
						data, _ := os.ReadFile(path)
						if string(data) != "old" {
							t.Fatal("old file changed")
						}
					}
				} else if err != nil || failures != 0 {
					t.Fatalf("err=%v failures=%d", err, failures)
				}
			})
		}
	}
}

func TestMaxResponseSizeAfterGzipAndValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(w)
		io.WriteString(writer, strings.Repeat("x", 4096))
		writer.Close()
	}))
	defer server.Close()
	err := New().GET(server.URL).MaxResponseSize(1024).OnResponse(func(res *QResponse) error { _, err := res.Data(); return err }).Send()
	if err == nil || err.Error() != "riqu: response body exceeds maximum size" {
		t.Fatal(err)
	}
	if err := New().GET(server.URL).MaxResponseSize(-1).Send(); err == nil {
		t.Fatal("expected negative limit error")
	}
}
