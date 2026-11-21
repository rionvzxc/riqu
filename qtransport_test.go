package riqu

import (
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTransportTLSVerification(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		fmt.Fprint(w, "secure")
	}))
	defer server.Close()
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if err := os.WriteFile(caFile, certPEM, 0600); err != nil {
		t.Fatal(err)
	}
	transport, err := NewTransport(&TTransportOptions{CAFile: caFile})
	if err != nil {
		t.Fatal(err)
	}
	defer transport.CloseIdleConnections()
	var text string
	var readErr error
	err = New().GET(server.URL).Transport(transport).
		OnSuccess(func(response *QResponse) { text, readErr = response.Text() }).Send()
	if err != nil || readErr != nil || text != "secure" {
		t.Fatalf("trusted TLS failed: %q, %v, %v", text, err, readErr)
	}
	standard, err := NewTransport(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer standard.CloseIdleConnections()
	errorCalls := 0
	err = New().GET(server.URL).Transport(standard).OnError(func(error) { errorCalls++ }).Send()
	if err == nil || errorCalls != 1 {
		t.Fatalf("expected TLS verification failure: %v, callbacks %d", err, errorCalls)
	}
}

func TestRequestAcceptsEmptyTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()
	var transport QTransport
	var text string
	var readErr error
	err := New().GET(server.URL).Transport(&transport).
		OnSuccess(func(response *QResponse) { text, readErr = response.Text() }).Send()
	if err != nil || readErr != nil || text != "ok" {
		t.Fatalf("empty transport failed: %q, %v, %v", text, err, readErr)
	}
}

func TestTransportHTTPProxy(t *testing.T) {
	var receivedURL string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		receivedURL = request.URL.String()
		fmt.Fprint(w, "proxied")
	}))
	defer proxy.Close()
	transport, err := NewTransport(&TTransportOptions{ProxyURL: proxy.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer transport.CloseIdleConnections()
	var text string
	var readErr error
	err = New().GET("http://example.invalid/resource?q=1").Transport(transport).
		OnSuccess(func(response *QResponse) { text, readErr = response.Text() }).Send()
	if err != nil || readErr != nil || text != "proxied" || receivedURL != "http://example.invalid/resource?q=1" {
		t.Fatalf("proxy failed: %q, %q, %v, %v", receivedURL, text, err, readErr)
	}
}

func TestTransportRejectsInvalidOptions(t *testing.T) {
	dir := t.TempDir()
	invalidCA := filepath.Join(dir, "invalid.pem")
	if err := os.WriteFile(invalidCA, []byte("invalid certificate"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, options := range []*TTransportOptions{
		{ProxyURL: "%"}, {ProxyURL: "localhost:8080"}, {ProxyURL: "ftp://localhost"},
		{CAFile: filepath.Join(dir, "missing.pem")}, {CAFile: invalidCA},
	} {
		if _, err := NewTransport(options); err == nil {
			t.Fatalf("expected invalid options to fail: %+v", options)
		}
	}
	var transport *QTransport
	transport.CloseIdleConnections()
	(&QTransport{}).CloseIdleConnections()
}

func TestTransportHTTPSProxyTunnel(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		fmt.Fprint(w, "secure proxy")
	}))
	defer server.Close()
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	connected := make(chan string, 1)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodConnect || request.Host != target.Host {
			http.Error(w, "unexpected proxy target", http.StatusBadRequest)
			return
		}
		upstream, err := net.DialTimeout("tcp", target.Host, 5*time.Second)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer upstream.Close()
		client, buffered, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer client.Close()
		if _, err := buffered.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
			t.Error(err)
			return
		}
		if err := buffered.Flush(); err != nil {
			t.Error(err)
			return
		}
		connected <- request.Host
		done := make(chan struct{})
		go func() {
			io.Copy(upstream, buffered)
			upstream.Close()
			close(done)
		}()
		io.Copy(client, upstream)
		client.Close()
		<-done
	}))
	defer proxy.Close()
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE", Bytes: server.Certificate().Raw,
	}), 0600); err != nil {
		t.Fatal(err)
	}
	transport, err := NewTransport(&TTransportOptions{ProxyURL: proxy.URL, CAFile: caFile})
	if err != nil {
		t.Fatal(err)
	}
	defer transport.CloseIdleConnections()
	var text string
	var readErr error
	err = New().GET(server.URL).Transport(transport).
		OnSuccess(func(response *QResponse) { text, readErr = response.Text() }).Send()
	if err != nil || readErr != nil || text != "secure proxy" {
		t.Fatalf("HTTPS tunnel failed: %q, %v, %v", text, err, readErr)
	}
	select {
	case <-connected:
	default:
		t.Fatal("expected a CONNECT request")
	}
}
