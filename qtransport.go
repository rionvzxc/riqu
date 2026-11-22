package riqu

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

// QTransport wraps a reusable HTTP transport.
type QTransport struct {
	transport *http.Transport
}
// NewTransport creates a transport using environment proxies and system CAs by default.
func NewTransport(options *TTransportOptions) (*QTransport, error) {
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok || defaultTransport == nil {
		return nil, fmt.Errorf("riqu: default HTTP transport cannot be cloned")
	}
	transport := defaultTransport.Clone()
	if options == nil {
		return &QTransport{transport: transport}, nil
	}
	if options.ProxyURL != "" {
		proxyURL, err := url.Parse(options.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("riqu: invalid proxy URL: %w", err)
		}
		if proxyURL.Hostname() == "" || (proxyURL.Scheme != "http" && proxyURL.Scheme != "https") {
			return nil, fmt.Errorf("riqu: proxy URL must include an http or https scheme and a host")
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	if options.CAFile != "" {
		certPEM, err := os.ReadFile(options.CAFile)
		if err != nil {
			return nil, fmt.Errorf("riqu: read CA file: %w", err)
		}
		roots, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("riqu: load system CA certificates: %w", err)
		}
		if !roots.AppendCertsFromPEM(certPEM) {
			return nil, fmt.Errorf("riqu: no valid CA certificates found in %q", options.CAFile)
		}
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.RootCAs = roots
	}
	return &QTransport{transport: transport}, nil
}
// CloseIdleConnections closes idle connections owned by the transport.
func (this *QTransport) CloseIdleConnections() {
	if this == nil || this.transport == nil {
		return
	}
	this.transport.CloseIdleConnections()
}
