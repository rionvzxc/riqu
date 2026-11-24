package riqu

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"time"
)

type tTestStaleUploadConn struct {
	net.Conn
	writeCount atomic.Int32
}

func (this *tTestStaleUploadConn) Write(data []byte) (int, error) {
	if this.writeCount.Add(1) > 1 {
		return 0, errors.New("simulated stale pooled connection")
	}
	return this.Conn.Write(data)
}

func TestUploadProgressPreservesTransportRetry(t *testing.T) {
	for _, isProgressEnabled := range []bool{false, true} {
		t.Run(fmt.Sprint(isProgressEnabled), func(t *testing.T) {
			var dialCount atomic.Int32
			payloads := make(chan string, 2)
			transport := &QTransport{transport: &http.Transport{
				DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					client, server := net.Pipe()
					t.Cleanup(func() { client.Close(); server.Close() })
					isFailing := dialCount.Add(1) == 1
					go func() {
						defer server.Close()
						reader := bufio.NewReader(server)
						for {
							request, err := http.ReadRequest(reader)
							if err != nil {
								return
							}
							data, readErr := io.ReadAll(request.Body)
							request.Body.Close()
							if readErr != nil {
								return
							}
							if request.Method == "POST" {
								payloads <- string(data)
							}
							if _, err := io.WriteString(server, "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"); err != nil {
								return
							}
						}
					}()
					if isFailing {
						return &tTestStaleUploadConn{Conn: client}, nil
					}
					return client, nil
				},
			}}
			defer transport.CloseIdleConnections()
			if err := New().GET("http://audit.invalid").Transport(transport).Send(); err != nil {
				t.Fatal(err)
			}
			var mutex sync.Mutex
			var ends []TTransferInfo
			starts, progress := 0, 0
			request := New().POST("http://audit.invalid").Transport(transport).
				Data([]byte("payload")).SetHeader("Idempotency-Key", "retry-test")
			if isProgressEnabled {
				request.OnUploadStart(func(info TTransferInfo) {
					mutex.Lock()
					defer mutex.Unlock()
					starts++
					if info.Transferred != 0 {
						t.Error("replay did not reset byte count")
					}
				}).OnUploadProgress(0, func(TTransferInfo) {
					mutex.Lock()
					defer mutex.Unlock()
					progress++
				}).OnUploadEnd(func(info TTransferInfo) {
					mutex.Lock()
					defer mutex.Unlock()
					ends = append(ends, info)
				})
			}
			if err := request.Send(); err != nil {
				t.Fatalf("retry failed with progress=%v: %v", isProgressEnabled, err)
			}
			if dialCount.Load() != 2 {
				t.Fatalf("dials=%d, want 2", dialCount.Load())
			}
			select {
			case payload := <-payloads:
				if payload != "payload" {
					t.Errorf("replayed body=%q", payload)
				}
			default:
				t.Fatal("server did not receive replayed body")
			}
			mutex.Lock()
			defer mutex.Unlock()
			if isProgressEnabled {
				if starts == 0 || starts != len(ends) || progress == 0 {
					t.Fatalf("starts=%d ends=%d progress=%d", starts, len(ends), progress)
				}
				last := ends[len(ends)-1]
				if last.Transferred != 7 || last.Total != 7 || last.Err != nil {
					t.Fatalf("replayed transfer=%+v", last)
				}
			}
		})
	}
}

func TestUploadEventsIgnoreUnusedRedirectReplay(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		data, err := io.ReadAll(request.Body)
		if err != nil || string(data) != "payload" {
			t.Errorf("body=%q error=%v", data, err)
		}
		if request.URL.Path == "/" {
			http.Redirect(w, request, "/final", http.StatusTemporaryRedirect)
		}
	}))
	defer server.Close()
	var mutex sync.Mutex
	var events []string
	err := New().POST(server.URL).Data([]byte("payload")).
		OnUploadStart(func(TTransferInfo) { mutex.Lock(); defer mutex.Unlock(); events = append(events, "start") }).
		OnUploadEnd(func(info TTransferInfo) {
			mutex.Lock()
			defer mutex.Unlock()
			events = append(events, "end")
			if info.Transferred != 7 || info.Err != nil {
				t.Errorf("end=%+v", info)
			}
		}).Send()
	mutex.Lock()
	defer mutex.Unlock()
	if err != nil || strings.Join(events, ",") != "start,end,start,end" {
		t.Fatalf("events=%v error=%v", events, err)
	}
}

func TestUploadEndOnReaderError(t *testing.T) {
	want := errors.New("source read failed")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.Copy(io.Discard, r.Body) }))
	defer server.Close()
	var end TTransferInfo
	ends := 0
	err := New().POST(server.URL).BodyReader(io.MultiReader(strings.NewReader("partial"), iotest.ErrReader(want))).
		OnUploadEnd(func(info TTransferInfo) { end = info; ends++ }).Send()
	if !errors.Is(err, want) || !errors.Is(end.Err, want) || ends != 1 || end.Transferred != 7 {
		t.Fatalf("error=%v end=%+v ends=%d", err, end, ends)
	}
}

func TestDownloadEventsPerRedirectResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		io.WriteString(w, "final")
	}))
	defer server.Close()
	for _, mode := range []TResponseMode{VLastResponse, VAllResponses} {
		var ends []TTransferInfo
		err := New().GET(server.URL).RedirectOptions(TRedirectOptions{Max: 1, Responses: mode}).
			OnDownloadEnd(func(info TTransferInfo) { ends = append(ends, info) }).
			OnResponse(func(res *QResponse) error { _, err := res.Data(); return err }).Send()
		want := 1
		if mode == VAllResponses {
			want = 2
		}
		if err != nil || len(ends) != want {
			t.Fatalf("mode=%v error=%v ends=%v", mode, err, ends)
		}
		last := ends[len(ends)-1]
		if last.URL != server.URL+"/final" || last.Transferred != 5 || last.Err != nil {
			t.Fatalf("last=%+v", last)
		}
	}
}

func TestTransferEventsAndIntervals(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.Copy(w, r.Body) }))
	defer server.Close()
	for _, interval := range []time.Duration{0, time.Hour} {
		var mutex sync.Mutex
		var events []string
		var ends []TTransferInfo
		progress := 0
		record := func(event string) func(TTransferInfo) {
			return func(info TTransferInfo) {
				mutex.Lock()
				defer mutex.Unlock()
				events = append(events, event)
				if strings.HasSuffix(event, "end") {
					ends = append(ends, info)
				}
				if strings.HasSuffix(event, "progress") {
					progress++
				}
			}
		}
		err := New().POST(server.URL).Data([]byte("payload")).
			OnUploadStart(record("upload-start")).OnUploadProgress(interval, record("upload-progress")).OnUploadEnd(record("upload-end")).
			OnDownloadStart(record("download-start")).OnDownloadProgress(interval, record("download-progress")).OnDownloadEnd(record("download-end")).
			OnResponse(func(res *QResponse) error {
				data, err := res.Text()
				if data != "payload" {
					t.Errorf("body=%q", data)
				}
				return err
			}).Send()
		if err != nil {
			t.Fatal(err)
		}
		if len(ends) != 2 || events[0] != "upload-start" || events[len(events)-1] != "download-end" {
			t.Fatalf("events=%v ends=%v", events, ends)
		}
		for _, info := range ends {
			if info.Transferred != 7 || info.Err != nil || info.Method != "POST" {
				t.Errorf("end=%+v", info)
			}
		}
		if interval == 0 && progress == 0 || interval > 0 && progress != 0 {
			t.Fatalf("progress=%d interval=%v", progress, interval)
		}
	}
}

func TestTransferEndIncludesConsumerErrorAndSkippedBodyHasNoEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "payload") }))
	defer server.Close()
	want := errors.New("writer failed")
	var end TTransferInfo
	starts, ends := 0, 0
	request := New().GET(server.URL).OnDownloadStart(func(TTransferInfo) { starts++ }).OnDownloadEnd(func(info TTransferInfo) { ends++; end = info })
	if err := request.Send(); err != nil || starts != 0 || ends != 0 {
		t.Fatalf("unread body events: %d %d %v", starts, ends, err)
	}
	err := request.OnResponse(func(res *QResponse) error {
		_, err := res.WriteTo(tTestWriterFunc(func([]byte) (int, error) { return 0, want }))
		return err
	}).Send()
	if !errors.Is(err, want) || !errors.Is(end.Err, want) || starts != 1 || ends != 1 {
		t.Fatalf("err=%v end=%+v starts=%d ends=%d", err, end, starts, ends)
	}
	if err := New().GET(server.URL).OnUploadProgress(-1, nil).Send(); err == nil {
		t.Fatal("expected negative progress interval error")
	}
}
